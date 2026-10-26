package gateway

import (
	"context"
	"net/http"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	reportsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/reports/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc/metadata"
)

func registerReports(ctx context.Context, mux *runtime.ServeMux, u *Upstreams) error {
	return reportsv1.RegisterReportServiceHandler(ctx, mux, u.Reports)
}

// PDF downloads and the public share dashboard. Share access rides on the
// share_<clientId> cookie set by /api/share/{slug}/unlock.
func (b *BrowserRoutes) mountReports(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/reports/{id}/pdf", b.reportPdf)
	mux.HandleFunc("GET /api/share/{slug}/pdf", b.sharePdf)
	mux.HandleFunc("GET /api/share/{slug}/dashboard", b.shareDashboard)
}

func (b *BrowserRoutes) reportPdf(w http.ResponseWriter, r *http.Request) {
	ctx := metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
	res, err := reportsv1.NewReportServiceClient(b.up.Reports).RenderReportPdf(ctx, &reportsv1.RenderReportPdfRequest{ReportId: r.PathValue("id")})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writePdf(w, res.Filename, res.Pdf)
}

// shareUnlockToken finds the client behind a slug and returns its share cookie.
func (b *BrowserRoutes) shareUnlockToken(r *http.Request, slug string) string {
	sc, err := identityv1.NewShareServiceClient(b.up.Identity).GetShareContext(r.Context(), &identityv1.GetShareContextRequest{Slug: slug})
	if err != nil || sc.Client == nil {
		return ""
	}
	if c, err := r.Cookie(shareCookieName(sc.Client.Id)); err == nil {
		return c.Value
	}
	return ""
}

// shareStatusError mirrors the old share PDF route: 404 NOT_FOUND,
// 404 DISABLED, 401 PIN_REQUIRED.
func shareStatusError(w http.ResponseWriter, st reportsv1.ShareStatus) bool {
	switch st {
	case reportsv1.ShareStatus_OK:
		return false
	case reportsv1.ShareStatus_LOCKED:
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]string{"error": "PIN_REQUIRED"})
	case reportsv1.ShareStatus_DISABLED:
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "DISABLED"})
	default:
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "NOT_FOUND"})
	}
	return true
}

func (b *BrowserRoutes) sharePdf(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	res, err := reportsv1.NewReportServiceClient(b.up.Reports).RenderSharePdf(r.Context(), &reportsv1.RenderSharePdfRequest{
		Slug: slug, UnlockToken: b.shareUnlockToken(r, slug), Preset: r.URL.Query().Get("preset"),
	})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	if shareStatusError(w, res.Status) {
		return
	}
	writePdf(w, res.Filename, res.Pdf)
}

func (b *BrowserRoutes) shareDashboard(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	res, err := reportsv1.NewReportServiceClient(b.up.Reports).GetShareDashboard(r.Context(), &reportsv1.GetShareDashboardRequest{
		Slug: slug, UnlockToken: b.shareUnlockToken(r, slug), Preset: r.URL.Query().Get("preset"),
	})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	// The page renders its own locked/disabled states, so the body always
	// carries the status; only robots get told to stay away.
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	writeProtoJSON(w, res)
}

func writePdf(w http.ResponseWriter, filename string, pdf []byte) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(pdf)
}
