package scraper

import "gopkg.in/yaml.v3"

// Definition is the parsed representation of a Cardigann-inspired YAML scraper
// definition. Each .yml file in the scrapers directory corresponds to one
// Definition and produces one YAMLScraper at runtime.
type Definition struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Language    string    `yaml:"language"`
	Type        string    `yaml:"type"`
	Encoding    string    `yaml:"encoding"`
	Links       []string  `yaml:"links"`
	Settings    []Setting `yaml:"settings"`
	Login       *LoginDef `yaml:"login"`
	Stats       StatsDef  `yaml:"stats"`
}

// Setting describes a credential field that must be provided when registering a
// tracker site that uses this definition.
type Setting struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"` // text | password
	Label    string `yaml:"label"`
	Required bool   `yaml:"required"`
}

// LoginDef describes how to authenticate against a tracker before fetching stats.
type LoginDef struct {
	// Method controls the authentication strategy:
	//   form – GET path, extract dynamic fields, POST to submitpath (or path)
	//   json – POST JSON body directly to path
	//   post – POST form-encoded body to path
	Method      string `yaml:"method"`
	Path        string `yaml:"path"`
	SubmitPath  string `yaml:"submitpath"`  // alternate POST target (form method)
	ContentType string `yaml:"contenttype"` // override; "application/json" for JSON POST

	// Inputs are rendered as Go templates and added to the POST body.
	Inputs map[string]interface{} `yaml:"inputs"`

	// SelectorInputs extracts values from the login page HTML and adds them to
	// the POST body (e.g. hidden CSRF tokens).
	SelectorInputs map[string]SelectorField `yaml:"selectorinputs"`

	// SelectorHeaders extracts values from the login page HTML and adds them as
	// HTTP request headers on the POST (e.g. c411's X-Csrf-Token header).
	SelectorHeaders map[string]SelectorField `yaml:"selectorheaders"`

	Response *ResponseDef             `yaml:"response"`
	Captures map[string]SelectorField `yaml:"captures"` // store response values into tctx.Captures
	Error    []ErrorDef               `yaml:"error"`    // login failure indicators
	Test     *TestDef                 `yaml:"test"`     // post-login validation
}

// SelectorField extracts a string value from an HTML or JSON response.
// For HTML: Selector is a CSS selector, Attribute is optional (defaults to
// element text content).
// For JSON: Selector is a gjson dot-path.
type SelectorField struct {
	Selector  string   `yaml:"selector"`
	Attribute string   `yaml:"attribute"`
	Filters   []Filter `yaml:"filters"`
}

// ErrorDef describes a login failure indicator.
// For HTML responses: Selector is a CSS selector; if it matches, login failed.
// For JSON responses: Selector is a JSON path; if its value equals Value, login failed.
type ErrorDef struct {
	Selector string `yaml:"selector"`
	Value    string `yaml:"value"` // JSON error check: matches if gjson(selector) == Value
}

// TestDef is an optional post-login verification: after login, GET Path and
// verify that Selector matches (or that the response is successful).
type TestDef struct {
	Path     string `yaml:"path"`
	Selector string `yaml:"selector"`
}

// ResponseDef specifies the expected response content type.
type ResponseDef struct {
	Type string `yaml:"type"` // html (default) | json
}

// StatsDef describes how to fetch and extract upload/download/ratio statistics.
type StatsDef struct {
	Path     string            `yaml:"path"`
	Method   string            `yaml:"method"`  // get (default) | post
	Headers  map[string]string `yaml:"headers"` // Go template values
	Response *ResponseDef      `yaml:"response"`
	Fields   OrderedFields     `yaml:"fields"`
}

// Field describes how to extract a single value from an HTML or JSON response.
type Field struct {
	// Selector is a CSS selector (HTML) or gjson dot-path (JSON).
	Selector  string `yaml:"selector"`
	Attribute string `yaml:"attribute"` // HTML only: attribute name (default: text content)
	// Match controls which element is used when multiple elements match the
	// selector. Accepted values: "first" (default) and "last". Use "last" when
	// the selector can match both an outer ancestor container and an inner
	// element — "last" picks the innermost (deepest in document order) match.
	Match string `yaml:"match"`
	// Text is a Go template that produces the value directly, bypassing Selector.
	// It can reference .Result to use previously-computed field values.
	Text     string   `yaml:"text"`
	Optional bool     `yaml:"optional"`
	Default  string   `yaml:"default"`
	Filters  []Filter `yaml:"filters"`
}

// Filter applies a named transformation to a string value.
type Filter struct {
	Name string      `yaml:"name"`
	Args interface{} `yaml:"args"`
}

// FieldEntry is one entry in an ordered map of fields.
type FieldEntry struct {
	Name  string
	Field Field
}

// OrderedFields preserves the YAML insertion order of a mapping node so that
// template fields (which reference .Result.*) are evaluated after the fields
// they depend on.
type OrderedFields struct {
	entries []FieldEntry
}

// Entries returns the fields in their original YAML definition order.
func (of OrderedFields) Entries() []FieldEntry {
	return of.entries
}

// UnmarshalYAML decodes a YAML mapping while preserving key insertion order.
func (of *OrderedFields) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		var f Field
		if err := valNode.Decode(&f); err != nil {
			return err
		}
		of.entries = append(of.entries, FieldEntry{Name: keyNode.Value, Field: f})
	}
	return nil
}
