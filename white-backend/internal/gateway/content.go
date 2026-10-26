package gateway

import (
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
)

// Content uploads: design versions (several files) and brand assets, 25 MB
// per file. The content service re-checks size and type.
const (
	maxContentFile    = 25 << 20
	maxContentRequest = 4*maxContentFile + 1<<20
)

func (b *BrowserRoutes) mountContentFiles(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/content/clients/{client_id}/design-tasks/{task_id}/files", b.uploadDesignFiles)
	mux.HandleFunc("POST /api/content/clients/{client_id}/assets", b.createAsset)
	mux.HandleFunc("POST /api/content/clients/{client_id}/assets/{asset_id}/versions", b.uploadAssetVersion)
}

// contentForm parses the multipart body and returns the files in `field`.
func contentForm(w http.ResponseWriter, r *http.Request, field string) ([]*contentv1.UploadFile, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxContentRequest)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Ukuran file melebihi 25 MB."))
		return nil, false
	}
	var out []*contentv1.UploadFile
	for _, fh := range r.MultipartForm.File[field] {
		data, err := readPart(fh)
		if err != nil {
			grpcx.WriteError(w, apperr.Wrap(apperr.Validation, "", err))
			return nil, false
		}
		out = append(out, &contentv1.UploadFile{Filename: fh.Filename, Content: data})
	}
	if len(out) == 0 {
		grpcx.WriteError(w, apperr.New(apperr.Validation, "Pilih file terlebih dahulu."))
		return nil, false
	}
	return out, true
}

func readPart(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (b *BrowserRoutes) uploadDesignFiles(w http.ResponseWriter, r *http.Request) {
	files, ok := contentForm(w, r, "files")
	if !ok {
		return
	}
	res, err := contentv1.NewDesignServiceClient(b.up.Content).UploadDesignFiles(userCtx(r), &contentv1.UploadDesignFilesRequest{
		ClientId: r.PathValue("client_id"), TaskId: r.PathValue("task_id"), Files: files, Note: r.FormValue("note"),
	})
	writeResult(w, res, err)
}

func (b *BrowserRoutes) createAsset(w http.ResponseWriter, r *http.Request) {
	files, ok := contentForm(w, r, "file")
	if !ok {
		return
	}
	var tags []string
	for _, t := range strings.Split(r.FormValue("tags"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	res, err := contentv1.NewAssetServiceClient(b.up.Content).CreateAsset(userCtx(r), &contentv1.CreateAssetRequest{
		ClientId:    r.PathValue("client_id"),
		Name:        r.FormValue("name"),
		Kind:        contentv1.AssetKind(contentv1.AssetKind_value[r.FormValue("kind")]),
		Description: r.FormValue("description"),
		Tags:        tags,
		FolderId:    r.FormValue("folderId"),
		File:        files[0],
	})
	writeResult(w, res, err)
}

func (b *BrowserRoutes) uploadAssetVersion(w http.ResponseWriter, r *http.Request) {
	files, ok := contentForm(w, r, "file")
	if !ok {
		return
	}
	res, err := contentv1.NewAssetServiceClient(b.up.Content).UploadAssetVersion(userCtx(r), &contentv1.UploadAssetVersionRequest{
		ClientId: r.PathValue("client_id"), AssetId: r.PathValue("asset_id"), File: files[0], Note: r.FormValue("note"),
	})
	writeResult(w, res, err)
}
