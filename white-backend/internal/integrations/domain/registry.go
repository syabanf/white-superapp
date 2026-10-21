package domain

import (
	"time"
	"unicode/utf8"
)

// IntegrationKey names one credential pair the workspace can store.
type IntegrationKey string

const (
	KeyMeta       IntegrationKey = "META"
	KeyGoogle     IntegrationKey = "GOOGLE"
	KeyDataForSeo IntegrationKey = "DATAFORSEO"
	KeyApify      IntegrationKey = "APIFY"
	KeyOpenRouter IntegrationKey = "OPENROUTER"
)

// IntegrationKeys is the wizard order.
var IntegrationKeys = []IntegrationKey{KeyMeta, KeyGoogle, KeyDataForSeo, KeyApify, KeyOpenRouter}

type IntegrationDef struct {
	Key IntegrationKey
	// PublicEnv holds the non-secret half (app id, client id, login, model).
	PublicEnv string
	// SecretEnv holds the secret half.
	SecretEnv string
	// PublicRequired: both halves are needed for "configured". OpenRouter and
	// Apify need only the secret.
	PublicRequired bool
}

var Integrations = map[IntegrationKey]IntegrationDef{
	KeyMeta:       {Key: KeyMeta, PublicEnv: "META_APP_ID", SecretEnv: "META_APP_SECRET", PublicRequired: true},
	KeyGoogle:     {Key: KeyGoogle, PublicEnv: "GOOGLE_CLIENT_ID", SecretEnv: "GOOGLE_CLIENT_SECRET", PublicRequired: true},
	KeyDataForSeo: {Key: KeyDataForSeo, PublicEnv: "DATAFORSEO_LOGIN", SecretEnv: "DATAFORSEO_PASSWORD", PublicRequired: true},
	KeyApify:      {Key: KeyApify, PublicEnv: "APIFY_ACTORS", SecretEnv: "APIFY_TOKEN", PublicRequired: false},
	KeyOpenRouter: {Key: KeyOpenRouter, PublicEnv: "OPENROUTER_MODEL", SecretEnv: "OPENROUTER_API_KEY", PublicRequired: false},
}

func (k IntegrationKey) Valid() bool {
	_, ok := Integrations[k]
	return ok
}

// IsConfiguredPair: a secret is always required; the public half only when
// the definition says so.
func IsConfiguredPair(def IntegrationDef, publicID, secret string) bool {
	if secret == "" {
		return false
	}
	return !def.PublicRequired || publicID != ""
}

// MaskSecret keeps enough to recognise a key, never enough to use it:
// `sk-or-v1-abc…xyz` becomes `sk-o••••wxyz`.
func MaskSecret(secret string) string {
	if utf8.RuneCountInString(secret) <= 8 {
		return "••••"
	}
	r := []rune(secret)
	return string(r[:4]) + "••••" + string(r[len(r)-4:])
}

type CredentialSource string

const (
	SourceDB   CredentialSource = "db"
	SourceEnv  CredentialSource = "env"
	SourceNone CredentialSource = "none"
)

// IntegrationStatus is what the wizard shows for one key.
type IntegrationStatus struct {
	Key          IntegrationKey
	Configured   bool
	Source       CredentialSource
	PublicID     *string
	SecretMasked *string
	Meta         []byte
	VerifiedAt   *time.Time
}

// ProviderCredentials is the effective pair other services use.
type ProviderCredentials struct {
	Key        IntegrationKey
	Configured bool
	PublicID   string
	Secret     string
}

// Resolve merges a stored row (already decrypted) with the deployed env
// fallback: a DB value wins because the admin changing it in the UI is the
// more recent intent.
func Resolve(def IntegrationDef, row *IntegrationSetting, secret string, env func(string) string) (IntegrationStatus, ProviderCredentials) {
	fromDB := row != nil && ((row.PublicID != nil && *row.PublicID != "") || secret != "")
	publicID := ""
	if fromDB && row.PublicID != nil {
		publicID = *row.PublicID
	}
	if publicID == "" {
		publicID = env(def.PublicEnv)
	}
	if secret == "" {
		secret = env(def.SecretEnv)
	}
	configured := IsConfiguredPair(def, publicID, secret)
	source := SourceNone
	switch {
	case fromDB:
		source = SourceDB
	case configured:
		source = SourceEnv
	}
	st := IntegrationStatus{Key: def.Key, Configured: configured, Source: source}
	if publicID != "" {
		st.PublicID = &publicID
	}
	if secret != "" {
		masked := MaskSecret(secret)
		st.SecretMasked = &masked
	}
	if row != nil {
		st.Meta = row.Meta
		st.VerifiedAt = row.VerifiedAt
	}
	return st, ProviderCredentials{Key: def.Key, Configured: configured, PublicID: publicID, Secret: secret}
}

// VerifyResult is the outcome of one credential check.
type VerifyResult struct {
	OK      bool
	Message string
	Detail  map[string]string
}
