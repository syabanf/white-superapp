package grpcapi_test

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// TestHandlersOverGRPC seeds the demo data and calls the service through a
// real gRPC server: anonymous portal calls through the edge, permission
// checks on staff RPCs, token hiding and file downloads.
func TestHandlersOverGRPC(t *testing.T) {
	pool := pg.TestPool(t, recruitment.Migrations(), "migrations", recruitment.Schema)
	pg.TruncateSchema(t, pool, recruitment.Schema)
	ctx := context.Background()
	raw, err := fs.ReadFile(recruitment.Migrations(), "00002_reference_data.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	if err := seed.Seed(ctx, pool, nil, nil, time.Now().UTC(), logger); err != nil {
		t.Fatal(err)
	}
	deps := app.Deps{Store: postgres.NewStore(pool), AI: openrouter.NewSource(nil, openrouter.Options{}), Files: storage.NewLocal(t.TempDir()),
		PDF: pdf.Renderer{}, Events: events.Noop{Logger: logger}, Logger: logger}
	svc := grpcapi.Services{Jobs: app.NewJobOpenings(deps), Candidates: app.NewCandidates(deps), Psikotes: app.NewPsikotes(deps),
		Interviews: app.NewInterviews(deps), Offers: app.NewOffers(deps), Portal: app.NewPortal(deps)}

	priv, err := authn.KeyPair("", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := authn.PublicKey("", "", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpcx.NewServer(grpcx.ServerOptions{Logger: logger, Verifier: authn.NewVerifier(pub)})
	grpcapi.Register(srv, svc)
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	issuer := authn.NewIssuer(priv, time.Hour)
	edge := metadata.AppendToOutgoingContext(ctx, grpcx.EdgeHeader, "1", "x-forwarded-for", "203.0.113.9")
	as := func(perms ...string) context.Context {
		tok, _, err := issuer.Issue(authn.Principal{UserID: "u_hr", Name: "Nadia", Role: authn.RoleMember, Perms: perms}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return grpcx.WithBearer(edge, tok)
	}
	code := func(err error) apperr.Code { return grpcx.FromStatus(err).Code }
	writer, reader := as(authn.PermRecruitWrite), as(authn.PermRecruitRead)

	public := pb.NewPublicRecruitmentServiceClient(conn)
	jobs, err := public.ListPublicJobs(edge, &pb.ListPublicJobsRequest{})
	if err != nil || len(jobs.Jobs) != 6 {
		t.Fatalf("career page: %v %v", jobs, err)
	}
	job, err := public.GetPublicJob(edge, &pb.PublicJobRequest{Slug: "seo-analyst"})
	if err != nil || job.Title != "SEO Analyst" {
		t.Fatalf("job: %v %v", job, err)
	}
	if _, err := public.GetPublicJob(edge, &pb.PublicJobRequest{Slug: "tidak-ada"}); code(err) != apperr.NotFound {
		t.Fatalf("unknown slug: %v", err)
	}
	applied, err := public.Apply(edge, &pb.ApplyRequest{Slug: "seo-analyst", FullName: "Wahyu Hidayat", Email: "wahyu@mail.com",
		Phone: "081211223344", Domicile: "Jakarta", Cv: &pb.FileUpload{FileName: "cv.pdf", Data: []byte("%PDF-1.4 test")}})
	if err != nil || applied.Message != "Lamaran berhasil dikirim" {
		t.Fatalf("apply: %v %v", applied, err)
	}

	cands := pb.NewCandidateServiceClient(conn)
	if _, err := cands.ListCandidates(edge, &pb.ListCandidatesRequest{}); code(err) != apperr.Unauthenticated {
		t.Fatalf("staff RPCs need a login: %v", err)
	}
	if _, err := cands.ListCandidates(as(authn.PermSelfService), &pb.ListCandidatesRequest{}); code(err) != apperr.Unauthorized {
		t.Fatalf("staff RPCs need recruitment.read: %v", err)
	}
	list, err := cands.ListCandidates(reader, &pb.ListCandidatesRequest{Limit: 10})
	if err != nil || list.Total != 481 || len(list.Candidates) != 10 || list.TotalPages != 49 {
		t.Fatalf("list: %v %v", list.GetTotal(), err)
	}
	if _, err := pb.NewJobOpeningServiceClient(conn).CreateJobOpening(reader, &pb.JobOpeningInput{Title: "X"}); code(err) != apperr.Unauthorized {
		t.Fatalf("writes need recruitment.write: %v", err)
	}
	cv, err := cands.DownloadCv(reader, &pb.CandidateRequest{Id: applied.CandidateId})
	if err != nil || cv.ContentType != "application/pdf" || cv.FileName != "cv.pdf" {
		t.Fatalf("cv: %v %v", cv, err)
	}
	board, err := cands.GetPipeline(reader, &pb.GetPipelineRequest{})
	if err != nil || len(board.Columns) != 8 || board.Columns[0].Count != 11 {
		t.Fatalf("pipeline: %v %v", board, err)
	}
	report, err := cands.DownloadPipelineReport(reader, &pb.CandidateRequest{Id: "seed_cand_31"})
	if err != nil || !bytes.HasPrefix(report.Data, []byte("%PDF-")) || report.FileName != "laporan-pipeline-gilang-ramadhan.pdf" {
		t.Fatalf("report: %v", err)
	}
	summary, err := cands.GetRecruitmentSummary(reader, nil)
	if err != nil || summary.OpenJobs != 6 || summary.OpenOffers != 3 {
		t.Fatalf("summary: %v %v", summary, err)
	}

	// dashboard: twelve months by default, the funnel and the sources add up to the applicants
	dash, err := cands.GetRecruitmentDashboard(reader, &pb.RecruitmentDashboardRequest{})
	if err != nil || len(dash.Trend) < 52 || len(dash.PreviousTrend) < 52 || dash.Compare != "previous" {
		t.Fatalf("dashboard: %v", err)
	}
	k := dash.Kpis
	if k.Applicants < 250 || k.Hires < 15 || k.OpenPositions != 6 || k.AvgDaysToHire == nil || k.OfferAcceptanceRate == nil {
		t.Fatalf("dashboard kpis: %v", k)
	}
	var fromSources int32
	for _, src := range dash.Sources {
		fromSources += src.Applications
	}
	if dash.Funnel[0].Reached != k.Applicants || fromSources != k.Applicants || dash.Funnel[5].Reached == 0 {
		t.Fatalf("dashboard funnel: %v sources %d", dash.Funnel, fromSources)
	}
	if len(dash.StageDurations) == 0 || len(dash.Openings) < 12 || len(dash.Offers) == 0 || len(dash.Scorecards) == 0 ||
		len(dash.Interviewers) == 0 || len(dash.PsikotesScores) == 0 {
		t.Fatalf("dashboard detail: %d durations, %d openings, %d offers", len(dash.StageDurations), len(dash.Openings), len(dash.Offers))
	}
	smm, err := cands.GetRecruitmentDashboard(reader, &pb.RecruitmentDashboardRequest{Compare: "year", Department: "Social Media", Source: "referral"})
	if err != nil || smm.Compare != "year" || smm.Kpis.Applicants == 0 || smm.Kpis.Applicants >= k.Applicants {
		t.Fatalf("filtered dashboard: %v %v", smm.GetKpis(), err)
	}
	for _, src := range smm.Sources {
		if src.Source != "referral" {
			t.Fatalf("source filter leaked %s", src.Source)
		}
	}
	if _, err := cands.GetRecruitmentDashboard(reader, &pb.RecruitmentDashboardRequest{From: "2026-10-01", To: "2026-09-01"}); code(err) != apperr.Validation {
		t.Fatalf("reversed range: %v", err)
	}
	if _, err := cands.GetRecruitmentDashboard(as(authn.PermSelfService), &pb.RecruitmentDashboardRequest{}); code(err) != apperr.Unauthorized {
		t.Fatalf("dashboard needs recruitment.read: %v", err)
	}

	// tokens go to writers only; the portals open with them
	psi := pb.NewPsikotesServiceClient(conn)
	hidden, err := psi.GetCandidatePsikotes(reader, &pb.CandidatePanelRequest{CandidateId: "seed_cand_13"})
	if err != nil || hidden.Sessions[0].Token != "" {
		t.Fatalf("reader panel: %v %v", hidden, err)
	}
	panel, err := psi.GetCandidatePsikotes(writer, &pb.CandidatePanelRequest{CandidateId: "seed_cand_13"})
	if err != nil || panel.Sessions[0].Token == "" {
		t.Fatalf("writer panel: %v %v", panel, err)
	}
	portal, err := public.GetPsikotesPortal(edge, &pb.PortalTokenRequest{Token: panel.Sessions[0].Token})
	if err != nil || portal.CandidateName != "Melati Anggraini" || len(portal.Tests) != 2 {
		t.Fatalf("psikotes portal: %v %v", portal, err)
	}
	scored, err := psi.GetCandidatePsikotes(reader, &pb.CandidatePanelRequest{CandidateId: "seed_cand_14"})
	if err != nil || scored.Summary.Recommendation != "lolos" || scored.Sessions[0].Tests[0].GetScoreDetail() == nil {
		t.Fatalf("scored psikotes: %v %v", scored, err)
	}

	offers := pb.NewOfferServiceClient(conn)
	panelOffers, err := offers.GetCandidateOffers(writer, &pb.CandidatePanelRequest{CandidateId: "seed_cand_08"})
	if err != nil || len(panelOffers.Offers) != 1 || panelOffers.Offers[0].Status != "sent" {
		t.Fatalf("offers: %v %v", panelOffers, err)
	}
	offerView, err := public.GetOfferPortal(edge, &pb.PortalTokenRequest{Token: panelOffers.Offers[0].Token})
	if err != nil || offerView.CompanyName != "Mixist" || offerView.BaseSalary != "7000000.00" {
		t.Fatalf("offer portal: %v %v", offerView, err)
	}
	letter, err := public.DownloadPublicOfferLetter(edge, &pb.PortalTokenRequest{Token: panelOffers.Offers[0].Token})
	if err != nil || letter.ContentType != "application/pdf" {
		t.Fatalf("letter: %v", err)
	}
	if _, err := public.RespondOffer(edge, &pb.RespondOfferRequest{Token: panelOffers.Offers[0].Token, Action: "maybe"}); code(err) != apperr.Validation {
		t.Fatalf("unknown action: %v", err)
	}
	ai, err := pb.NewInterviewServiceClient(conn).GetCandidateAiInterviews(reader, &pb.CandidatePanelRequest{CandidateId: "seed_cand_14"})
	if err != nil || len(ai.Sessions) != 1 || ai.Sessions[0].AiSummary.GetSalaryValue() != "8,5 juta" || len(ai.Sessions[0].Turns) != 5 {
		t.Fatalf("ai interviews: %v %v", ai, err)
	}
}
