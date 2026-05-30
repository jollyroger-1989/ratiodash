package scraper

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/jose/ratiodash/internal/domain"
)

const yggRebornDefaultURL = "https://www.yggreborn.org"

// YggRebornScraper fetches ratio stats from yggreborn.org.
//
// Flow:
//  1. GET /login → extract CSRF token from hidden form field.
//  2. POST /login with identifier (email), password, and CSRF token → session
//     cookies are set automatically via the cookie jar.
//  3. GET /account/ → parse upload/download/ratio from the stats grid.
//
// Credentials JSON:
//
//	{"username": "<email>", "password": "<password>"}
type YggRebornScraper struct {
	baseURL string
}

func NewYggRebornScraper() *YggRebornScraper {
	u := os.Getenv("YGGREBORN_URL")
	if u == "" {
		u = yggRebornDefaultURL
	}
	return &YggRebornScraper{baseURL: u}
}

func (s *YggRebornScraper) Key() string { return "yggreborn" }

func (s *YggRebornScraper) CredentialFields() []domain.CredentialField {
	return []domain.CredentialField{
		{Key: "username", Label: "Email / Username", Type: "text", Required: true},
		{Key: "password", Label: "Password", Type: "password", Required: true},
	}
}

func (s *YggRebornScraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	creds, err := ParseCredentials(tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("yggreborn: %w", err)
	}
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("yggreborn: credentials must contain \"username\" and \"password\" fields")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("yggreborn: creating cookie jar: %w", err)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		// Do not follow redirects — we handle the 302 after login explicitly.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	csrfToken, err := yggRebornGetCSRF(ctx, client, s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("yggreborn: %w", err)
	}

	if err := yggRebornLogin(ctx, client, creds.Username, creds.Password, csrfToken, s.baseURL); err != nil {
		return nil, fmt.Errorf("yggreborn: %w", err)
	}

	uploaded, downloaded, ratio, err := yggRebornFetchStats(ctx, client, s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("yggreborn: %w", err)
	}

	return &domain.TrackerStats{
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Ratio:      ratio,
	}, nil
}

// yggRebornGetCSRF performs a GET on the login page and extracts the hidden
// csrf_token field value.
func yggRebornGetCSRF(ctx context.Context, client *http.Client, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/login", nil)
	if err != nil {
		return "", fmt.Errorf("building login-page request: %w", err)
	}
	yggRebornSetBrowserHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching login page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading login page: %w", err)
	}

	token, err := yggRebornExtractCSRF(body)
	if err != nil {
		return "", err
	}
	return token, nil
}

// yggRebornExtractCSRF parses HTML and returns the value of
// <input name="csrf_token">.
func yggRebornExtractCSRF(body []byte) (string, error) {
	root, err := ParseHTML(body)
	if err != nil {
		return "", fmt.Errorf("parsing login page HTML: %w", err)
	}

	var token string
	WalkHTML(root, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "input" {
			return true
		}
		if HTMLAttr(n, "name") == "csrf_token" {
			token = HTMLAttr(n, "value")
			return false // stop
		}
		return true
	})

	if token == "" {
		return "", fmt.Errorf("csrf_token not found on login page")
	}
	return token, nil
}

// yggRebornLogin posts the login form and expects an HTTP 302 redirect on
// success.
func yggRebornLogin(ctx context.Context, client *http.Client, identifier, password, csrfToken, baseURL string) error {
	form := url.Values{}
	form.Set("csrf_token", csrfToken)
	form.Set("identifier", identifier)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		baseURL+"/login",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("building login request: %w", err)
	}
	yggRebornSetBrowserHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	switch resp.StatusCode {
	case http.StatusFound, http.StatusMovedPermanently, http.StatusSeeOther:
		// Success — cookie jar now holds __ygg_sess and remember_token.
		return nil
	case http.StatusOK:
		// Server returned 200 instead of a redirect — likely a login error page.
		return fmt.Errorf("authentication failed — check your credentials")
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed (HTTP %d) — check your credentials", resp.StatusCode)
	default:
		return fmt.Errorf("unexpected HTTP %d during login", resp.StatusCode)
	}
}

