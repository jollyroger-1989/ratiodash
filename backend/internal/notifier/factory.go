package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jose/ratiodash/internal/domain"
)

// Factory creates domain.Notifier instances from persisted NotifierConfig data.
// It is the bridge between the generic NotifierConfig CRUD layer and the
// concrete transport implementations.
type Factory struct{}

// NewFactory returns a Factory that implements domain.NotifierBuilder.
func NewFactory() domain.NotifierBuilder {
	return Factory{}
}

// Build instantiates a live Notifier from a type key and JSON config blob.
func (Factory) Build(cfgType, cfgJSON string) (domain.Notifier, error) {
	if cfgJSON == "" {
		cfgJSON = "{}"
	}

	switch cfgType {
	case "ntfy":
		var cfg struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse ntfy config: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf(`ntfy: missing required field "url"`)
		}
		return &ntfyNotifier{url: cfg.URL, token: cfg.Token, client: &http.Client{}}, nil

	case "email":
		var cfg struct {
			Host     string `json:"host"`
			Port     string `json:"port"`
			From     string `json:"from"`
			To       string `json:"to"`
			Username string `json:"username"`
			Password string `json:"password"`
			TLSMode  string `json:"tls_mode"`
		}
		if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
			return nil, fmt.Errorf("parse email config: %w", err)
		}
		if strings.TrimSpace(cfg.Host) == "" {
			return nil, fmt.Errorf(`email: missing required field "host"`)
		}
		if strings.TrimSpace(cfg.Port) == "" {
			return nil, fmt.Errorf(`email: missing required field "port"`)
		}
		port, err := strconv.Atoi(strings.TrimSpace(cfg.Port))
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("email: invalid port %q", cfg.Port)
		}
		if strings.TrimSpace(cfg.From) == "" {
			return nil, fmt.Errorf(`email: missing required field "from"`)
		}
		if strings.TrimSpace(cfg.To) == "" {
			return nil, fmt.Errorf(`email: missing required field "to"`)
		}

		n, err := newEmailNotifier(emailConfig{
			host:     strings.TrimSpace(cfg.Host),
			port:     port,
			from:     strings.TrimSpace(cfg.From),
			to:       strings.TrimSpace(cfg.To),
			username: strings.TrimSpace(cfg.Username),
			password: cfg.Password,
			tlsMode:  strings.TrimSpace(cfg.TLSMode),
		})
		if err != nil {
			return nil, fmt.Errorf("email: %w", err)
		}
		return n, nil

	default:
		return nil, fmt.Errorf("unknown notifier type: %q", cfgType)
	}
}
