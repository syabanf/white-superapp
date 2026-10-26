package gateway

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	publishingv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/publishing/v1"
	seosuitev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/seosuite/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var emptyPB = emptypb.Empty{}

type handlerReg = func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error

type handlerGroup struct {
	conn *grpc.ClientConn
	regs []handlerReg
}

func registerAll(ctx context.Context, mux *runtime.ServeMux, groups []handlerGroup) error {
	for _, g := range groups {
		for _, r := range g.regs {
			if err := r(ctx, mux, g.conn); err != nil {
				return err
			}
		}
	}
	return nil
}

// registerMore wires the analytics, publishing and seosuite REST surfaces.
func registerMore(ctx context.Context, mux *runtime.ServeMux, u *Upstreams) error {
	if err := registerAll(ctx, mux, []handlerGroup{
		{u.Analytics, []handlerReg{
			analyticsv1.RegisterOverviewServiceHandler,
			analyticsv1.RegisterSocialServiceHandler,
			analyticsv1.RegisterSeoServiceHandler,
			analyticsv1.RegisterAdsServiceHandler,
		}},
		{u.Publishing, []handlerReg{
			publishingv1.RegisterMediaServiceHandler,
			publishingv1.RegisterPostServiceHandler,
		}},
		{u.SeoSuite, []handlerReg{
			seosuitev1.RegisterKeywordServiceHandler,
			seosuitev1.RegisterResearchServiceHandler,
			seosuitev1.RegisterBacklinkServiceHandler,
			seosuitev1.RegisterCompetitorServiceHandler,
		}},
	}); err != nil {
		return err
	}
	if err := registerReports(ctx, mux, u); err != nil {
		return err
	}
	return registerErp(ctx, mux, u)
}

// File routes: multipart uploads and raw bytes, which JSON transcoding
// handles poorly (base64 inflates uploads by a third).
func (b *BrowserRoutes) mountFiles(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/clients/{client_id}/media/upload", b.uploadMedia)
	mux.HandleFunc("POST /api/clients/{client_id}/ads/import", b.importAdsCsv)
	mux.HandleFunc("GET /api/media/{key...}", b.serveMedia)
	b.mountErpFiles(mux)
	b.mountContentFiles(mux)
	b.mountRecruitmentFiles(mux)
}

const (
	maxUploadRequest = 30 << 20 // 25 MiB per file plus multipart overhead; the service re-checks
	maxCSVRequest    = 11 << 20 // 10 MB CSV limit enforced by analytics
)

func (b *BrowserRoutes) uploadMedia(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequest)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Ukuran file melebihi 25 MB."))
		return
	}
	req := &publishingv1.UploadMediaRequest{ClientId: r.PathValue("client_id")}
	for _, fh := range r.MultipartForm.File["files"] {
		f, err := fh.Open()
		if err != nil {
			grpcx.WriteError(w, apperr.Wrap(apperr.Validation, "", err))
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			grpcx.WriteError(w, apperr.Wrap(apperr.Validation, "", err))
			return
		}
		req.Files = append(req.Files, &publishingv1.UploadMediaRequest_File{Filename: fh.Filename, Content: data})
	}
	ctx := metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
	res, err := publishingv1.NewMediaServiceClient(b.up.Publishing).UploadMedia(ctx, req)
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writeProtoJSON(w, res)
}

func (b *BrowserRoutes) importAdsCsv(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCSVRequest)
	if err := r.ParseMultipartForm(maxCSVRequest); err != nil {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Ukuran file melebihi 10MB"))
		return
	}
	f, fh, err := r.FormFile("file")
	if err != nil {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Pilih file CSV terlebih dahulu."))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		grpcx.WriteError(w, apperr.Wrap(apperr.Validation, "", err))
		return
	}
	ctx := metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
	res, err := analyticsv1.NewAdsServiceClient(b.up.Analytics).ImportAdsCsv(ctx, &analyticsv1.ImportAdsCsvRequest{
		ClientId: r.PathValue("client_id"), Filename: fh.Filename, ContentType: fh.Header.Get("Content-Type"), Content: data,
	})
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writeProtoJSON(w, res)
}

// serveMedia is public on purpose: Meta fetches post media by URL, and the
// UUID keys are unguessable, like the old /api/media route.
func (b *BrowserRoutes) serveMedia(w http.ResponseWriter, r *http.Request) {
	obj, err := publishingv1.NewMediaServiceClient(b.up.Publishing).GetMediaObject(r.Context(), &publishingv1.GetMediaObjectRequest{Key: r.PathValue("key")})
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	etag := `"` + obj.Etag + `"`
	if match := r.Header.Get("If-None-Match"); match != "" && (match == etag || strings.Trim(match, `"`) == obj.Etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", etag)
	_, _ = w.Write(obj.Content)
}

func writeProtoJSON(w http.ResponseWriter, m proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(m)
	_, _ = w.Write(b)
}
