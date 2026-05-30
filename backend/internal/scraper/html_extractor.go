package scraper

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// extractHTML extracts a field value from an HTML body using a CSS selector.
func extractHTML(body []byte, field Field) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}
	return extractHTMLFromDoc(doc, field)
}

// extractHTMLFromDoc extracts a value from an already-parsed goquery Document.
func extractHTMLFromDoc(doc *goquery.Document, field Field) (string, error) {
	if field.Selector == "" {
		return "", nil
	}

	sel := doc.Find(field.Selector)
	if sel.Length() == 0 {
		if field.Optional {
			return field.Default, nil
		}
		return "", fmt.Errorf("selector %q matched no elements", field.Selector)
	}

	node := sel.First()
	if strings.EqualFold(field.Match, "last") {
		node = sel.Last()
	}

	var value string
	if field.Attribute != "" {
		attr, exists := node.Attr(field.Attribute)
		if !exists {
			if field.Optional {
				return field.Default, nil
			}
			return "", fmt.Errorf("attribute %q not found for selector %q", field.Attribute, field.Selector)
		}
		value = attr
	} else {
		value = strings.TrimSpace(node.Text())
	}

	return value, nil
}