// yggRebornFetchStats fetches /account/ and parses the stats grid.
func yggRebornFetchStats(ctx context.Context, client *http.Client, baseURL string) (uploaded, downloaded int64, ratio float64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/account/", nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("building account request: %w", err)
	}
	yggRebornSetBrowserHeaders(req)

	// Re-enable redirects for GET requests (override the no-redirect policy).
	localClient := *client
	localClient.CheckRedirect = nil

	resp, err := localClient.Do(req)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetching account page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return 0, 0, 0, fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, fmt.Errorf("account page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reading account page: %w", err)
	}

	return yggRebornParseStats(body)
}

// yggRebornParseStats walks the account-page HTML and extracts the Upload,
// Download and Ratio values from the tracker-stats grid.
//
// The page renders a 2×2 grid where each cell contains a value div followed
// by a label div (e.g. "Upload", "Download", "Ratio"). We locate label text
// nodes and read the value from the preceding sibling element.
func yggRebornParseStats(body []byte) (uploaded, downloaded int64, ratio float64, err error) {
	root, err := ParseHTML(body)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parsing account page: %w", err)
	}

	stats := map[string]string{}

	WalkHTML(root, func(n *html.Node) bool {
		if n.Type != html.TextNode {
			return true
		}
		label := strings.TrimSpace(n.Data)
		if label != "Upload" && label != "Download" && label != "Ratio" {
			return true
		}

		// The text node is inside a stats-grid label div:
		//   <div class="text-[0.7rem] uppercase tracking-wider text-dark-400 mt-1">Upload</div>
		// We distinguish these from other "Upload" text on the page (e.g. the
		// quick-action button) by requiring the parent element to be a <div>
		// that contains "mt-1" in its class attribute.
		labelDiv := n.Parent
		if labelDiv == nil || labelDiv.Type != html.ElementNode || labelDiv.Data != "div" {
			return true
		}
		if !strings.Contains(HTMLAttr(labelDiv, "class"), "mt-1") {
			return true
		}

		valueDiv := prevElementSibling(labelDiv)
		if valueDiv == nil {
			return true
		}
		stats[label] = strings.TrimSpace(textContent(valueDiv))
		return true
	})

	if len(stats) < 3 {
		return 0, 0, 0, fmt.Errorf("could not find all stats on account page (found: %v)", stats)
	}

	uploaded, err = parseYggBytes(stats["Upload"])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parsing uploaded %q: %w", stats["Upload"], err)
	}
	downloaded, err = parseYggBytes(stats["Download"])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parsing downloaded %q: %w", stats["Download"], err)
	}
	ratio, err = parseYggRatio(stats["Ratio"])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("parsing ratio %q: %w", stats["Ratio"], err)
	}

	return uploaded, downloaded, ratio, nil
}

// parseYggBytes converts a YggReborn size string to raw bytes.
// YggReborn uses SI-style French suffixes: "o" (octet), "Ko", "Mo", "Go", "To", "Po".
// Values use a period as the decimal separator.
func parseYggBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0 o" {
		return 0, nil
	}

	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		// Bare number with no unit — treat as bytes.
		v, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("unexpected format %q", s)
		}
		return int64(math.Round(v)), nil
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, fmt.Errorf("parsing number in %q: %w", s, err)
	}

	// Binary multipliers: the app stores and displays bytes using 1024-based
	// powers (GiB). YggReborn labels sizes with French "Go/Mo/Ko" but treats
	// them as binary units (2^30, 2^20, …), matching this convention.
	multipliers := map[string]float64{
		"o":  1,
		"Ko": 1024,
		"Mo": 1024 * 1024,
		"Go": 1024 * 1024 * 1024,
		"To": 1024 * 1024 * 1024 * 1024,
		"Po": 1024 * 1024 * 1024 * 1024 * 1024,
	}

	unit := strings.TrimSpace(parts[1])
	mult, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q in %q", unit, s)
	}

	return int64(math.Round(value * mult)), nil
}

// parseYggRatio parses the ratio string from YggReborn.
// "∞" (infinite ratio, i.e. no downloads) is returned as 0 to indicate N/A.
func parseYggRatio(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "∞" || s == "" || s == "—" {
		return 0, nil
	}
	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing ratio: %w", err)
	}
	return r, nil
}

// prevElementSibling returns the first element-node sibling preceding n.
func prevElementSibling(n *html.Node) *html.Node {
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

// textContent returns the concatenated text content of a node and its children.
func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// yggRebornSetBrowserHeaders adds minimal browser-like headers to avoid being
// blocked by Cloudflare's bot-detection heuristics.
func yggRebornSetBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}
