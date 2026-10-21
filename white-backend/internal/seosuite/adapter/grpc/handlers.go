package grpcapi

import (
	"context"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	seosuitev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/seosuite/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Services struct {
	Keywords    *app.Keywords
	Research    *app.Research
	Backlinks   *app.Backlinks
	Competitors *app.Competitors
	Reports     *app.Reports
}

func Register(s *grpc.Server, svc Services) {
	seosuitev1.RegisterKeywordServiceServer(s, &keywordServer{svc: svc})
	seosuitev1.RegisterResearchServiceServer(s, &researchServer{svc: svc})
	seosuitev1.RegisterBacklinkServiceServer(s, &backlinkServer{svc: svc})
	seosuitev1.RegisterCompetitorServiceServer(s, &competitorServer{svc: svc})
	seosuitev1.RegisterReportDataServiceServer(s, &reportServer{svc: svc})
}

func resolve(q *commonv1.RangeQuery) dates.Resolved { return dates.Resolve(q, time.Now().UTC()) }

// ─── KeywordService ──────────────────────────────────────────

type keywordServer struct {
	seosuitev1.UnimplementedKeywordServiceServer
	svc Services
}

func (s *keywordServer) TrackKeywords(ctx context.Context, req *seosuitev1.TrackKeywordsRequest) (*seosuitev1.AddedResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	ideas := make([]app.IdeaInput, 0, len(req.Ideas))
	for _, i := range req.Ideas {
		ideas = append(ideas, app.IdeaInput{Keyword: i.Keyword, Volume: i.Volume, Difficulty: i.Difficulty, CPC: i.Cpc, Intent: intentFrom(i.Intent)})
	}
	added, err := s.svc.Keywords.Track(ctx, p.UserID, req.ClientId, req.PropertyId, ideas, deviceFrom(req.Device), req.Tags)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.AddedResponse{Added: added}, nil
}

func (s *keywordServer) AddTrackedKeywords(ctx context.Context, req *seosuitev1.AddTrackedKeywordsRequest) (*seosuitev1.AddedResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	added, err := s.svc.Keywords.Add(ctx, p.UserID, req.ClientId, req.PropertyId, req.Text, deviceFrom(req.Device), req.Tags)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.AddedResponse{Added: added}, nil
}

func (s *keywordServer) DeleteTrackedKeyword(ctx context.Context, req *seosuitev1.DeleteTrackedKeywordRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Keywords.Delete(ctx, p.UserID, req.ClientId, req.KeywordId)
}

func (s *keywordServer) UpdateKeywordTags(ctx context.Context, req *seosuitev1.UpdateKeywordTagsRequest) (*seosuitev1.UpdateKeywordTagsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	tags, err := s.svc.Keywords.UpdateTags(ctx, p.UserID, req.ClientId, req.KeywordId, req.Tags)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.UpdateKeywordTagsResponse{Tags: tags}, nil
}

func (s *keywordServer) RefreshRanks(ctx context.Context, req *seosuitev1.PropertyRequest) (*seosuitev1.RefreshRanksResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := s.svc.Keywords.RefreshRanks(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.RefreshRanksResponse{Updated: updated}, nil
}

func (s *keywordServer) GetRankData(ctx context.Context, req *seosuitev1.GetRankDataRequest) (*seosuitev1.RankData, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	data, err := s.svc.Keywords.RankData(ctx, p.UserID, req.ClientId, req.PropertyId, resolve(req.Range))
	if err != nil {
		return nil, err
	}
	return rankData(data), nil
}

func (s *keywordServer) GetTrackedKeywordSet(ctx context.Context, req *seosuitev1.PropertyRequest) (*seosuitev1.TrackedKeywordSet, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	keywords, err := s.svc.Keywords.TrackedSet(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.TrackedKeywordSet{Keywords: keywords}, nil
}

func (s *keywordServer) ListSyncJobs(ctx context.Context, req *seosuitev1.ListSyncJobsRequest) (*seosuitev1.ListSyncJobsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	jobs, err := s.svc.Keywords.ListJobs(ctx, p.UserID, req.ClientId, req.Kinds, req.Limit)
	if err != nil {
		return nil, err
	}
	out := &seosuitev1.ListSyncJobsResponse{}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, syncJob(j))
	}
	return out, nil
}

// ─── ResearchService ─────────────────────────────────────────

type researchServer struct {
	seosuitev1.UnimplementedResearchServiceServer
	svc Services
}

func (s *researchServer) GetResearch(ctx context.Context, req *seosuitev1.GetResearchRequest) (*seosuitev1.ResearchResult, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Research.Get(ctx, p.UserID, req.ClientId, req.Seed, researchModeFrom(req.Mode))
	if err != nil {
		return nil, err
	}
	out := &seosuitev1.ResearchResult{FetchedAt: timestamppb.New(res.FetchedAt), FromCache: res.FromCache, Provider: res.Provider, Mode: researchMode(res.Mode), Seed: res.Seed}
	for _, i := range res.Ideas {
		out.Ideas = append(out.Ideas, keywordIdea(i))
	}
	return out, nil
}

func (s *researchServer) GetKeywordGap(ctx context.Context, req *seosuitev1.GetKeywordGapRequest) (*seosuitev1.KeywordGap, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	gap, err := s.svc.Research.KeywordGap(ctx, p.UserID, req.ClientId, req.CompetitorDomain)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.KeywordGap{Missing: gapRows(gap.Missing), Weak: gapRows(gap.Weak), Strong: gapRows(gap.Strong), Shared: gapRows(gap.Shared), OwnDomain: gap.OwnDomain, CompetitorDomain: gap.CompetitorDomain}, nil
}

// ─── BacklinkService ─────────────────────────────────────────

type backlinkServer struct {
	seosuitev1.UnimplementedBacklinkServiceServer
	svc Services
}

func (s *backlinkServer) RefreshBacklinks(ctx context.Context, req *seosuitev1.PropertyRequest) (*seosuitev1.RefreshBacklinksResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	added, lost, err := s.svc.Backlinks.Refresh(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.RefreshBacklinksResponse{Added: added, Lost: lost}, nil
}

func (s *backlinkServer) GetBacklinkData(ctx context.Context, req *seosuitev1.GetBacklinkDataRequest) (*seosuitev1.BacklinkData, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	data, err := s.svc.Backlinks.Data(ctx, p.UserID, req.ClientId, req.PropertyId, resolve(req.Range))
	if err != nil {
		return nil, err
	}
	return backlinkData(data), nil
}

// ─── CompetitorService ───────────────────────────────────────

type competitorServer struct {
	seosuitev1.UnimplementedCompetitorServiceServer
	svc Services
}

func (s *competitorServer) AddCompetitorDomain(ctx context.Context, req *seosuitev1.AddCompetitorDomainRequest) (*seosuitev1.CompetitorDomain, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	row, err := s.svc.Competitors.Add(ctx, p.UserID, req.ClientId, req.PropertyId, req.Domain)
	if err != nil {
		return nil, err
	}
	return competitorDomain(*row), nil
}

func (s *competitorServer) RemoveCompetitorDomain(ctx context.Context, req *seosuitev1.RemoveCompetitorDomainRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Competitors.Remove(ctx, p.UserID, req.ClientId, req.Id)
}

func (s *competitorServer) RefreshDomains(ctx context.Context, req *seosuitev1.PropertyRequest) (*seosuitev1.RefreshDomainsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.svc.Competitors.Refresh(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	return &seosuitev1.RefreshDomainsResponse{Domains: n}, nil
}

func (s *competitorServer) GetCompetitorData(ctx context.Context, req *seosuitev1.GetCompetitorDataRequest) (*seosuitev1.CompetitorData, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	data, err := s.svc.Competitors.Data(ctx, p.UserID, req.ClientId, req.PropertyId, resolve(req.Range))
	if err != nil {
		return nil, err
	}
	return competitorData(data), nil
}

// ─── ReportDataService (internal) ────────────────────────────

type reportServer struct {
	seosuitev1.UnimplementedReportDataServiceServer
	svc Services
}

func rangeArg(dr *commonv1.DateRange) (dates.Range, error) {
	r, err := dates.FromProto(dr)
	if err != nil {
		return dates.Range{}, apperr.New(apperr.Validation, err.Error())
	}
	return r, nil
}

func (s *reportServer) GetRankSection(ctx context.Context, req *seosuitev1.GetRankSectionRequest) (*seosuitev1.RankSection, error) {
	current, err := rangeArg(req.Current)
	if err != nil {
		return nil, err
	}
	previous := dates.Previous(current)
	if req.Previous != nil {
		if previous, err = rangeArg(req.Previous); err != nil {
			return nil, err
		}
	}
	sec, err := s.svc.Reports.RankSection(ctx, req.PropertyId, current, previous)
	if err != nil {
		return nil, err
	}
	out := &seosuitev1.RankSection{HasKeywords: sec.HasKeywords, LatestDate: dayPtr(sec.LatestDate), Kpis: rankKpis(sec.Kpis), Deltas: rankDeltas(sec.Deltas)}
	for _, r := range sec.Rows {
		out.Rows = append(out.Rows, &seosuitev1.RankSectionRow{Keyword: r.Keyword, Position: r.Position, Change: rankChange(r.Change), Volume: r.Volume})
	}
	return out, nil
}

func (s *reportServer) GetBacklinkSection(ctx context.Context, req *seosuitev1.GetBacklinkSectionRequest) (*seosuitev1.BacklinkSection, error) {
	rng, err := rangeArg(req.Range)
	if err != nil {
		return nil, err
	}
	sec, err := s.svc.Reports.BacklinkSection(ctx, req.PropertyId, rng)
	if err != nil {
		return nil, err
	}
	out := &seosuitev1.BacklinkSection{DofollowShare: sec.DofollowShare, NewBacklinks: sec.New, LostBacklinks: sec.Lost}
	if l := sec.Latest; l != nil {
		out.HasSnapshot = true
		out.LatestDate = dayPtr(&l.Date)
		out.Backlinks = l.Backlinks
		out.ReferringDomains = l.ReferringDomains
		out.DomainRank = l.DomainRank
	}
	return out, nil
}
