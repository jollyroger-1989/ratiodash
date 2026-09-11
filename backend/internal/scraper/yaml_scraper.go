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
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/jose/ratiodash/internal/domain"
	"github.com/jose/ratiodash/internal/proxy"
)

// YAMLScraper implements domain.TrackerScraper from a YAML definition file.
type YAMLScraper struct {
	def Definition
	// sessions persists reusable login sessions (cookies, auth captures)
	// between fetches. May be nil (e.g. in unit tests), in which case every
	// fetch performs a fresh login when the definition requires one.
	sessions domain.TrackerRepository
	// multisolverr provides the multisolverr proxy configuration used when a
	// tracker has UseMultisolverr set. May be nil (e.g. in unit tests), in
	// which case Fetch refuses to honor UseMultisolverr.
	multisolverr domain.MultisolverrConfigRepository
}

func (ys *YAMLScraper) logger() *logrus.Entry {
	return logrus.WithField("scraper", ys.def.ID)
}

// Key returns the scraper's unique identifier (the definition's id field).
func (ys *YAMLScraper) Key() string { return ys.def.ID }

// Deprecated reports whether the definition is marked deprecated.
func (ys *YAMLScraper) Deprecated() bool { return ys.def.Deprecated }

// CredentialFields returns the credential form fields declared in the definition.
func (ys *YAMLScraper) CredentialFields() []domain.CredentialField {
	fields := make([]domain.CredentialField, 0, len(ys.def.Settings))
	for _, s := range ys.def.Settings {
		fields = append(fields, domain.CredentialField{
			Key:      s.Name,
			Label:    s.Label,
			Type:     s.Type,
			Required: s.Required,
		})
	}
	return fields
}

// Fetch retrieves upload/download/ratio statistics for the given tracker.
func (ys *YAMLScraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	ys.logger().WithFields(logrus.Fields{
		"tracker_id":   tracker.ID,
		"tracker_name": tracker.Name,
	}).Info("scraper_fetching_stats")
	creds, err := parseCredMap(tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("%s: parsing credentials: %w", ys.def.ID, err)
	}

	for _, s := range ys.def.Settings {
		if s.Required && strings.TrimSpace(creds[s.Name]) == "" {
			return nil, fmt.Errorf("%s: required credential %q is missing", ys.def.ID, s.Name)
		}
	}

	sitelink, err := ys.resolveSitelink(creds)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ys.def.ID, err)
	}
	creds["sitelink"] = sitelink

	tctx := &TemplateContext{
		Config:   creds,
		Captures: make(map[string]string),
		Result:   make(map[string]string),
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("%s: creating cookie jar: %w", ys.def.ID, err)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	if tracker.UseMultisolverr {
		transport, err := ys.multisolverrTransport()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ys.def.ID, err)
		}
		client.Transport = transport
	}

	siteURL, err := url.Parse(sitelink)
	if err != nil {
		return nil, fmt.Errorf("%s: parsing sitelink: %w", ys.def.ID, err)
	}

	reused := false
	if ys.def.Login != nil {
		if session := decodeSession(tracker.SessionData); session != nil {
			applySession(jar, siteURL, session)
			for k, v := range session.Captures {
				tctx.Captures[k] = v
			}
			reused = true
			ys.logger().Debug("scraper_session_reused")
		} else {
			if err := ys.login(ctx, client, sitelink, tctx, tracker); err != nil {
				ys.logger().WithError(err).WithFields(logrus.Fields{
					"tracker_id":   tracker.ID,
					"tracker_name": tracker.Name,
					"sitelink":     sitelink,
				}).Warn("scraper_login_failed")
				return nil, fmt.Errorf("%s: login: %w", ys.def.ID, err)
			}
			ys.persistSession(ctx, tracker.ID, jar, siteURL, tctx.Captures)
		}
	}

	ys.logger().WithFields(logrus.Fields{
		"cookies": cookieNames(jar, siteURL),
		"reused":  reused,
	}).Info("scraper_stats_request_cookies")

	stats, err := ys.doStats(ctx, client, sitelink, tctx)
	if err != nil && reused {
		// The reused session may have expired: force a fresh login and retry once.
		ys.logger().WithError(err).Debug("scraper_stale_session_reauthenticating")
		jar, err = cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("%s: creating cookie jar: %w", ys.def.ID, err)
		}
		client.Jar = jar
		tctx.Captures = make(map[string]string)
		tctx.Result = make(map[string]string)

		if err := ys.login(ctx, client, sitelink, tctx, tracker); err != nil {
			ys.logger().WithError(err).WithFields(logrus.Fields{
				"tracker_id":   tracker.ID,
				"tracker_name": tracker.Name,
				"sitelink":     sitelink,
			}).Warn("scraper_login_failed")
			return nil, fmt.Errorf("%s: login: %w", ys.def.ID, err)
		}
		ys.persistSession(ctx, tracker.ID, jar, siteURL, tctx.Captures)

		stats, err = ys.doStats(ctx, client, sitelink, tctx)
	}
	if err != nil {
		ys.logger().WithError(err).WithFields(logrus.Fields{
			"tracker_id":   tracker.ID,
			"tracker_name": tracker.Name,
			"sitelink":     sitelink,
		}).Warn("scraper_stats_fetch_failed")
		return nil, fmt.Errorf("%s: %w", ys.def.ID, err)
	}
	return stats, nil
}

