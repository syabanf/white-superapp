package grpcx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusRoundTripKeepsCodeFieldsAndProvider(t *testing.T) {
	in := apperr.Field("slug", "Slug sudah dipakai")
	in.Provider = "meta"
	in.RequiresReconnect = true
	st := ToStatus(in)
	if s, _ := status.FromError(st); s.Code() != codes.InvalidArgument {
		t.Fatalf("grpc code = %v", s.Code())
	}
	out := FromStatus(st)
	if out.Code != apperr.Validation || out.Message != "Slug sudah dipakai" || out.FieldErrors["slug"][0] != "Slug sudah dipakai" || out.Provider != "meta" || !out.RequiresReconnect {
		t.Fatalf("%+v", out)
	}
	// Codes that share a gRPC code survive through ErrorInfo.
	if FromStatus(ToStatus(apperr.New(apperr.Permission, ""))).Code != apperr.Permission {
		t.Fatal("PERMISSION collapsed into UNAUTHORIZED")
	}
	if FromStatus(ToStatus(errors.New("boom"))).Code != apperr.Unknown {
		t.Fatal("plain error must be UNKNOWN")
	}
}

func TestWriteErrorShape(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, apperr.New(apperr.RateLimit, ""))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatal(rec.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.OK || body.Code != apperr.RateLimit || body.Error != apperr.Message(apperr.RateLimit) {
		t.Fatalf("%+v %v", body, err)
	}
}
