package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/jose/ratiodash/internal/domain"
)

const c411DefaultURL = "https://c411.org"

// C411Scraper fetches ratio stats from c411.org (AdonisJS / Nuxt.js stack).
//
// Authentication flow:
//  1. GET <baseURL>/login → captures the __csrf cookie and CSRF token from the page.
//  2. POST <baseURL>/api/auth/login with username + password + csrf-token header.
//  3. GET <baseURL>/api/auth/me → returns uploaded / downloaded / ratio.
//
// If the C411_URL environment variable is set it takes precedence over the
// tracker's stored URL, allowing a fixed deployment to skip per-tracker config.
//
// Credentials JSON:
//
//	{"username": "<username>", "password": "<password>"}
type C411Scraper struct {
	staticURL string
}

func NewC411Scraper() *C411Scraper {
	u := os.Getenv("C411_URL")
	if u == "" {
		u = c411DefaultURL
	}
	return &C411Scraper{staticURL: u}
}

func (s *C411Scraper) resolveURL() string {
	return s.staticURL
}

func (s *C411Scraper) Key() string { return "c411" }

func (s *C411Scraper) CredentialFields() []domain.CredentialField {
	return []domain.CredentialField{
		{Key: "username", Label: "Username", Type: "text", Required: true},
		{Key: "password", Label: "Password", Type: "password", Required: true},
	}
}

// c411MeResponse mirrors the relevant fields from GET /api/auth/me.
type c411MeResponse struct {
	User struct {
		Uploaded   int64   `json:"uploaded"`
		Downloaded int64   `json:"downloaded"`
		Ratio      float64 `json:"ratio"`
	} `json:"user"`
}

func (s *C411Scraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	creds, err := ParseCredentials(tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("c411: %w", err)
	}
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("c411: credentials must contain \"username\" and \"password\"")
	}

	baseURL := strings.TrimRight(s.resolveURL(), "/")

	// Fresh cookie jar per fetch keeps sessions clean.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 30 * time.Second, Jar: jar}

	csrfToken, err := c411FetchCSRF(ctx, client, baseURL)
	if err != nil {
		return nil, fmt.Errorf("c411: %w", err)
	}

	if err := c411Login(ctx, client, baseURL, creds.Username, creds.Password, csrfToken); err != nil {
		return nil, fmt.Errorf("c411: %w", err)
	}

	stats, err := c411FetchStats(ctx, client, baseURL)
	if err != nil {
		return nil, fmt.Errorf("c411: %w", err)
	}
	return stats, nil
}

// c411FetchCSRF GETs the login page to trigger the __csrf cookie and extract
// the CSRF token. It checks (in order): response header, HTML meta tag, hidden
// input, and finally any XSRF-TOKEN cookie set by the server.
func c411FetchCSRF(ctx context.Context, client *http.Client, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/login", nil)
	if err != nil {
		return "", fmt.Errorf("building login page request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching login page: %w", err)
	}
	defer resp.Body.Close()

	// Some AdonisJS setups send the token as a response header.
	for _, h := range []string{"X-Csrf-Token", "X-XSRF-Token"} {
		if v := resp.Header.Get(h); v != "" {
			return v, nil
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading login page: %w", err)
	}

	// Parse the HTML and look for the token in a meta tag or hidden input.
	root, err := ParseHTML(body)
	if err != nil {
		return "", err
	}

	var token string
	WalkHTML(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return true
		}
		switch n.Data {
		case "meta":
			if HTMLAttr(n, "name") == "csrf-token" {
				token = HTMLAttr(n, "content")
				return false
			}
		case "input":
			if HTMLAttr(n, "name") == "_csrf" {
				token = HTMLAttr(n, "value")
				return false
			}
		}
		return true
	})
	if token != "" {
		return token, nil
	}

	// Fallback: some apps set an XSRF-TOKEN cookie the client must echo back.
	siteURL, _ := url.Parse(baseURL)
	for _, cookie := range client.Jar.Cookies(siteURL) {
		if strings.EqualFold(cookie.Name, "XSRF-TOKEN") {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("CSRF token not found on login page — the site layout may have changed")
}

type c411LoginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func c411Login(ctx context.Context, client *http.Client, baseURL, username, password, csrfToken string) error {
	payload, _ := json.Marshal(c411LoginBody{Username: username, Password: password})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/auth/login", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("csrf-token", csrfToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (HTTP %d) — check your username and password", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := ParseJSON(body, &result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("authentication failed — check your username and password")
	}
	return nil
}

func c411FetchStats(ctx context.Context, client *http.Client, baseURL string) (*domain.TrackerStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/auth/me", nil)
	if err != nil {
		return nil, fmt.Errorf("building stats request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stats request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed (HTTP %d) — check your credentials", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading stats response: %w", err)
	}

	var me c411MeResponse
	if err := ParseJSON(body, &me); err != nil {
		return nil, err
	}
	return &domain.TrackerStats{
		Uploaded:   me.User.Uploaded,
		Downloaded: me.User.Downloaded,
		Ratio:      me.User.Ratio,
	}, nil
}
