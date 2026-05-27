package scraper

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jose/ratiodash/internal/domain"
)

// Unit3DScraper fetches ratio stats from any UNIT3D-based private tracker.
//
// UNIT3D is open-source tracker software used by many private trackers
// (e.g. Aither, BluRay.World, FearNoPeer, …).
//
// Authentication:
//
//	{"token": "<api_token>"}
//
// The API token is found in the user's profile settings on the tracker.
//
// The scraper calls GET <site.URL>/api/user with an "Authorization: Bearer"
// header and parses the JSON response. Uploaded and downloaded are returned
// as human-readable strings (e.g. "1.23 TiB"); this scraper converts them
// to raw bytes for storage.
type Unit3DScraper struct {
	BaseScraper
}

func NewUnit3DScraper() *Unit3DScraper {
	return &Unit3DScraper{BaseScraper: NewBase()}
}

func (s *Unit3DScraper) Key() string { return "unit3d" }

func (s *Unit3DScraper) CredentialFields() []domain.CredentialField {
	return []domain.CredentialField{
		{Key: "url", Label: "Tracker URL", Type: "text", Required: true},
		{Key: "token", Label: "API Token", Type: "password", Required: true},
	}
}

// unit3dUserResponse mirrors the fields returned by UNIT3D's UserResource.
type unit3dUserResponse struct {
	Username   string `json:"username"`
	Uploaded   string `json:"uploaded"`
	Downloaded string `json:"downloaded"`
	Ratio      string `json:"ratio"`
}

func (s *Unit3DScraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	creds, err := ParseCredentials(tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("unit3d: %w", err)
	}
	if creds.Token == "" {
		return nil, fmt.Errorf("unit3d: credentials must contain a \"token\" field")
	}
	if creds.URL == "" {
		return nil, fmt.Errorf("unit3d: credentials must contain a \"url\" field")
	}

	apiURL := strings.TrimRight(creds.URL, "/") + "/api/user"
	body, err := s.DoRequest(ctx, "GET", apiURL, creds)
	if err != nil {
		return nil, fmt.Errorf("unit3d: %w", err)
	}

	var resp unit3dUserResponse
	if err := ParseJSON(body, &resp); err != nil {
		return nil, fmt.Errorf("unit3d: %w", err)
	}

	uploaded, err := parseUnit3DBytes(resp.Uploaded)
	if err != nil {
		return nil, fmt.Errorf("unit3d: parsing uploaded %q: %w", resp.Uploaded, err)
	}
	downloaded, err := parseUnit3DBytes(resp.Downloaded)
	if err != nil {
		return nil, fmt.Errorf("unit3d: parsing downloaded %q: %w", resp.Downloaded, err)
	}
	ratio, err := strconv.ParseFloat(strings.TrimSpace(resp.Ratio), 64)
	if err != nil {
		return nil, fmt.Errorf("unit3d: parsing ratio %q: %w", resp.Ratio, err)
	}

	return &domain.TrackerStats{
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Ratio:      ratio,
	}, nil
}

// parseUnit3DBytes converts a UNIT3D formatted size string (e.g. "1.23 TiB")
// to a raw byte count. UNIT3D uses binary prefixes (KiB, MiB, GiB, TiB, PiB).
func parseUnit3DBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	// Split on the first space: "1.23 TiB" → ["1.23", "TiB"]
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected format %q", s)
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, fmt.Errorf("parsing number in %q: %w", s, err)
	}

	multipliers := map[string]float64{
		"B":   1,
		"KiB": 1024,
		"MiB": 1024 * 1024,
		"GiB": 1024 * 1024 * 1024,
		"TiB": 1024 * 1024 * 1024 * 1024,
		"PiB": 1024 * 1024 * 1024 * 1024 * 1024,
	}

	unit := strings.TrimSpace(parts[1])
	mult, ok := multipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q in %q", unit, s)
	}

	return int64(math.Round(value * mult)), nil
}
