package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Research is the keyword research cache and the keyword gap.
type Research struct {
	store domain.Store
	guard *Guard
	seo   seodata.Source
	clk   clock.Clock
}

func NewResearch(store domain.Store, guard *Guard, seo seodata.Source, clk clock.Clock) *Research {
	return &Research{store: store, guard: guard, seo: seo, clk: clk}
}

type ResearchResult struct {
	Ideas     []domain.KeywordIdea
	FetchedAt time.Time
	FromCache bool
	Provider  string
	Mode      domain.ResearchMode
	Seed      string
}

// Get answers from keyword_research when the newest row for
// "{mode}:{seed}" is under 7 days old, otherwise calls the provider, stores
// the run and drops older runs of the same key.
func (r *Research) Get(ctx context.Context, userID, clientID, seed string, mode domain.ResearchMode) (*ResearchResult, error) {
	seed = domain.NormalizeKeyword(seed)
	if seed == "" {
		return nil, apperr.New(apperr.Validation, "")
	}
	if _, err := r.guard.Client(ctx, userID, clientID, false); err != nil {
		return nil, err
	}
	p, err := r.guard.SuiteProperty(ctx, clientID)
	if err != nil {
		return nil, err
	}
	key := string(mode) + ":" + seed
	now := r.clk.Now()
	cached, err := r.store.GetLatestResearch(ctx, clientID, key, p.LocationCode, p.LanguageCode)
	if err != nil {
		return nil, err
	}
	if cached != nil && domain.IsCacheFresh(cached.FetchedAt, now, domain.ResearchMaxAgeDays) {
		return &ResearchResult{Ideas: cached.Results, FetchedAt: cached.FetchedAt, FromCache: true, Provider: cached.Provider, Mode: mode, Seed: seed}, nil
	}
	provider, pmode, err := r.seo.Provider(ctx)
	if err != nil {
		return nil, err
	}
	ideas, err := provider.KeywordIdeas(ctx, seed, p.LocationCode, p.LanguageCode, mode, 100)
	if err != nil {
		return nil, err
	}
	if ideas == nil {
		ideas = []domain.KeywordIdea{}
	}
	row := domain.KeywordResearch{ID: ids.New(), ClientID: clientID, Seed: key, LocationCode: p.LocationCode, LanguageCode: p.LanguageCode, Provider: string(pmode), Results: ideas, FetchedAt: now}
	if err := r.store.InsertResearch(ctx, row); err != nil {
		return nil, err
	}
	if err := r.store.DeleteOtherResearch(ctx, clientID, key, p.LocationCode, p.LanguageCode, row.ID); err != nil {
		return nil, err
	}
	return &ResearchResult{Ideas: ideas, FetchedAt: now, FromCache: false, Provider: string(pmode), Mode: mode, Seed: seed}, nil
}

type KeywordGapResult struct {
	domain.KeywordGap
	OwnDomain        string
	CompetitorDomain string
}

// KeywordGap compares the suite property's host with one competitor domain.
// Live provider call, not cached.
func (r *Research) KeywordGap(ctx context.Context, userID, clientID, competitorDomain string) (*KeywordGapResult, error) {
	comp := domain.NormalizeDomain(competitorDomain)
	if comp == "" {
		return nil, apperr.New(apperr.Validation, msgDomainInvalid)
	}
	if _, err := r.guard.Client(ctx, userID, clientID, false); err != nil {
		return nil, err
	}
	p, err := r.guard.SuiteProperty(ctx, clientID)
	if err != nil {
		return nil, err
	}
	provider, _, err := r.seo.Provider(ctx)
	if err != nil {
		return nil, err
	}
	own, err := provider.DomainKeywords(ctx, p.Host(), p.LocationCode, p.LanguageCode, 200)
	if err != nil {
		return nil, err
	}
	theirs, err := provider.DomainKeywords(ctx, comp, p.LocationCode, p.LanguageCode, 200)
	if err != nil {
		return nil, err
	}
	return &KeywordGapResult{KeywordGap: domain.KeywordGapOf(own, theirs), OwnDomain: p.Host(), CompetitorDomain: comp}, nil
}
