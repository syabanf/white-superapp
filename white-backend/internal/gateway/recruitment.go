package gateway

import (
	"net/http"
	"strconv"

	recruitmentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/protobuf/proto"
)

// Recruitment uploads: the public apply form and psikotes drawings (no
// login; the service rate-limits by x-forwarded-for and token), and CVs
// added by staff.
func (b *BrowserRoutes) mountRecruitmentFiles(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/recruitment/public/jobs/{slug}/apply", b.applyJob)
	mux.HandleFunc("POST /api/recruitment/public/psikotes/{token}/tests/{test_id}/upload", b.uploadDrawing)
	mux.HandleFunc("POST /api/recruitment/candidates/{id}/cv", b.uploadCv)
}

func fileUpload(w http.ResponseWriter, r *http.Request) (*recruitmentv1.FileUpload, bool) {
	data, name, mime, ok := readUpload(w, r)
	if !ok {
		return nil, false
	}
	return &recruitmentv1.FileUpload{FileName: name, ContentType: mime, Data: data}, true
}

func writeResult(w http.ResponseWriter, res proto.Message, err error) {
	if err != nil {
		grpcx.WriteError(w, grpcx.FromStatus(err))
		return
	}
	writeProtoJSON(w, res)
}

func (b *BrowserRoutes) applyJob(w http.ResponseWriter, r *http.Request) {
	cv, ok := fileUpload(w, r)
	if !ok {
		return
	}
	req := &recruitmentv1.ApplyRequest{
		Slug:           r.PathValue("slug"),
		FullName:       r.FormValue("fullName"),
		Email:          r.FormValue("email"),
		Phone:          r.FormValue("phone"),
		Domicile:       r.FormValue("domicile"),
		Source:         r.FormValue("source"),
		LastExperience: r.FormValue("lastExperience"),
		LastEducation:  r.FormValue("lastEducation"),
		Availability:   r.FormValue("availability"),
		Notes:          r.FormValue("notes"),
		Cv:             cv,
	}
	if v, err := strconv.ParseInt(r.FormValue("expectedSalary"), 10, 64); err == nil {
		req.ExpectedSalary = &v
	}
	res, err := recruitmentv1.NewPublicRecruitmentServiceClient(b.up.Recruitment).Apply(userCtx(r), req)
	writeResult(w, res, err)
}

func (b *BrowserRoutes) uploadDrawing(w http.ResponseWriter, r *http.Request) {
	file, ok := fileUpload(w, r)
	if !ok {
		return
	}
	res, err := recruitmentv1.NewPublicRecruitmentServiceClient(b.up.Recruitment).UploadPsikotesDrawing(userCtx(r), &recruitmentv1.UploadDrawingRequest{
		Token: r.PathValue("token"), TestId: r.PathValue("test_id"), File: file,
	})
	writeResult(w, res, err)
}

func (b *BrowserRoutes) uploadCv(w http.ResponseWriter, r *http.Request) {
	file, ok := fileUpload(w, r)
	if !ok {
		return
	}
	res, err := recruitmentv1.NewCandidateServiceClient(b.up.Recruitment).UploadCv(userCtx(r), &recruitmentv1.UploadCvRequest{Id: r.PathValue("id"), File: file})
	writeResult(w, res, err)
}
