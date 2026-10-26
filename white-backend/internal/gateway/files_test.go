package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
)

func TestFileMarshaler(t *testing.T) {
	m := fileMarshaler{&runtime.JSONPb{}}
	file := &hrv1.FileBytes{FileName: "slip gaji.pdf", ContentType: "application/pdf", Data: []byte("%PDF")}
	if got := m.ContentType(file); got != "application/pdf" {
		t.Fatalf("content type %q", got)
	}
	if b, _ := m.Marshal(file); string(b) != "%PDF" {
		t.Fatalf("body %q", b)
	}
	w := httptest.NewRecorder()
	_ = fileHeaders(t.Context(), w, file)
	if got := w.Header().Get("Content-Disposition"); got != "inline; filename*=UTF-8''slip%20gaji.pdf" {
		t.Fatalf("disposition %q", got)
	}

	plain := &hrv1.Department{Name: "SEO"}
	if got := m.ContentType(plain); got != "application/json" {
		t.Fatalf("plain content type %q", got)
	}
	if b, _ := m.Marshal(plain); string(b) == "" || b[0] != '{' {
		t.Fatalf("plain body %q", b)
	}
}
