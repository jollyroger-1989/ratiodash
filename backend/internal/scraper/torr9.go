package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jose/ratiodash/internal/domain"
)

const torr9DefaultURL = "https://api.torr9.net"

// Torr9Scraper fetches ratio stats from a Torr9-based tracker via its REST API.
// Flow: POST /auth/login → JWT token → GET /users/me.
// If the TORR9_URL environment variable is set it takes precedence over the
// tracker's stored URL, allowing a fixed deployment to skip per-tracker config.
type Torr9Scraper struct {
	staticURL string
}

func NewTorr9Scraper() *Torr9Scraper {
	u := os.Getenv("TORR9_URL")
	if u == "" {
		u = torr9DefaultURL
	}
	return &Torr9Scraper{staticURL: u}
}

func (s *Torr9Scraper) resolveURL() string {
	return s.staticURL
}

func (s *Torr9Scraper) Key() string { return "torr9" }

func (s *Torr9Scraper) CredentialFields() []domain.CredentialField {
	return []domain.CredentialField{
		{Key: "username", Label: "Username", Type: "text", Required: true},
		{Key: "password", Label: "Password", Type: "password", Required: true},
	}
}

func (s *Torr9Scraper) Fetch(ctx context.Context, tracker domain.Tracker) (*domain.TrackerStats, error) {
	creds, err := ParseCredentials(tracker.Credentials)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	baseURL := s.resolveURL()

	token, err := torr9Login(ctx, baseURL, creds.Username, creds.Password)
	if err != nil {
		return nil, err
	}

	uploaded, downloaded, err := torr9FetchStats(ctx, baseURL, token)
	if err != nil {
		return nil, err
	}

	var ratio float64
	if downloaded > 0 {
		ratio = float64(uploaded) / float64(downloaded)
	}

	return &domain.TrackerStats{
		Uploaded:   uploaded,
		Downloaded: downloaded,
		Ratio:      ratio,
	}, nil
}

type torr9LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

type torr9LoginResponse struct {
	Token string `json:"token"`
}

func torr9Login(ctx context.Context, baseURL, username, password string) (string, error) {
	body, err := json.Marshal(torr9LoginRequest{
		Username:   username,
		Password:   password,
		RememberMe: true,
	})
	if err != nil {
		return "", fmt.Errorf("encoding login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("authentication failed (HTTP %d) — check your credentials", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login request returned HTTP %d", resp.StatusCode)
	}

	var loginResp torr9LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decoding login response: %w", err)
	}
	if loginResp.Token == "" {
		return "", fmt.Errorf("login succeeded but no token returned — check credentials")
	}
	return loginResp.Token, nil
}

type torr9UserResponse struct {
	TotalUploadedBytes   int64 `json:"total_uploaded_bytes"`
	TotalDownloadedBytes int64 `json:"total_downloaded_bytes"`
	BonusUploaded        int64 `json:"bonus_uploaded"`
	BonusDownloaded      int64 `json:"bonus_downloaded"`
}

func torr9FetchStats(ctx context.Context, baseURL, token string) (uploaded, downloaded int64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/users/me", nil)
	if err != nil {
		return 0, 0, fmt.Errorf("building stats request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("stats request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return 0, 0, fmt.Errorf("authentication failed (HTTP %d) — check your credentials", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("stats request returned HTTP %d", resp.StatusCode)
	}

	var user torr9UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return 0, 0, fmt.Errorf("decoding stats response: %w", err)
	}
	// Torr9 includes bonus bytes in its displayed ratio.
	return user.TotalUploadedBytes + user.BonusUploaded, user.TotalDownloadedBytes + user.BonusDownloaded, nil
}
