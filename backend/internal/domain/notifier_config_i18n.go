package domain

import (
	"encoding/json"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var notifierTypeBundle = newNotifierTypeBundle()

func newNotifierTypeBundle() *i18n.Bundle {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	bundle.AddMessages(language.English,
		&i18n.Message{ID: "notifier.type.ntfy", Other: "ntfy"},
		&i18n.Message{ID: "notifier.field.url", Other: "Topic URL"},
		&i18n.Message{ID: "notifier.field.token", Other: "Access Token"},
		&i18n.Message{ID: "notifier.type.email", Other: "Email (SMTP)"},
		&i18n.Message{ID: "notifier.field.host", Other: "SMTP Host"},
		&i18n.Message{ID: "notifier.field.port", Other: "SMTP Port"},
		&i18n.Message{ID: "notifier.field.from", Other: "From Address"},
		&i18n.Message{ID: "notifier.field.to", Other: "To Address(es)"},
		&i18n.Message{ID: "notifier.field.username", Other: "Username"},
		&i18n.Message{ID: "notifier.field.password", Other: "Password"},
		&i18n.Message{ID: "notifier.field.tls_mode", Other: "TLS Mode"},
	)

	bundle.AddMessages(language.French,
		&i18n.Message{ID: "notifier.type.ntfy", Other: "ntfy"},
		&i18n.Message{ID: "notifier.field.url", Other: "URL du sujet"},
		&i18n.Message{ID: "notifier.field.token", Other: "Jeton d'accès"},
		&i18n.Message{ID: "notifier.type.email", Other: "Email (SMTP)"},
		&i18n.Message{ID: "notifier.field.host", Other: "Hôte SMTP"},
		&i18n.Message{ID: "notifier.field.port", Other: "Port SMTP"},
		&i18n.Message{ID: "notifier.field.from", Other: "Adresse expéditeur"},
		&i18n.Message{ID: "notifier.field.to", Other: "Adresse(s) destinataire"},
		&i18n.Message{ID: "notifier.field.username", Other: "Nom d'utilisateur"},
		&i18n.Message{ID: "notifier.field.password", Other: "Mot de passe"},
		&i18n.Message{ID: "notifier.field.tls_mode", Other: "Mode TLS"},
	)

	return bundle
}

type notifierTypeLocalizer struct {
	localizer *i18n.Localizer
}

func newNotifierTypeLocalizer(lang string) notifierTypeLocalizer {
	normalized := strings.ToLower(strings.TrimSpace(lang))
	if normalized != "fr" {
		normalized = "en"
	}
	return notifierTypeLocalizer{
		localizer: i18n.NewLocalizer(notifierTypeBundle, normalized, "en"),
	}
}

func (l notifierTypeLocalizer) msg(id, fallback string) string {
	text, err := l.localizer.Localize(&i18n.LocalizeConfig{MessageID: id})
	if err != nil {
		return fallback
	}
	return text
}

// LocalizeNotifierTypes returns a copy of notifier type metadata with
// labels translated to the requested language. Unknown languages fallback to en.
func LocalizeNotifierTypes(types []NotifierTypeInfo, lang string) []NotifierTypeInfo {
	loc := newNotifierTypeLocalizer(lang)
	out := make([]NotifierTypeInfo, 0, len(types))
	for _, t := range types {
		copyType := NotifierTypeInfo{
			Key:          t.Key,
			Label:        loc.msg("notifier.type."+t.Key, t.Label),
			ConfigFields: make([]NotifierConfigField, 0, len(t.ConfigFields)),
		}
		for _, f := range t.ConfigFields {
			copyType.ConfigFields = append(copyType.ConfigFields, NotifierConfigField{
				Key:      f.Key,
				Label:    loc.msg("notifier.field."+f.Key, f.Label),
				Type:     f.Type,
				Required: f.Required,
				Options:  append([]string(nil), f.Options...),
			})
		}
		out = append(out, copyType)
	}
	return out
}
