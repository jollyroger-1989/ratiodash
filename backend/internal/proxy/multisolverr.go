// Package proxy implements a client for multisolverr
// (https://github.com/jollyroger-1989/multisolverr), a proxy that resolves
// Cloudflare/WAF challenges via a FlareSolverr-compatible API. Routing a
// tracker's scraper requests through it lets RatioDash scrape trackers that
// would otherwise reject direct requests.
package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds the connection details needed to reach a multisolverr instance.
type Config struct {
	BaseURL string
	APIKey  string
	// Timeout bounds how long multisolverr is allowed to spend solving a
	// single request (sent as maxTimeout). Defaults to 60s.
	Timeout time.Duration
}

// ValidateBaseURL returns an error unless rawURL is an absolute http(s) URL.
func ValidateBaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("proxy URL must be an absolute http:// or https:// URL")
	}
	return nil
}

// --- FlareSolverr-compatible wire protocol ---

type solveRequest struct {
	Cmd        string            `json:"cmd"`
	URL        string            `json:"url,omitempty"`
	MaxTimeout int               `json:"maxTimeout,omitempty"`
	PostData   string            `json:"postData,omitempty"`
	Cookies    []solveCookie     `json:"cookies,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type solveCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type solveResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string            `json:"url"`
		Status   int               `json:"status"`
		Headers  map[string]string `json:"headers"`
		Response string            `json:"response"`
		Cookies  []solveCookie     `json:"cookies"`
	} `json:"solution"`
}

// Client talks to a multisolverr instance's FlareSolverr-compatible API.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient returns a Client. The underlying HTTP client's own timeout is
// kept generously above cfg.Timeout so multisolverr has room to return a
// proper "could not solve" response instead of the call being cut off first.
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout + 15*time.Second},
	}
}

func (c *Client) endpoint() string {
	return strings.TrimRight(c.cfg.BaseURL, "/") + "/v1"
}

func (c *Client) call(ctx context.Context, req solveRequest) (*solveResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding multisolverr request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("X-Api-Key", c.cfg.APIKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling multisolverr at %s: %w", c.cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading multisolverr response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("multisolverr returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var sr solveResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("parsing multisolverr response: %w", err)
	}
	if sr.Status != "ok" {
		msg := sr.Message
		if msg == "" {
			msg = "unknown error"
		}
		return nil, fmt.Errorf("multisolverr could not solve the request: %s", msg)
	}
	return &sr, nil
}

// Solve resolves method+targetURL through multisolverr and returns the
// solved page. Only GET and POST are supported — the only methods the
// scraper engine issues.
func (c *Client) Solve(ctx context.Context, method, targetURL, postData string, cookies []*http.Cookie) (*solveResponse, error) {
	var cmd string
	switch method {
	case http.MethodGet:
		cmd = "request.get"
	case http.MethodPost:
		cmd = "request.post"
	default:
		return nil, fmt.Errorf("multisolverr proxy does not support HTTP method %s", method)
	}

	req := solveRequest{
		Cmd:        cmd,
		URL:        targetURL,
		MaxTimeout: int(c.cfg.Timeout / time.Millisecond),
		PostData:   postData,
	}
	for _, ck := range cookies {
		req.Cookies = append(req.Cookies, solveCookie{Name: ck.Name, Value: ck.Value})
	}
	return c.call(ctx, req)
}

// Ping checks that a multisolverr instance is reachable at the configured
// BaseURL. It deliberately does not invoke the solving pipeline (a "sessions"
// or "request" command) since that could consume a paid solver's quota
// (e.g. Scrappey) just to test connectivity — a plain HTTP round trip to the
// host is enough to confirm it is up and answering.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.cfg.BaseURL, "/")+"/", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to multisolverr at %s: %w", c.cfg.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("multisolverr at %s returned HTTP %d", c.cfg.BaseURL, resp.StatusCode)
	}
	return nil
}
