package scraper

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// TemplateContext holds all variables available to Go templates embedded in
// YAML definition values (paths, headers, inputs, field texts).
type TemplateContext struct {
	// Config holds the tracker's parsed credential map. The special key
	// "sitelink" is injected automatically and resolves to the base URL.
	Config map[string]string

	// Captures holds values extracted from the login response (e.g. auth tokens).
	Captures map[string]string

	// Result holds field values computed so far during stats extraction.
	// Template-based fields may reference previously computed fields here.
	Result map[string]string
}

var templateFuncs = template.FuncMap{
	// isum parses two int64 strings and returns their sum as a decimal string.
	"isum": func(a, b string) string {
		x, _ := strconv.ParseInt(strings.TrimSpace(a), 10, 64)
		y, _ := strconv.ParseInt(strings.TrimSpace(b), 10, 64)
		return strconv.FormatInt(x+y, 10)
	},

	// fratio computes uploaded/downloaded as a float64 string.
	// Returns "0" when downloaded is zero to avoid division by zero.
	"fratio": func(uploaded, downloaded string) string {
		u, _ := strconv.ParseInt(strings.TrimSpace(uploaded), 10, 64)
		d, _ := strconv.ParseInt(strings.TrimSpace(downloaded), 10, 64)
		if d == 0 {
			return "0"
		}
		return strconv.FormatFloat(float64(u)/float64(d), 'f', -1, 64)
	},

	// re_replace applies a regular expression substitution.
	"re_replace": func(s, pattern, replacement string) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return s
		}
		return re.ReplaceAllString(s, replacement)
	},
}

// renderTemplate applies Go text/template to tmpl with ctx as the data object.
// If tmpl contains no template syntax it is returned unchanged (fast path).
func renderTemplate(tmpl string, ctx TemplateContext) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}
	t, err := template.New("").Funcs(templateFuncs).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", tmpl, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("executing template %q: %w", tmpl, err)
	}
	return buf.String(), nil
}
