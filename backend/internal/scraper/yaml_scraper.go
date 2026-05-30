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
)

// YAMLScraper implements domain.TrackerScraper from a YAML definition file.
type YAMLScraper struct {
	def Definition
}

func (ys *YAMLScraper) logger() *logrus.Entry {
	return logrus.WithField("scraper", ys.def.ID)
}

// Key returns the scraper's unique identifier (the definition's id field).
func (ys *YAMLScraper) Key() string { return ys.def.ID }

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

	if ys.def.Login != nil {
		if err := ys.doLogin(ctx, client, sitelink, ys.def.Login, tctx); err != nil {
			ys.logger().WithError(err).WithFields(logrus.Fields{
				"tracker_id":   tracker.ID,
				"tracker_name": tracker.Name,
				"sitelink":     sitelink,
			}).Warn("scraper_login_failed")
			return nil, fmt.Errorf("%s: login: %w", ys.def.ID, err)
		}
		ys.logger().Info("scraper_login_successful")
	}

	stats, err := ys.doStats(ctx, client, sitelink, tctx)
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

	ys.logger().WithField("url", loginURL).Debug("scraper_login_page_get")
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

	resp, err := noRedirClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", submitURL, err)
	}
	defer resp.Body.Close()

	ys.logger().WithFields(logrus.Fields{
		"url":    submitURL,
		"status": resp.StatusCode,
	}).Debug("scraper_login_submit_response")

	// A redirect means login succeeded — the server is directing us elsewhere.
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
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

	return nil
}

// doJSONLogin POSTs a JSON body and extracts captures from the JSON response.
// This is the correct method for API-first trackers like Torr9.
func (ys *YAMLScraper) doJSONLogin(ctx context.Context, client *http.Client, sitelink string, login *LoginDef, tctx *TemplateContext) error {
	loginURL := joinURL(sitelink, login.Path)
	ys.logger().WithField("url", loginURL).Debug("scraper_json_login_post")

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

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", loginURL, err)
	}
	defer resp.Body.Close()

	ys.logger().WithFields(logrus.Fields{
		"url":    loginURL,
		"status": resp.StatusCode,
	}).Debug("scraper_json_login_response")

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
	ys.logger().WithField("url", loginURL).Debug("scraper_form_login_post")

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

	resp, err := client.Do(req)
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

	ys.logger().WithField("url", statsURL).Debug("scraper_stats_get")
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

func (ys *YAMLScraper) doGet(ctx context.Context, client *http.Client, rawURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
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
