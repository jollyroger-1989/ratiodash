// export_test.go exposes internal symbols for use in external (_test) tests.
// This file is compiled only during testing.
package scraper

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/jose/ratiodash/internal/domain"
)

// ParseBytesForTest exposes parseBytes for unit tests.
func ParseBytesForTest(s string) int64 {
	n, _ := parseBytes(s)
	return n
}

// ParseFloatForTest exposes parseFloatValue for unit tests.
func ParseFloatForTest(s string) float64 {
	n, _ := parseFloatValue(s)
	return n
}

// RenderTemplateForTest exposes renderTemplate for unit tests.
func RenderTemplateForTest(tmpl string, ctx TemplateContext) (string, error) {
	return renderTemplate(tmpl, ctx)
}

// LoadFromYAMLForTest parses a YAML string and returns a ready-to-use YAMLScraper.
func LoadFromYAMLForTest(t testing.TB, raw string) *YAMLScraper {
	t.Helper()
	var def Definition
	if err := yaml.Unmarshal([]byte(raw), &def); err != nil {
		t.Fatalf("LoadFromYAMLForTest: parsing YAML: %v", err)
	}
	return &YAMLScraper{def: def}
}

// LoadFromYAMLWithSessionsForTest is LoadFromYAMLForTest but also wires a
// session store, for tests exercising session reuse/renewal.
func LoadFromYAMLWithSessionsForTest(t testing.TB, raw string, sessions domain.TrackerRepository) *YAMLScraper {
	t.Helper()
	var def Definition
	if err := yaml.Unmarshal([]byte(raw), &def); err != nil {
		t.Fatalf("LoadFromYAMLWithSessionsForTest: parsing YAML: %v", err)
	}
	return &YAMLScraper{def: def, sessions: sessions}
}

// LoadSingleForTest loads one YAML definition file and returns a YAMLScraper.
func LoadSingleForTest(t testing.TB, path string) *YAMLScraper {
	t.Helper()
	s, err := loadFile(path, nil, nil)
	if err != nil {
		t.Fatalf("LoadSingleForTest: %v", err)
	}
	return s
}

// LoadFromYAMLWithMultisolverrForTest is LoadFromYAMLForTest but also wires a
// multisolverr config repository, for tests exercising the proxy transport.
func LoadFromYAMLWithMultisolverrForTest(t testing.TB, raw string, multisolverr domain.MultisolverrConfigRepository) *YAMLScraper {
	t.Helper()
	var def Definition
	if err := yaml.Unmarshal([]byte(raw), &def); err != nil {
		t.Fatalf("LoadFromYAMLWithMultisolverrForTest: parsing YAML: %v", err)
	}
	return &YAMLScraper{def: def, multisolverr: multisolverr}
}
