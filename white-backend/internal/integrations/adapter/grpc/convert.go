// Package grpcapi exposes the integrations use cases over gRPC. Handlers
// only translate between proto and domain; rules live in app.
package grpcapi

import (
	"encoding/json"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/app"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func providerFrom(p commonv1.ConnectionProvider) domain.Provider {
	switch p {
	case commonv1.ConnectionProvider_META:
		return domain.ProviderMeta
	case commonv1.ConnectionProvider_GOOGLE:
		return domain.ProviderGoogle
	}
	return ""
}

func providerPB(p domain.Provider) commonv1.ConnectionProvider {
	switch p {
	case domain.ProviderMeta:
		return commonv1.ConnectionProvider_META
	case domain.ProviderGoogle:
		return commonv1.ConnectionProvider_GOOGLE
	}
	return commonv1.ConnectionProvider_CONNECTION_PROVIDER_UNSPECIFIED
}

func connectionPB(c *domain.Connection) *integrationsv1.Connection {
	if c == nil {
		return nil
	}
	return &integrationsv1.Connection{
		Id: c.ID, ClientId: c.ClientID, Provider: providerPB(c.Provider), AccountId: c.AccountID, DisplayName: c.DisplayName,
		ExpiresAt: tsPtr(c.ExpiresAt), Scopes: c.Scopes, CreatedAt: timestamppb.New(c.CreatedAt),
	}
}

var keyFromPB = map[integrationsv1.IntegrationKey]domain.IntegrationKey{
	integrationsv1.IntegrationKey_META:       domain.KeyMeta,
	integrationsv1.IntegrationKey_GOOGLE:     domain.KeyGoogle,
	integrationsv1.IntegrationKey_DATAFORSEO: domain.KeyDataForSeo,
	integrationsv1.IntegrationKey_APIFY:      domain.KeyApify,
	integrationsv1.IntegrationKey_OPENROUTER: domain.KeyOpenRouter,
}

func keyFrom(k integrationsv1.IntegrationKey) domain.IntegrationKey { return keyFromPB[k] }

func keyPB(k domain.IntegrationKey) integrationsv1.IntegrationKey {
	for pb, dk := range keyFromPB {
		if dk == k {
			return pb
		}
	}
	return integrationsv1.IntegrationKey_INTEGRATION_KEY_UNSPECIFIED
}

func sourcePB(s domain.CredentialSource) integrationsv1.IntegrationStatus_Source {
	switch s {
	case domain.SourceDB:
		return integrationsv1.IntegrationStatus_DB
	case domain.SourceEnv:
		return integrationsv1.IntegrationStatus_ENV
	}
	return integrationsv1.IntegrationStatus_NONE
}

func metaStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	st, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return st
}

func statusPB(s domain.IntegrationStatus) *integrationsv1.IntegrationStatus {
	return &integrationsv1.IntegrationStatus{
		Key: keyPB(s.Key), Configured: s.Configured, Source: sourcePB(s.Source), PublicId: s.PublicID,
		SecretMasked: s.SecretMasked, Meta: metaStruct(s.Meta), VerifiedAt: tsPtr(s.VerifiedAt),
	}
}

func reviewPB(s domain.MetaReviewState) *integrationsv1.MetaReview {
	p := s.Progress()
	perms := make(map[string]string, len(s.Permissions))
	for k, v := range s.Permissions {
		perms[k] = string(v)
	}
	return &integrationsv1.MetaReview{
		Permissions: perms, AppLive: s.AppLive, BusinessVerified: s.BusinessVerified,
		ApprovedRequired: int32(p.Approved), TotalRequired: int32(p.Total), PublishReady: p.PublishReady, AdsReady: p.AdsReady,
	}
}

func verifyPB(v *domain.VerifyResult) *integrationsv1.VerifyResult {
	if v == nil {
		return nil
	}
	return &integrationsv1.VerifyResult{Ok: v.OK, Message: v.Message, Detail: v.Detail}
}

func estimatePB(e domain.CostEstimate) *integrationsv1.EstimateCostsResponse_CostEstimate {
	return &integrationsv1.EstimateCostsResponse_CostEstimate{LinesUsd: e.LinesUSD, TotalUsd: e.TotalUSD, TotalIdr: e.TotalIDR}
}

func costInput(r *integrationsv1.EstimateCostsRequest) app.CostInput {
	return app.CostInput{
		TrackedKeywords: int(r.TrackedKeywords), CompetitorDomains: int(r.CompetitorDomains), ResearchRunsPerMonth: int(r.ResearchRunsPerMonth),
		SocialAccounts: int(r.SocialAccounts), PostsPerAccount: int(r.PostsPerAccount), UsdIdr: r.UsdIdr,
	}
}
