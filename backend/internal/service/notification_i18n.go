package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/jose/ratiodash/internal/domain"
)

var notificationBundle = newNotificationBundle()

func newNotificationBundle() *i18n.Bundle {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	bundle.AddMessages(language.English,
		&i18n.Message{ID: "alert.sync_error.title", Other: "[RatioDash] Sync failed: {{.TrackerName}}"},
		&i18n.Message{ID: "alert.sync_error.body", Other: "{{.Error}}"},
		&i18n.Message{ID: "alert.ratio_low.title", Other: "[RatioDash] Low ratio: {{.TrackerName}}"},
		&i18n.Message{ID: "alert.ratio_low.body", Other: "Ratio is {{.Ratio}} (threshold: {{.Threshold}})"},
		&i18n.Message{ID: "report.title", Other: "[RatioDash] 📊 {{.ReportName}}"},
		&i18n.Message{ID: "report.body.evolution_since", Other: "Evolution since {{.Timestamp}}"},
		&i18n.Message{ID: "report.body.first_report", Other: "First report — no previous baseline"},
		&i18n.Message{ID: "report.body.global", Other: "Global"},
		&i18n.Message{ID: "report.body.up_label", Other: "UP"},
		&i18n.Message{ID: "report.body.dl_label", Other: "DL"},
		&i18n.Message{ID: "report.body.ratio_label", Other: "Ratio"},
		&i18n.Message{ID: "report.body.no_data_yet", Other: "No data yet"},
		&i18n.Message{ID: "report.body.no_change", Other: "no change"},
		&i18n.Message{ID: "report.body.generated_at", Other: "Generated at {{.Timestamp}}"},
	)

	bundle.AddMessages(language.French,
		&i18n.Message{ID: "alert.sync_error.title", Other: "[RatioDash] Échec de synchro: {{.TrackerName}}"},
		&i18n.Message{ID: "alert.sync_error.body", Other: "{{.Error}}"},
		&i18n.Message{ID: "alert.ratio_low.title", Other: "[RatioDash] Ratio faible: {{.TrackerName}}"},
		&i18n.Message{ID: "alert.ratio_low.body", Other: "Le ratio est de {{.Ratio}} (seuil: {{.Threshold}})"},
		&i18n.Message{ID: "report.title", Other: "[RatioDash] 📊 {{.ReportName}}"},
		&i18n.Message{ID: "report.body.evolution_since", Other: "Évolution depuis {{.Timestamp}}"},
		&i18n.Message{ID: "report.body.first_report", Other: "Premier rapport — aucune base précédente"},
		&i18n.Message{ID: "report.body.global", Other: "Global"},
		&i18n.Message{ID: "report.body.up_label", Other: "UP"},
		&i18n.Message{ID: "report.body.dl_label", Other: "DL"},
		&i18n.Message{ID: "report.body.ratio_label", Other: "Ratio"},
		&i18n.Message{ID: "report.body.no_data_yet", Other: "Aucune donnée"},
		&i18n.Message{ID: "report.body.no_change", Other: "aucun changement"},
		&i18n.Message{ID: "report.body.generated_at", Other: "Généré à {{.Timestamp}}"},
	)

	return bundle
}

type notificationLocalizer struct {
	localizer *i18n.Localizer
	lang      string
}

func newNotificationLocalizer(lang string) notificationLocalizer {
	normalized := strings.ToLower(strings.TrimSpace(lang))
	if normalized != "fr" {
		normalized = "en"
	}
	return notificationLocalizer{
		localizer: i18n.NewLocalizer(notificationBundle, normalized, "en"),
		lang:      normalized,
	}
}

func (l notificationLocalizer) msg(id string, data map[string]any) string {
	text, err := l.localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: data,
	})
	if err != nil {
		return id
	}
	return text
}

func (l notificationLocalizer) formatTimestamp(ts time.Time) string {
	ts = ts.UTC()
	if l.lang == "fr" {
		return ts.Format("02/01/2006 15:04 UTC")
	}
	return ts.Format("2006-01-02 15:04 UTC")
}

func notificationLanguage(authRepo domain.AuthRepository) string {
	if authRepo == nil {
		return "en"
	}
	cred, err := authRepo.Find()
	if err != nil || cred == nil {
		return "en"
	}
	lang := strings.ToLower(strings.TrimSpace(cred.Language))
	if lang == "fr" {
		return "fr"
	}
	return "en"
}
