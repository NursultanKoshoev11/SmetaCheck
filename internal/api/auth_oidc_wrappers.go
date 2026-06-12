package api

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
)

type bufferedAuthResponse struct {
	header      http.Header
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func newBufferedAuthResponse() *bufferedAuthResponse {
	return &bufferedAuthResponse{header: make(http.Header), status: http.StatusOK}
}

func (response *bufferedAuthResponse) Header() http.Header {
	return response.header
}

func (response *bufferedAuthResponse) WriteHeader(status int) {
	if response.wroteHeader {
		return
	}
	response.status = status
	response.wroteHeader = true
}

func (response *bufferedAuthResponse) Write(data []byte) (int, error) {
	if !response.wroteHeader {
		response.WriteHeader(http.StatusOK)
	}
	return response.body.Write(data)
}

func AuthGoogleBeginSecure(w http.ResponseWriter, r *http.Request) {
	beginOIDCWithBrowserState(w, r, "google", AuthGoogleBegin)
}

func AuthTelegramBeginSecure(w http.ResponseWriter, r *http.Request) {
	beginOIDCWithBrowserState(w, r, "telegram", AuthTelegramBegin)
}

func AuthGoogleCallbackSecure(w http.ResponseWriter, r *http.Request) {
	finishOIDCWithBrowserState(w, r, "google", AuthGoogleCallback)
}

func AuthTelegramCallbackSecure(w http.ResponseWriter, r *http.Request) {
	finishOIDCWithBrowserState(w, r, "telegram", AuthTelegramCallback)
}

func beginOIDCWithBrowserState(w http.ResponseWriter, r *http.Request, provider string, handler http.HandlerFunc) {
	buffered := newBufferedAuthResponse()
	handler(buffered, r)

	location := strings.TrimSpace(buffered.header.Get("Location"))
	if location != "" {
		if parsed, err := url.Parse(location); err == nil {
			state := strings.TrimSpace(parsed.Query().Get("state"))
			if state != "" {
				setOAuthStateCookie(w, provider, state)
			}
		}
	}
	copyBufferedAuthResponse(w, buffered)
}

func finishOIDCWithBrowserState(w http.ResponseWriter, r *http.Request, provider string, handler http.HandlerFunc) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if !validateAndClearOAuthStateCookie(w, r, provider, state) {
		redirectOAuthFailure(w, r, provider, "invalid_state", nil)
		return
	}
	handler(w, r)
}

func copyBufferedAuthResponse(w http.ResponseWriter, buffered *bufferedAuthResponse) {
	for key, values := range buffered.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(buffered.status)
	if buffered.body.Len() > 0 {
		_, _ = w.Write(buffered.body.Bytes())
	}
}
