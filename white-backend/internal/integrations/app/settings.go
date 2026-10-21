// Package app holds the integrations use cases: workspace credentials
// (setup wizard) and per-client OAuth connections.
package app

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/crypto"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

const (
	msgMetaAppID      = "App ID"
	msgGoogleClientID = "Client ID"
	msgDfsLogin       = "Login (email akun)"
)

// ProviderVerifier runs the one-call credential checks (adapter/providers/verify).
type ProviderVerifier interface {
	MetaApp(ctx context.Context, appID, appSecret string) domain.VerifyResult
	GoogleClient(ctx context.Context, clientID, clientSecret string) domain.VerifyResult
	DataForSeo(ctx context.Context, login, password string) domain.VerifyResult
	OpenRouter(ctx context.Context, apiKey string) domain.VerifyResult
	Apify(ctx context.Context, token string) domain.VerifyResult
}

type Settings struct {
	store    domain.Store
	cipher   *crypto.Cipher
	verifier ProviderVerifier
	bus      events.Publisher
	clk      clock.Clock
	// env reads the deployed fallback for a credential half.
	env func(string) string
}

func NewSettings(store domain.Store, cipher *crypto.Cipher, verifier ProviderVerifier, bus events.Publisher, clk clock.Clock) *Settings {
	return &Settings{store: store, cipher: cipher, verifier: verifier, bus: bus, clk: clk, env: func(k string) string { return config.Env(k, "") }}
}

// WithEnv replaces the environment lookup (tests).
func (s *Settings) WithEnv(env func(string) string) *Settings {
	s.env = env
	return s
}

// resolve returns the wizard status and the effective credentials of one
// key. A stored secret that no longer decrypts counts as absent.
func (s *Settings) resolve(key domain.IntegrationKey, row *domain.IntegrationSetting) (domain.IntegrationStatus, domain.ProviderCredentials) {
	secret := ""
	if row != nil && row.SecretEnc != nil && *row.SecretEnc != "" {
		if plain, err := s.cipher.Decrypt(*row.SecretEnc); err == nil {
			secret = plain
		}
	}
	return domain.Resolve(domain.Integrations[key], row, secret, s.env)
}

type IntegrationOverview struct {
	Statuses   []domain.IntegrationStatus
	MetaReview domain.MetaReviewState
}

func (s *Settings) ListStatuses(ctx context.Context) (*IntegrationOverview, error) {
	rows, err := s.store.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	byKey := map[domain.IntegrationKey]*domain.IntegrationSetting{}
	for i := range rows {
		byKey[rows[i].Provider] = &rows[i]
	}
	out := &IntegrationOverview{}
	for _, key := range domain.IntegrationKeys {
		st, _ := s.resolve(key, byKey[key])
		out.Statuses = append(out.Statuses, st)
		if key == domain.KeyMeta {
			out.MetaReview = domain.ParseMetaReview(st.Meta)
		}
	}
	return out, nil
}

// Credentials returns the effective pair for a provider (DB wins over env).
func (s *Settings) Credentials(ctx context.Context, key domain.IntegrationKey) (domain.ProviderCredentials, error) {
	if !key.Valid() {
		return domain.ProviderCredentials{}, apperr.New(apperr.Validation, "")
	}
	row, err := s.store.GetSetting(ctx, key)
	if err != nil {
		return domain.ProviderCredentials{}, err
	}
	_, creds := s.resolve(key, row)
	return creds, nil
}

type SaveCredentialInput struct {
	Key domain.IntegrationKey
	// PublicID nil keeps the stored value; "" clears it.
	PublicID *string
	// Secret nil or "" keeps the stored secret.
	Secret *string
	Verify bool
}

type SaveCredentialResult struct {
	Status   domain.IntegrationStatus
	Verified *domain.VerifyResult
}

func (s *Settings) SaveCredential(ctx context.Context, userID string, in SaveCredentialInput) (*SaveCredentialResult, error) {
	if !in.Key.Valid() {
		return nil, apperr.New(apperr.Validation, "")
	}
	patch := domain.SettingPatch{UpdatedByID: &userID}
	if in.PublicID != nil {
		v := strings.TrimSpace(*in.PublicID)
		if utf8.RuneCountInString(v) > 300 {
			return nil, apperr.New(apperr.Validation, "")
		}
		patch.SetPublicID = true
		if v != "" {
			patch.PublicID = &v
		}
	}
	if in.Secret != nil {
		v := strings.TrimSpace(*in.Secret)
		if utf8.RuneCountInString(v) > 2000 {
			return nil, apperr.New(apperr.Validation, "")
		}
		if v != "" {
			enc, err := s.cipher.Encrypt(v)
			if err != nil {
				return nil, err
			}
			patch.SetSecret = true
			patch.SecretEnc = &enc
		}
	}
	row, err := s.store.UpsertSetting(ctx, in.Key, patch)
	if err != nil {
		return nil, err
	}
	out := &SaveCredentialResult{}
	if in.Verify {
		_, creds := s.resolve(in.Key, row)
		v := s.runVerify(ctx, in.Key, creds)
		out.Verified = &v
		vp := domain.SettingPatch{SetVerifiedAt: true}
		if v.OK {
			now := s.clk.Now()
			vp.VerifiedAt = &now
		}
		if row, err = s.store.UpsertSetting(ctx, in.Key, vp); err != nil {
			return nil, err
		}
	}
	out.Status, _ = s.resolve(in.Key, row)
	_ = s.bus.Publish(ctx, events.IntegrationUpdated, events.IntegrationUpdatedPayload{Key: string(in.Key)})
	return out, nil
}