// multisolverrTransport builds an http.RoundTripper that routes requests
// through the configured multisolverr proxy. It errors out (rather than
// silently falling back to a direct request) when UseMultisolverr is set on
// the tracker but no multisolverr proxy is configured and enabled, since a
// silent fallback would just fail against the WAF again with a confusing
// error.
func (ys *YAMLScraper) multisolverrTransport() (http.RoundTripper, error) {
	if ys.multisolverr == nil {
		return nil, fmt.Errorf("multisolverr proxy is not available in this environment")
	}
	cfg, err := ys.multisolverr.Get()
	if err != nil {
		return nil, fmt.Errorf("loading multisolverr config: %w", err)
	}
	if !cfg.Enabled || cfg.BaseURL == "" {
		return nil, fmt.Errorf("multisolverr proxy is not configured or enabled — check Settings")
	}
	return proxy.NewTransport(proxy.Config{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}), nil
}

// login runs the definition's login flow and logs success.
func (ys *YAMLScraper) login(ctx context.Context, client *http.Client, sitelink string, tctx *TemplateContext, tracker domain.Tracker) error {
	if err := ys.doLogin(ctx, client, sitelink, ys.def.Login, tctx); err != nil {
		return err
	}
	ys.logger().WithFields(logrus.Fields{
		"tracker_id":   tracker.ID,
		"tracker_name": tracker.Name,
	}).Info("scraper_login_successful")
	return nil
}

// persistSession saves the current cookie jar and captured auth values so the
// next Fetch can skip logging in. It is a best-effort operation: a missing
// session store, a zero tracker ID (unsaved/test trackers), or a context that
// disables persistence (TrackerService.Test / TestByID) are silently skipped.
func (ys *YAMLScraper) persistSession(ctx context.Context, trackerID uint, jar *cookiejar.Jar, siteURL *url.URL, captures map[string]string) {
	if ys.sessions == nil || trackerID == 0 || domain.SessionPersistDisabled(ctx) {
		return
	}
	data, err := encodeSession(jar, siteURL, captures)
	if err != nil {
		ys.logger().WithError(err).Warn("scraper_session_encode_failed")
		return
	}
	if data == "" {
		return
	}
	if err := ys.sessions.UpdateSession(trackerID, data); err != nil {
		ys.logger().WithError(err).Warn("scraper_session_persist_failed")
	}
}

// ---------------------------------------------------------------------------
// Login helpers
// ---------------------------------------------------------------------------

func (ys *YAMLScraper) doLogin(ctx context.Context, client *http.Client, sitelink string, login *LoginDef, tctx *TemplateContext) error {
	switch strings.ToLower(login.Method) {
	case "form":
		return ys.doFormLogin(ctx, client, sitelink, login, tctx)
	case "json":
		return ys.doJSONLogin(ctx, client, sitelink, login, tctx)
	case "post":
		return ys.doPostLogin(ctx, client, sitelink, login, tctx)
	default:
		return fmt.Errorf("unsupported login method %q", login.Method)
	}
}

