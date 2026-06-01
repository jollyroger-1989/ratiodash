package handler

import (
	"strings"

	"github.com/jose/ratiodash/internal/domain"
)

func notifierTypesLanguage(authRepo domain.AuthRepository) string {
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
