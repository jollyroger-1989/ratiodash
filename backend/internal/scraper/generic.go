package scraper

import (
	"context"
	"fmt"

	"github.com/jose/ratiodash/internal/domain"
)

// GenericScraper is a cookie-authenticated template scraper.
//
// To add support for a new torrent site:
//
//  1. Copy this file and rename the struct (e.g. PassThePopcornScraper).
//  2. Change Key() to return the site's unique key (e.g. "ptp").
//  3. Implement Fetch() using Get() + ParseHTML / ParseJSON / ParseXML.
//  4. Register the new constructor in scraper.Module with the `group:"scrapers"` tag.
type GenericScraper struct {
	BaseScraper
}

func NewGenericScraper() *GenericScraper {
	return &GenericScraper{BaseScraper: NewBase()}
}

func (s *GenericScraper) Key() string { return "generic" }

func (s *GenericScraper) CredentialFields() []domain.CredentialField { return nil }

// Fetch makes an authenticated GET request and extracts upload/download/ratio.
// Replace the TODO section with real parsing logic for the target site.
//
// HTML example:
//
//	root, err := ParseHTML(body)
//	WalkHTML(root, func(n *html.Node) bool { … })
//
// JSON example:
//
//	var resp struct{ Uploaded int64; Downloaded int64; Ratio float64 }
//	ParseJSON(body, &resp)
//
// XML example:
//
//	var resp struct{ XMLName xml.Name; Uploaded int64 `xml:"uploaded"` }
//	ParseXML(body, &resp)
func (s *GenericScraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	body, err := s.Get(ctx, tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("generic scraper: %w", err)
	}

	_ = body // TODO: parse body and fill Uploaded, Downloaded, Ratio

	return &domain.TrackerStats{
		Uploaded:   0,
		Downloaded: 0,
		Ratio:      0,
	}, nil
}
