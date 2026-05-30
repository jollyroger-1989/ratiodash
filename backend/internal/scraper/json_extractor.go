package scraper

import (
	"fmt"

	"github.com/tidwall/gjson"
)

// extractJSON extracts a field value from a JSON body using a gjson dot-path selector.
func extractJSON(body []byte, field Field) (string, error) {
	if field.Selector == "" {
		return "", nil
	}

	result := gjson.GetBytes(body, field.Selector)
	if !result.Exists() {
		if field.Optional {
			return field.Default, nil
		}
		return "", fmt.Errorf("JSON path %q not found", field.Selector)
	}

	return result.String(), nil
}