// doFormLogin handles logins where the CSRF or other dynamic values must be
// scraped from a GET page before the credentials are POSTed.
//
// Flow:
//  1. GET login.Path → extract selectorinputs (POST body) and selectorheaders (request headers)
//  2. Build POST body from inputs + selectorinputs
//  3. POST to submitpath (or path) with the configured content type
//  4. Check for login failure indicators
func (ys *YAMLScraper) doFormLogin(ctx context.Context, client *http.Client, sitelink string, login *LoginDef, tctx *TemplateContext) error {
	loginURL := joinURL(sitelink, login.Path)

	pageBody, err := ys.doGet(ctx, client, loginURL, nil)
	if err != nil {
		return fmt.Errorf("fetching login page: %w", err)
	}

	extraInputs := make(map[string]string)
	for k, sf := range login.SelectorInputs {
		val, err := extractHTML(pageBody, Field{Selector: sf.Selector, Attribute: sf.Attribute, Optional: true})
		if err != nil {
			ys.logger().WithError(err).WithFields(logrus.Fields{
				"selectorinput": k,
				"selector":      sf.Selector,
			}).Warn("scraper_login_selectorinput_extract_failed")
			return fmt.Errorf("selectorinputs[%s]: %w", k, err)
		}
		if strings.TrimSpace(val) == "" {
			ys.logger().WithFields(logrus.Fields{
				"selectorinput": k,
				"selector":      sf.Selector,
			}).Warn("scraper_login_selectorinput_empty")
		}
		extraInputs[k] = val
	}

	extraHeaders := make(map[string]string)
	for k, sf := range login.SelectorHeaders {
		val, err := extractHTML(pageBody, Field{Selector: sf.Selector, Attribute: sf.Attribute})
		if err != nil {
			ys.logger().WithError(err).WithFields(logrus.Fields{
				"selectorheader": k,
				"selector":       sf.Selector,
			}).Warn("scraper_login_selectorheader_extract_failed")
			return fmt.Errorf("selectorheaders[%s]: %w", k, err)
		}
		extraHeaders[k] = val
	}

	allInputs := make(map[string]interface{})
	for k, v := range login.Inputs {
		rendered, err := renderInputValue(v, *tctx)
		if err != nil {
			return fmt.Errorf("rendering input %q: %w", k, err)
		}
		allInputs[k] = rendered
	}
	for k, v := range extraInputs {
		allInputs[k] = v
	}

	submitPath := login.Path
	if login.SubmitPath != "" {
		submitPath = login.SubmitPath
	}
	submitURL := joinURL(sitelink, submitPath)

	contentType := "application/x-www-form-urlencoded"
	if strings.EqualFold(login.ContentType, "application/json") {
		contentType = "application/json"
	}

	bodyBytes, err := encodeInputs(allInputs, contentType)
	if err != nil {
		return fmt.Errorf("encoding login body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	// Do not follow redirects: a 3xx response from the login POST is a common
	// tracker pattern for signalling success (the server redirects to the home
	// page after a successful login).
	noRedirClient := *client
	noRedirClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := ys.doClientRequest(&noRedirClient, req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", submitURL, err)
	}
	defer resp.Body.Close()

	landedURL := submitURL
	if resp.Request != nil && resp.Request.URL != nil {
		landedURL = resp.Request.URL.String()
	}
	ys.logger().WithFields(logrus.Fields{
		"submit_url": submitURL,
		"landed_url": landedURL,
		"status":     resp.StatusCode,
	}).Info("scraper_login_response")

	// A redirect means login succeeded — the server is directing us elsewhere.
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil
	}
	// The multisolverr transport follows redirects with its own headless
	// browser and reports the final page's status (so the 3xx check above
	// never fires for it), but it does tell us the URL it landed on. Treat
	// landing anywhere other than submitURL as the same "redirected away"
	// success signal a raw 3xx would have given us directly.
	if landedURL != submitURL {
		return nil
	}
	if resp.StatusCode >= 400 {
		ys.logger().WithFields(logrus.Fields{
			"url":    submitURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_http_failed")
		return fmt.Errorf("login request returned HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := ys.checkLoginErrors(login, respBody); err != nil {
		ys.logger().WithError(err).WithFields(logrus.Fields{
			"url":    submitURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_validation_failed")
		return err
	}
	if len(login.Error) == 0 {
		// We never saw a redirect (raw 3xx or, via multisolverr, a landed URL
		// different from submitURL) and this definition has no login.error
		// indicators to check the response against, so there is no signal
		// left to tell a real login success apart from a silent failure that
		// just re-rendered the same page. Flag that loudly instead of
		// reporting success on faith.
		ys.logger().WithFields(logrus.Fields{
			"url":    submitURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_unverified")
	}

	return nil
}

// doJSONLogin POSTs a JSON body and extracts captures from the JSON response.
// This is the correct method for API-first trackers like Torr9.
func (ys *YAMLScraper) doJSONLogin(ctx context.Context, client *http.Client, sitelink string, login *LoginDef, tctx *TemplateContext) error {
	loginURL := joinURL(sitelink, login.Path)

	body := make(map[string]interface{})
	for k, v := range login.Inputs {
		rendered, err := renderInputValue(v, *tctx)
		if err != nil {
			return fmt.Errorf("rendering input %q: %w", k, err)
		}
		body[k] = rendered
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshalling login body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ys.doClientRequest(client, req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		ys.logger().WithFields(logrus.Fields{
			"url":    loginURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_http_failed")
		return fmt.Errorf("login request returned HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := ys.checkLoginErrors(login, respBody); err != nil {
		ys.logger().WithError(err).WithFields(logrus.Fields{
			"url":    loginURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_validation_failed")
		return err
	}

	// Extract captures from the JSON response body.
	for k, sf := range login.Captures {
		result := gjson.GetBytes(respBody, sf.Selector)
		if result.Exists() {
			tctx.Captures[k] = result.String()
			ys.logger().WithField("capture", k).Debug("scraper_capture_stored")
		}
	}

	return nil
}

// doPostLogin POSTs form-encoded data without a preceding GET.
func (ys *YAMLScraper) doPostLogin(ctx context.Context, client *http.Client, sitelink string, login *LoginDef, tctx *TemplateContext) error {
	loginURL := joinURL(sitelink, login.Path)

	form := url.Values{}
	for k, v := range login.Inputs {
		rendered, err := renderInputValue(v, *tctx)
		if err != nil {
			return fmt.Errorf("rendering input %q: %w", k, err)
		}
		form.Set(k, fmt.Sprint(rendered))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ys.doClientRequest(client, req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		ys.logger().WithFields(logrus.Fields{
			"url":    loginURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_http_failed")
		return fmt.Errorf("login request returned HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := ys.checkLoginErrors(login, respBody); err != nil {
		ys.logger().WithError(err).WithFields(logrus.Fields{
			"url":    loginURL,
			"status": resp.StatusCode,
		}).Warn("scraper_login_validation_failed")
		return err
	}

	return nil
}

// checkLoginErrors inspects the login response body for error indicators.
// For JSON responses, it checks gjson paths against expected failure values.
// For HTML responses, it checks whether any error CSS selector matches.
func (ys *YAMLScraper) checkLoginErrors(login *LoginDef, body []byte) error {
	responseIsJSON := login.Response != nil && strings.EqualFold(login.Response.Type, "json")

	for _, errDef := range login.Error {
		if responseIsJSON {
			val := gjson.GetBytes(body, errDef.Selector)
			if errDef.Value != "" && val.String() == errDef.Value {
				ys.logger().WithFields(logrus.Fields{
					"selector": errDef.Selector,
					"value":    errDef.Value,
				}).Warn("scraper_login_error_indicator_matched")
				return fmt.Errorf("authentication failed")
			}
		} else {
			match, _ := extractHTML(body, Field{Selector: errDef.Selector, Optional: true})
			if match != "" {
				ys.logger().WithField("selector", errDef.Selector).Warn("scraper_login_error_indicator_matched")
				return fmt.Errorf("authentication failed: error indicator %q found on login page", errDef.Selector)
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stats helpers
// ---------------------------------------------------------------------------

func (ys *YAMLScraper) doStats(ctx context.Context, client *http.Client, sitelink string, tctx *TemplateContext) (*domain.TrackerStats, error) {
	stats := ys.def.Stats

	statsPath, err := renderTemplate(stats.Path, *tctx)
	if err != nil {
		return nil, fmt.Errorf("rendering stats path: %w", err)
	}
	statsURL := joinURL(sitelink, statsPath)

	headers := make(map[string]string, len(stats.Headers))
	for k, v := range stats.Headers {
		rendered, err := renderTemplate(v, *tctx)
		if err != nil {
			return nil, fmt.Errorf("rendering header %q: %w", k, err)
		}
		headers[k] = rendered
	}

	body, err := ys.doGet(ctx, client, statsURL, headers)
	if err != nil {
		return nil, err
	}

	responseType := "html"
	if stats.Response != nil && stats.Response.Type != "" {
		responseType = strings.ToLower(stats.Response.Type)
	}

	result := make(map[string]string)

	for _, fe := range stats.Fields.Entries() {
		name := fe.Name
		field := fe.Field

		var rawValue string

		switch {
		case field.Text != "":
			tctx.Result = result
			rawValue, err = renderTemplate(field.Text, *tctx)
			if err != nil && !field.Optional {
				return nil, fmt.Errorf("field %q template: %w", name, err)
			}

		case field.Selector != "":
			switch responseType {
			case "json":
				rawValue, err = extractJSON(body, field)
			default:
				rawValue, err = extractHTML(body, field)
			}
			if err != nil {
				if field.Optional {
					rawValue = field.Default
					err = nil
				} else {
					ys.logger().WithFields(logrus.Fields{
						"field":         name,
						"selector":      field.Selector,
						"response_type": responseType,
						"body_length":   len(body),
						"body_preview":  previewBody(body, 800),
					}).Warn("scraper_stats_field_extract_failed")
					return nil, fmt.Errorf("field %q: %w", name, err)
				}
			}
		}

		if rawValue == "" && field.Default != "" {
			rawValue = field.Default
		}

		rawValue, err = applyFilters(rawValue, field.Filters)
		if err != nil {
			return nil, fmt.Errorf("field %q filter: %w", name, err)
		}

		result[name] = rawValue
	}

	uploaded, _ := strconv.ParseInt(result["uploaded"], 10, 64)
	downloaded, _ := strconv.ParseInt(result["downloaded"], 10, 64)

	var ratio float64
	if rs := result["ratio"]; rs != "" {
		ratio, _ = strconv.ParseFloat(rs, 64)
	}
	if ratio == 0 && downloaded > 0 && result["ratio"] == "" {
		ratio = float64(uploaded) / float64(downloaded)
	}

	ys.logger().WithFields(logrus.Fields{
		"uploaded":   uploaded,
		"downloaded": downloaded,
		"ratio":      ratio,
	}).Info("scraper_stats_parsed")
	return &domain.TrackerStats{
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Ratio:      ratio,
	}, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// doClientRequest runs req via client and logs "METHOD url -> status" on a
// single line for every completed HTTP request the scraper makes.
func (ys *YAMLScraper) doClientRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	ys.logger().WithFields(logrus.Fields{
		"method": req.Method,
		"url":    req.URL.String(),
		"status": resp.StatusCode,
	}).Infof("%s %s -> %d", req.Method, req.URL.String(), resp.StatusCode)
	return resp, nil
}

// cookieNames returns the names (never values — these can be session
// secrets) of the cookies the jar currently holds for siteURL, for logging
// what a stats/login request is actually authenticated with.
func cookieNames(jar *cookiejar.Jar, siteURL *url.URL) []string {
	cookies := jar.Cookies(siteURL)
	names := make([]string, 0, len(cookies))
	for _, c := range cookies {
		names = append(names, c.Name)
	}
	return names
}

// previewBody returns up to n bytes of body with whitespace collapsed, for
// logging a compact snippet of a response that failed field extraction —
// enough to tell a login/challenge/empty page apart from the expected markup
// without dumping the whole (potentially large) response into the logs.
func previewBody(body []byte, n int) string {
	s := strings.Join(strings.Fields(string(body)), " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func (ys *YAMLScraper) doGet(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := ys.doClientRequest(client, req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s returned HTTP %d", rawURL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ---------------------------------------------------------------------------
// Session persistence
// ---------------------------------------------------------------------------

// storedSession is the JSON shape persisted in Tracker.SessionData.
type storedSession struct {
	Cookies  []storedCookie    `json:"cookies,omitempty"`
	Captures map[string]string `json:"captures,omitempty"`
}

type storedCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// decodeSession parses a tracker's stored session blob. It returns nil if the
// blob is empty, invalid, or carries no reusable state.
func decodeSession(raw string) *storedSession {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var s storedSession
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil
	}
	if len(s.Cookies) == 0 && len(s.Captures) == 0 {
		return nil
	}
	return &s
}

// applySession seeds the cookie jar with a previously stored session's cookies
// for siteURL. Captures are applied separately by the caller.
func applySession(jar *cookiejar.Jar, siteURL *url.URL, s *storedSession) {
	if len(s.Cookies) == 0 {
		return
	}
	cookies := make([]*http.Cookie, 0, len(s.Cookies))
	for _, c := range s.Cookies {
		cookies = append(cookies, &http.Cookie{Name: c.Name, Value: c.Value})
	}
	jar.SetCookies(siteURL, cookies)
}

// encodeSession serialises the jar's cookies for siteURL plus any auth
// captures into the JSON blob stored on the tracker. Returns "" if there is
// nothing worth persisting.
func encodeSession(jar *cookiejar.Jar, siteURL *url.URL, captures map[string]string) (string, error) {
	s := storedSession{Captures: captures}
	for _, c := range jar.Cookies(siteURL) {
		s.Cookies = append(s.Cookies, storedCookie{Name: c.Name, Value: c.Value})
	}
	if len(s.Cookies) == 0 && len(s.Captures) == 0 {
		return "", nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ---------------------------------------------------------------------------
// URL and credential helpers
// ---------------------------------------------------------------------------

// resolveSitelink determines the base URL for requests.
// It prefers the user-supplied "url" credential and falls back to the first
// link in the definition's links array.
func (ys *YAMLScraper) resolveSitelink(creds map[string]string) (string, error) {
	if u := strings.TrimRight(creds["url"], "/"); u != "" {
		if err := validateScraperURL(u); err != nil {
			return "", fmt.Errorf("invalid url credential: %w", err)
		}
		return u, nil
	}
	if len(ys.def.Links) > 0 {
		return strings.TrimRight(ys.def.Links[0], "/"), nil
	}
	return "", fmt.Errorf("no base URL: set the url credential or add links to the definition")
}

// validateScraperURL returns an error if rawURL does not use the http or https
// scheme, preventing SSRF via file://, gopher://, etc.
func validateScraperURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme %q is not allowed; use http or https", u.Scheme)
	}
	return nil
}

// joinURL appends path to base, ensuring exactly one slash between them.
// If path is already an absolute URL it is returned unchanged.
func joinURL(base, path string) string {
	if path == "" {
		return base
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

// parseCredMap decodes a tracker's credentials JSON string into a flat string map.
func parseCredMap(raw string) (map[string]string, error) {
	m := make(map[string]string)
	if raw == "" || raw == "{}" {
		return m, nil
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// renderInputValue renders a login input value. Strings are treated as Go
// templates; non-string values (booleans, numbers) are returned as-is so that
// JSON marshalling preserves their native types.
func renderInputValue(v interface{}, tctx TemplateContext) (interface{}, error) {
	s, ok := v.(string)
	if !ok {
		return v, nil // preserve native type (bool, int, float)
	}
	return renderTemplate(s, tctx)
}

// encodeInputs serialises the input map for the configured content type.
func encodeInputs(inputs map[string]interface{}, contentType string) ([]byte, error) {
	if strings.EqualFold(contentType, "application/json") {
		return json.Marshal(inputs)
	}
	form := url.Values{}
	for k, v := range inputs {
		form.Set(k, fmt.Sprint(v))
	}
	return []byte(form.Encode()), nil
}