func (s *Settings) runVerify(ctx context.Context, key domain.IntegrationKey, c domain.ProviderCredentials) domain.VerifyResult {
	if c.Secret == "" {
		return domain.VerifyResult{Message: apperr.Message(apperr.Validation)}
	}
	needPublic := func(msg string, check func() domain.VerifyResult) domain.VerifyResult {
		if c.PublicID == "" {
			return domain.VerifyResult{Message: msg}
		}
		return check()
	}
	switch key {
	case domain.KeyMeta:
		return needPublic(msgMetaAppID, func() domain.VerifyResult { return s.verifier.MetaApp(ctx, c.PublicID, c.Secret) })
	case domain.KeyGoogle:
		return needPublic(msgGoogleClientID, func() domain.VerifyResult { return s.verifier.GoogleClient(ctx, c.PublicID, c.Secret) })
	case domain.KeyDataForSeo:
		return needPublic(msgDfsLogin, func() domain.VerifyResult { return s.verifier.DataForSeo(ctx, c.PublicID, c.Secret) })
	case domain.KeyApify:
		return s.verifier.Apify(ctx, c.Secret)
	default:
		return s.verifier.OpenRouter(ctx, c.Secret)
	}
}

func (s *Settings) ClearCredential(ctx context.Context, key domain.IntegrationKey) error {
	if !key.Valid() {
		return apperr.New(apperr.Validation, "")
	}
	if _, err := s.store.DeleteSetting(ctx, key); err != nil {
		return err
	}
	return s.bus.Publish(ctx, events.IntegrationUpdated, events.IntegrationUpdatedPayload{Key: string(key)})
}

type MetaReviewInput struct {
	Permissions      map[string]string
	AppLive          bool
	BusinessVerified bool
}

// SaveMetaReview merges the checklist into the META row's meta. The
// permissions map replaces the stored one wholesale.
func (s *Settings) SaveMetaReview(ctx context.Context, userID string, in MetaReviewInput) (*domain.IntegrationStatus, error) {
	perms := map[string]domain.MetaReviewStatus{}
	for name, status := range in.Permissions {
		if utf8.RuneCountInString(name) > 80 || !domain.MetaReviewStatus(status).Valid() {
			return nil, apperr.New(apperr.Validation, "")
		}
		perms[name] = domain.MetaReviewStatus(status)
	}
	row, err := s.store.GetSetting(ctx, domain.KeyMeta)
	if err != nil {
		return nil, err
	}
	merged := map[string]any{}
	if row != nil && len(row.Meta) > 0 {
		_ = json.Unmarshal(row.Meta, &merged)
	}
	merged["permissions"] = perms
	merged["appLive"] = in.AppLive
	merged["businessVerified"] = in.BusinessVerified
	raw, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	row, err = s.store.UpsertSetting(ctx, domain.KeyMeta, domain.SettingPatch{Meta: raw, SetMeta: true, UpdatedByID: &userID})
	if err != nil {
		return nil, err
	}
	st, _ := s.resolve(domain.KeyMeta, row)
	return &st, nil
}

type CostInput struct {
	TrackedKeywords      int
	CompetitorDomains    int
	ResearchRunsPerMonth int
	SocialAccounts       int
	PostsPerAccount      int
	// UsdIdr 0 falls back to DATAFORSEO_USD_IDR, then 16000.
	UsdIdr float64
}

type CostEstimates struct {
	DataForSeo domain.CostEstimate
	Apify      domain.CostEstimate
}

func (s *Settings) EstimateCosts(in CostInput) CostEstimates {
	rate := in.UsdIdr
	if rate <= 0 {
		rate = float64(config.EnvInt("DATAFORSEO_USD_IDR", 16000))
	}
	return CostEstimates{
		DataForSeo: domain.EstimateSeoCost(domain.SeoCostInput{TrackedKeywords: in.TrackedKeywords, CompetitorDomains: in.CompetitorDomains, ResearchRunsPerMonth: in.ResearchRunsPerMonth, UsdIdr: rate}),
		Apify:      domain.EstimateApifyCost(domain.ApifyCostInput{TrackedKeywords: in.TrackedKeywords, SocialAccounts: in.SocialAccounts, PostsPerAccount: in.PostsPerAccount, UsdIdr: rate}),
	}
}
