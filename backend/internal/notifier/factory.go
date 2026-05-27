package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	default:
		return nil, fmt.Errorf("unknown notifier type: %q", cfgType)
	}
}
