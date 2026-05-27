package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/html"
)

// Credentials is the common JSON structure stored in Site.Credentials.
// All fields are optional; scrapers should only use what they need.
type Credentials struct {
	URL      string `json:"url"`
	Cookie   string `json:"cookie"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
	// Token is sent as "Authorization: Bearer <token>" (used by UNIT3D and similar).
	Token   string            `json:"token"`
	Headers map[string]string `json:"headers"`
}

// ParseCredentials decodes a site's credential JSON into a Credentials struct.
func ParseCredentials(raw string) (Credentials, error) {
	var c Credentials
	if raw == "" || raw == "{}" {
		return c, nil
	}
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return c, fmt.Errorf("parsing credentials: %w", err)
	}
	return c, nil
}

// BaseScraper provides authenticated HTTP fetching shared by all site adapters.
// Embed it in a concrete scraper and call Fetch, then parse the body with
// ParseHTML, ParseJSON, or ParseXML.
//
// Example (JSON API):
//
//	type MyTracker struct{ scraper.BaseScraper }
//
//	func NewMyTracker() *MyTracker { return &MyTracker{scraper.NewBase()} }
//	func (s *MyTracker) Key() string { return "mytracker" }
//
//	//    {"url": "https://tracker.example.com", "token": "<api_token>"}
//
//	func (s *MyTracker) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
//	    body, err := s.Get(ctx, tracker.Credentials)
//	    if err != nil { return nil, err }
//
//	    var resp struct {
//	        Uploaded   int64   `json:"uploaded"`
//	        Downloaded int64   `json:"downloaded"`
//	        Ratio      float64 `json:"ratio"`
//	    }
//	    if err := scraper.ParseJSON(body, &resp); err != nil { return nil, err }
//
//	    return &domain.TrackerStats{
//	        Uploaded: resp.Uploaded, Downloaded: resp.Downloaded, Ratio: resp.Ratio,
//	    }, nil
//	}
type BaseScraper struct {
	client *http.Client
}

// NewBase returns a BaseScraper with a 30-second timeout.
func NewBase() BaseScraper {
	return BaseScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Get performs an authenticated GET using the URL stored in the credentials JSON
// and returns the raw body.
// It sets Cookie, Authorization (Bearer), X-API-Key, and any extra headers
// found in the credentials.
func (b *BaseScraper) Get(ctx context.Context, rawCredentials string) ([]byte, error) {
	creds, err := ParseCredentials(rawCredentials)
	if err != nil {
		return nil, err
	}
	return b.DoRequest(ctx, http.MethodGet, creds.URL, creds)
}

// DoRequest is the low-level helper for scrapers that need custom methods or URLs.
func (b *BaseScraper) DoRequest(ctx context.Context, method, rawURL string, creds Credentials) ([]byte, error) {
	// Validate scheme to prevent SSRF via file://, gopher://, etc.
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid or disallowed URL scheme in credentials (must be http or https): %q", rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", rawURL, err)
	}

	if creds.Cookie != "" {
		req.Header.Set("Cookie", creds.Cookie)
	}
	if creds.Token != "" {
		req.Header.Set("Authorization", "Bearer "+creds.Token)
	}
	if creds.APIKey != "" {
		req.Header.Set("X-Api-Key", creds.APIKey)
	}
	for k, v := range creds.Headers {
		req.Header.Set(k, v)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch {
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return nil, fmt.Errorf("authentication failed (HTTP %d) — check your credentials", resp.StatusCode)
		case resp.StatusCode == http.StatusNotFound:
			return nil, fmt.Errorf("endpoint not found (HTTP 404) — verify the tracker URL")
		case resp.StatusCode >= 500:
			return nil, fmt.Errorf("tracker server error (HTTP %d)", resp.StatusCode)
		default:
			return nil, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, rawURL)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", rawURL, err)
	}
	return body, nil
}

// --- Parse helpers ---

// ParseJSON decodes a JSON body into dst (must be a pointer).
func ParseJSON(body []byte, dst any) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '<' {
		return fmt.Errorf("tracker returned an HTML page instead of JSON — check the URL is correct")
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("unexpected JSON from tracker: %w", err)
	}
	return nil
}

// ParseXML decodes an XML body into dst (must be a pointer).
func ParseXML(body []byte, dst any) error {
	if err := xml.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("parsing XML response: %w", err)
	}
	return nil
}

// ParseHTML parses an HTML body and returns the root node.
// Use golang.org/x/net/html to traverse the tree.
//
// Example:
//
//	root, err := scraper.ParseHTML(body)
//	scraper.WalkHTML(root, func(n *html.Node) bool {
//	    // inspect n
//	    return true
//	})
func ParseHTML(body []byte) (*html.Node, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML response: %w", err)
	}
	return root, nil
}

// WalkHTML performs a depth-first traversal of an HTML node tree.
// fn receives each node; return false to stop the walk early.
func WalkHTML(n *html.Node, fn func(*html.Node) bool) {
	if n == nil {
		return
	}
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		WalkHTML(c, fn)
	}
}

// HTMLAttr returns the value of an attribute on a node, or "" if absent.
func HTMLAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
