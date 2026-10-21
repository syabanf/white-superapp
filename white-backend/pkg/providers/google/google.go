// Package google holds the OAuth endpoints, scopes and error mapping shared
// by the Search Console, Analytics and token-refresh code paths.
package google

import (
	"encoding/json"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	AuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL    = "https://oauth2.googleapis.com/token"
	UserinfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	GSCBase     = "https://www.googleapis.com/webmasters/v3"
	GA4Base     = "https://analyticsdata.googleapis.com/v1beta"
)

var Scopes = []string{
	"https://www.googleapis.com/auth/webmasters.readonly",
	"https://www.googleapis.com/auth/analytics.readonly",
	"openid",
	"email",
}

type apiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
	// OAuth token endpoint shape.
	OAuthError string `json:"error_description"`
}

// MapError covers both the OAuth token endpoint ("invalid_grant") and the
// REST APIs (PERMISSION_DENIED, RESOURCE_EXHAUSTED).
func MapError(provider string, status int, body []byte) *apperr.Error {
	text := string(body)
	var ae apiError
	_ = json.Unmarshal(body, &ae)
	msg := ae.Error.Message
	if msg == "" {
		msg = httpx.Describe(&httpx.Response{Status: status, Body: body})
	}
	var c apperr.Code
	switch {
	case status == 429 || strings.Contains(text, "RESOURCE_EXHAUSTED"):
		c = apperr.RateLimit
	case status == 401 || strings.Contains(text, "invalid_grant"):
		c = apperr.TokenExpired
	case status == 403 || strings.Contains(text, "access_denied") || strings.Contains(text, "PERMISSION_DENIED"):
		c = apperr.Permission
	case status == 404:
		c = apperr.NotFound
	default:
		c = apperr.Unknown
	}
	return apperr.FromProvider(provider, c, msg, nil)
}
