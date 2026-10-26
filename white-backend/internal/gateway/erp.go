package gateway

import (
	"context"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	performancev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	recruitmentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// dialErp dials the ERP services and content (marketing, built alongside). grpc dials lazily, so a service that
// is not running only fails the requests routed to it.
func (u *Upstreams) dialErp() (err error) {
	if u.Finance, err = grpcx.Dial(config.Env("FINANCE_ADDR", "localhost:9007")); err != nil {
		return err
	}
	if u.HR, err = grpcx.Dial(config.Env("HR_ADDR", "localhost:9008")); err != nil {
		return err
	}
	if u.Sales, err = grpcx.Dial(config.Env("SALES_ADDR", "localhost:9010")); err != nil {
		return err
	}
	if u.Performance, err = grpcx.Dial(config.Env("PERFORMANCE_ADDR", "localhost:9012")); err != nil {
		return err
	}
	if u.Content, err = grpcx.Dial(config.Env("CONTENT_ADDR", "localhost:9011")); err != nil {
		return err
	}
	u.Recruitment, err = grpcx.Dial(config.Env("RECRUITMENT_ADDR", "localhost:9009"))
	return err
}

func registerErp(ctx context.Context, mux *runtime.ServeMux, u *Upstreams) error {
	return registerAll(ctx, mux, []handlerGroup{
		{u.Finance, []handlerReg{
			financev1.RegisterAccountServiceHandler,
			financev1.RegisterFiscalServiceHandler,
			financev1.RegisterJournalServiceHandler,
			financev1.RegisterCashBankServiceHandler,
			financev1.RegisterPayableServiceHandler,
			financev1.RegisterReceivableServiceHandler,
			financev1.RegisterReportServiceHandler,
			financev1.RegisterReimbursementServiceHandler,
			financev1.RegisterAnalysisServiceHandler,
		}},
		{u.HR, []handlerReg{
			hrv1.RegisterEmployeeServiceHandler,
			hrv1.RegisterPeopleServiceHandler,
			hrv1.RegisterAttendanceServiceHandler,
			hrv1.RegisterScheduleServiceHandler,
			hrv1.RegisterLeaveServiceHandler,
			hrv1.RegisterOvertimeServiceHandler,
			hrv1.RegisterContractServiceHandler,
			hrv1.RegisterAnnouncementServiceHandler,
			hrv1.RegisterLifecycleServiceHandler,
			hrv1.RegisterPayrollServiceHandler,
			hrv1.RegisterSelfServiceServiceHandler,
			hrv1.RegisterHrReportServiceHandler,
		}},
		{u.Sales, []handlerReg{
			salesv1.RegisterLeadServiceHandler,
			salesv1.RegisterPipelineServiceHandler,
			salesv1.RegisterQuotationServiceHandler,
			salesv1.RegisterBillingServiceHandler,
			salesv1.RegisterInsightServiceHandler,
			salesv1.RegisterAutomationServiceHandler,
			salesv1.RegisterFormServiceHandler,
			salesv1.RegisterContractServiceHandler,
			salesv1.RegisterPitchDeckServiceHandler,
			salesv1.RegisterOnboardingServiceHandler,
			salesv1.RegisterChatServiceHandler,
		}},
		{u.Recruitment, []handlerReg{
			recruitmentv1.RegisterJobOpeningServiceHandler,
			recruitmentv1.RegisterCandidateServiceHandler,
			recruitmentv1.RegisterPsikotesServiceHandler,
			recruitmentv1.RegisterInterviewServiceHandler,
			recruitmentv1.RegisterOfferServiceHandler,
			recruitmentv1.RegisterPublicRecruitmentServiceHandler,
		}},
		{u.Content, []handlerReg{
			contentv1.RegisterPlanServiceHandler,
			contentv1.RegisterItemServiceHandler,
			contentv1.RegisterDesignServiceHandler,
			contentv1.RegisterAssetServiceHandler,
			contentv1.RegisterReportServiceHandler,
		}},
		{u.Performance, []handlerReg{
			performancev1.RegisterKpiServiceHandler,
			performancev1.RegisterReviewServiceHandler,
			performancev1.RegisterFeedbackServiceHandler,
			performancev1.RegisterTaskServiceHandler,
			performancev1.RegisterLogbookServiceHandler,
			performancev1.RegisterRecapServiceHandler,
			performancev1.RegisterImprovementServiceHandler,
		}},
	})
}

// Byte downloads that have an HTTP route go through the gateway mux as raw
// files (see files.go). These routes cover multipart uploads, which the JSON
// transcoder would take as base64, and receipts, which have no HTTP route.

const maxUploadFile = 11 << 20 // 10 MB file plus multipart overhead

func (b *BrowserRoutes) mountErpFiles(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/finance/reimbursements/{id}/receipts", b.uploadReceipt)
	mux.HandleFunc("GET /api/finance/receipts/{receipt_id}", fileRoute(func(ctx context.Context, r *http.Request) (*financev1.ReceiptFile, error) {
		return financev1.NewReimbursementServiceClient(b.up.Finance).GetReceiptFile(ctx, &financev1.GetReceiptFileRequest{ReceiptId: r.PathValue("receipt_id")})
	}))
	id := func(r *http.Request) *salesv1.IdRequest { return &salesv1.IdRequest{Id: r.PathValue("id")} }
	mux.HandleFunc("GET /api/sales/quotations/{id}/pdf", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewQuotationServiceClient(b.up.Sales).RenderQuotationPdf(ctx, id(r))
	}))
	mux.HandleFunc("GET /api/sales/invoices/{id}/pdf", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewBillingServiceClient(b.up.Sales).RenderInvoicePdf(ctx, id(r))
	}))
	mux.HandleFunc("GET /api/sales/contracts/{id}/pdf", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewContractServiceClient(b.up.Sales).RenderContractPdf(ctx, &salesv1.RenderContractPdfRequest{Id: r.PathValue("id")})
	}))
	mux.HandleFunc("GET /api/public/sales/contracts/{token}/pdf", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewContractServiceClient(b.up.Sales).RenderContractPdf(ctx, &salesv1.RenderContractPdfRequest{Token: r.PathValue("token")})
	}))
	mux.HandleFunc("GET /api/sales/pitch-decks/{id}/file", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewPitchDeckServiceClient(b.up.Sales).GetPitchDeckFile(ctx, id(r))
	}))
	mux.HandleFunc("GET /api/sales/onboarding/assets/{id}", fileRoute(func(ctx context.Context, r *http.Request) (*salesv1.File, error) {
		return salesv1.NewOnboardingServiceClient(b.up.Sales).GetOnboardingAsset(ctx, id(r))
	}))
	mux.HandleFunc("POST /api/performance/tasks/occurrences/{occurrence_id}/attachments", b.uploadAttachment)
	mux.HandleFunc("POST /api/hr/employees/{employee_id}/documents", b.uploadEmployeeDocument)
}

