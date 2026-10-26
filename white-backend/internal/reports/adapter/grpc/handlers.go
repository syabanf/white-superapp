// Package grpcapi exposes the reports use cases over gRPC.
package grpcapi

import (
	"context"

	reportsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/reports/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/grpc"
)

func Register(s *grpc.Server, reports *app.Reports) {
	reportsv1.RegisterReportServiceServer(s, &server{reports: reports})
}

type server struct {
	reportsv1.UnimplementedReportServiceServer
	reports *app.Reports
}

func caller(ctx context.Context) (string, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return "", err
	}
	return p.UserID, nil
}

func (s *server) ListReports(ctx context.Context, req *reportsv1.ListReportsRequest) (*reportsv1.ListReportsResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.reports.List(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &reportsv1.ListReportsResponse{Reports: make([]*reportsv1.Report, 0, len(rows))}
	for _, r := range rows {
		out.Reports = append(out.Reports, reportPB(r))
	}
	return out, nil
}

func (s *server) CreateReport(ctx context.Context, req *reportsv1.CreateReportRequest) (*reportsv1.Report, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	row, err := s.reports.Create(ctx, uid, req.ClientId, app.CreateInput{
		Title: req.Title, Modules: modules(req.Modules), Compare: req.Compare, IncludeAI: req.IncludeAi,
		Language: req.Language, From: req.From, To: req.To,
	})
	if err != nil {
		return nil, err
	}
	return reportPB(*row), nil
}

func (s *server) DeleteReport(ctx context.Context, req *reportsv1.DeleteReportRequest) (*reportsv1.DeleteReportResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.reports.Delete(ctx, uid, req.ClientId, req.ReportId); err != nil {
		return nil, err
	}
	return &reportsv1.DeleteReportResponse{Id: req.ReportId}, nil
}

func (s *server) GenerateInsight(ctx context.Context, req *reportsv1.GenerateInsightRequest) (*reportsv1.Insight, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	in, err := s.reports.GenerateInsight(ctx, uid, req.ClientId, insightModule(req.Module), req.From, req.To, req.Language)
	if err != nil {
		return nil, err
	}
	return insightPB(in), nil
}

func (s *server) GetLatestInsight(ctx context.Context, req *reportsv1.GetLatestInsightRequest) (*reportsv1.GetLatestInsightResponse, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	in, err := s.reports.LatestInsight(ctx, uid, req.ClientId, insightModule(req.Module), req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := &reportsv1.GetLatestInsightResponse{}
	if in != nil {
		out.Insight = insightPB(in)
	}
	return out, nil
}

func (s *server) GetReportSections(ctx context.Context, req *reportsv1.GetReportSectionsRequest) (*reportsv1.ReportSections, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	secs, err := s.reports.Sections(ctx, uid, req.ClientId, req.From, req.To, req.Compare, modules(req.Modules), req.Language)
	if err != nil {
		return nil, err
	}
	return sectionsPB(secs), nil
}

func (s *server) RenderReportPdf(ctx context.Context, req *reportsv1.RenderReportPdfRequest) (*reportsv1.PdfFile, error) {
	uid := req.UserId
	if uid == "" {
		p, ok := authn.FromContext(ctx)
		if !ok {
			return nil, apperr.New(apperr.Unauthenticated, "")
		}
		uid = p.UserID
	}
	name, pdf, err := s.reports.RenderReportPdf(ctx, uid, req.ReportId)
	if err != nil {
		return nil, err
	}
	return &reportsv1.PdfFile{Filename: name, Pdf: pdf}, nil
}

func (s *server) RenderSharePdf(ctx context.Context, req *reportsv1.RenderSharePdfRequest) (*reportsv1.SharePdf, error) {
	res, err := s.reports.SharePdf(ctx, req.Slug, req.UnlockToken, req.Preset)
	if err != nil {
		return nil, err
	}
	return &reportsv1.SharePdf{Status: shareStatusPB(res.Status), Filename: res.Filename, Pdf: res.Pdf}, nil
}

func (s *server) GetShareDashboard(ctx context.Context, req *reportsv1.GetShareDashboardRequest) (*reportsv1.ShareDashboard, error) {
	res, err := s.reports.ShareDashboard(ctx, req.Slug, req.UnlockToken, req.Preset)
	if err != nil {
		return nil, err
	}
	return shareDashboardPB(res), nil
}
