package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Transport adapts an *http.Client to route every request through
// multisolverr instead of dialing the target directly. Plug it in as
// client.Transport — the standard library's cookie-jar handling keeps
// working transparently because Transport reflects multisolverr's returned
// cookies back as ordinary Set-Cookie response headers.
//
// Limitation: multisolverr solves requests with a headless-browser-backed
// solver when needed, so custom request headers (Authorization, CSRF
// headers, etc.) are not delivered to the target site — anti-bot solving
// impersonates a real browser rather than proxying arbitrary headers.
// Likewise, the reported response status reflects multisolverr's solved page
// and may not preserve a raw 3xx from a redirect-based login flow: the
// browser follows the redirect itself before multisolverr ever replies.
// RoundTrip compensates for that by reporting the page multisolverr actually
// landed on (solution.url) as resp.Request.URL, so callers that need to
// detect "we were redirected away from the URL we posted to" (the usual way
// a login flow signals success) can compare that against the URL they sent
// instead of relying on a status code that will never arrive.
type Transport struct {
	client *Client
}

// NewTransport returns a Transport backed by a multisolverr instance
// configured with cfg.
func NewTransport(cfg Config) *Transport {
	return &Transport{client: NewClient(cfg)}
}

// RoundTrip implements http.RoundTripper.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var postData string
	if req.Method == http.MethodPost && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body for multisolverr: %w", err)
		}
		req.Body.Close()
		postData = string(raw)
	}

	sr, err := t.client.Solve(req.Context(), req.Method, req.URL.String(), postData, req.Cookies())
	if err != nil {
		return nil, err
	}

	header := make(http.Header, len(sr.Solution.Headers)+len(sr.Solution.Cookies))
	for k, v := range sr.Solution.Headers {
		header.Set(k, v)
	}
	for _, ck := range sr.Solution.Cookies {
		header.Add("Set-Cookie", (&http.Cookie{Name: ck.Name, Value: ck.Value}).String())
	}

	status := sr.Solution.Status
	if status == 0 {
		status = http.StatusOK
	}

	respReq := req
	if sr.Solution.URL != "" {
		if landedURL, err := url.Parse(sr.Solution.URL); err == nil {
			clone := req.Clone(req.Context())
			clone.URL = landedURL
			respReq = clone
		}
	}

	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(sr.Solution.Response)),
		Request:    respReq,
	}, nil
}