// readUpload reads the multipart "file" field, capped at 10 MB.
func readUpload(w http.ResponseWriter, r *http.Request) (data []byte, name, mime string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadFile)
	f, fh, err := r.FormFile("file")
	if err != nil {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Pilih file (maks. 10 MB)."))
		return nil, "", "", false
	}
	defer f.Close()
	if data, err = io.ReadAll(f); err != nil {
		grpcx.WriteError(w, apperr.Wrap(apperr.Validation, "", err))
		return nil, "", "", false
	}
	return data, fh.Filename, fh.Header.Get("Content-Type"), true
}

func userCtx(r *http.Request) context.Context {
	return metadata.NewOutgoingContext(r.Context(), authWithCookie(r))
}

func (b *BrowserRoutes) uploadReceipt(w http.ResponseWriter, r *http.Request) {
	data, name, _, ok := readUpload(w, r)
	if !ok {
		return
	}
	res, err := financev1.NewReimbursementServiceClient(b.up.Finance).UploadReceipt(userCtx(r), &financev1.UploadReceiptRequest{
		Id: r.PathValue("id"), Filename: name, Data: data,
	})
	writeResult(w, res, err)
}

func (b *BrowserRoutes) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	data, name, mime, ok := readUpload(w, r)
	if !ok {
		return
	}
	res, err := performancev1.NewTaskServiceClient(b.up.Performance).AddAttachment(userCtx(r), &performancev1.AddAttachmentRequest{
		OccurrenceId: r.PathValue("occurrence_id"), Name: name, Mime: mime, Data: data,
	})
	writeResult(w, res, err)
}

func (b *BrowserRoutes) uploadEmployeeDocument(w http.ResponseWriter, r *http.Request) {
	data, name, _, ok := readUpload(w, r)
	if !ok {
		return
	}
	res, err := hrv1.NewPeopleServiceClient(b.up.HR).UploadEmployeeDocument(userCtx(r), &hrv1.UploadEmployeeDocumentRequest{
		EmployeeId: r.PathValue("employee_id"), DocType: r.FormValue("docType"), Name: name, Data: data,
	})
	writeResult(w, res, err)
}

// fileRoute serves an internal file RPC (no HTTP annotation) under the
// caller's token; the owning service decides who may read it.
func fileRoute[M proto.Message](fetch func(context.Context, *http.Request) (M, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := fetch(userCtx(r), r)
		if err != nil {
			grpcx.WriteError(w, grpcx.FromStatus(err))
			return
		}
		f, fields, ok := fileOf(msg)
		if !ok {
			grpcx.WriteError(w, apperr.New(apperr.Unknown, ""))
			return
		}
		w.Header().Set("Content-Type", fileMarshaler{}.ContentType(msg))
		_ = fileHeaders(r.Context(), w, msg)
		_, _ = w.Write(f.Get(fields.data).Bytes())
	}
}
