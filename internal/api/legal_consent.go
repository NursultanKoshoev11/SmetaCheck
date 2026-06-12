package api

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	privacyPolicyVersion = "2026-06-12"
	termsOfServiceVersion = "2026-06-12"
)

type uploadConsent struct {
	PrivacyVersion string
	TermsVersion   string
}

func requireUploadConsent(r *http.Request) (uploadConsent, error) {
	truthy := func(value string) bool {
		value = strings.ToLower(strings.TrimSpace(value))
		return value == "true" || value == "1" || value == "yes" || value == "on"
	}
	if !truthy(r.FormValue("document_rights_confirmed")) {
		return uploadConsent{}, fmt.Errorf("confirm that you have the right to upload these documents")
	}
	if !truthy(r.FormValue("processing_consent")) {
		return uploadConsent{}, fmt.Errorf("consent to document processing is required")
	}
	if !truthy(r.FormValue("ai_processing_consent")) {
		return uploadConsent{}, fmt.Errorf("consent to the selected AI processing mode is required")
	}
	privacyVersion := strings.TrimSpace(r.FormValue("privacy_version"))
	termsVersion := strings.TrimSpace(r.FormValue("terms_version"))
	if privacyVersion != privacyPolicyVersion || termsVersion != termsOfServiceVersion {
		return uploadConsent{}, fmt.Errorf("legal terms have changed; reload the page and review them again")
	}
	return uploadConsent{PrivacyVersion: privacyVersion, TermsVersion: termsVersion}, nil
}

func recordUploadConsent(r *http.Request, userID, resourceType, resourceID, provider string, consent uploadConsent) {
	_ = writeAuditLog(r.Context(), r, userID, "legal.upload_consent.accepted", resourceType, resourceID, map[string]any{
		"privacy_version": consent.PrivacyVersion,
		"terms_version": consent.TermsVersion,
		"provider": provider,
		"document_rights_confirmed": true,
		"processing_consent": true,
		"ai_processing_consent": true,
	})
}
