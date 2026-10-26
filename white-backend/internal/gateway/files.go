package gateway

import (
	"context"
	"net/http"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// File responses (PDF payslips, XLSX exports, attachments) share one shape:
// bytes data, string content_type and an optional file name. The gateway
// sends those as raw bytes instead of base64 JSON, so a plain link downloads
// them.

type fileFields struct {
	data, contentType, name protoreflect.FieldDescriptor
}

func fileOf(v any) (protoreflect.Message, fileFields, bool) {
	m, ok := v.(proto.Message)
	if !ok {
		return nil, fileFields{}, false
	}
	r := m.ProtoReflect()
	fs := r.Descriptor().Fields()
	f := fileFields{data: fs.ByName("data"), contentType: fs.ByName("content_type"), name: fs.ByName("file_name")}
	if f.name == nil {
		f.name = fs.ByName("filename")
	}
	if f.data == nil || f.data.Kind() != protoreflect.BytesKind || f.contentType == nil || f.contentType.Kind() != protoreflect.StringKind {
		return nil, fileFields{}, false
	}
	return r, f, true
}

type fileMarshaler struct{ runtime.Marshaler }

func (m fileMarshaler) ContentType(v any) string {
	if r, f, ok := fileOf(v); ok {
		if ct := r.Get(f.contentType).String(); ct != "" {
			return ct
		}
		return "application/octet-stream"
	}
	return m.Marshaler.ContentType(v)
}

func (m fileMarshaler) Marshal(v any) ([]byte, error) {
	if r, f, ok := fileOf(v); ok {
		return r.Get(f.data).Bytes(), nil
	}
	return m.Marshaler.Marshal(v)
}

// fileHeaders names the download and keeps private files out of shared caches.
func fileHeaders(_ context.Context, w http.ResponseWriter, msg proto.Message) error {
	r, f, ok := fileOf(msg)
	if !ok {
		return nil
	}
	name := ""
	if f.name != nil {
		name = r.Get(f.name).String()
	}
	w.Header().Set("Content-Disposition", inlineDisposition(name))
	w.Header().Set("Cache-Control", "private, no-cache")
	return nil
}

func inlineDisposition(filename string) string {
	if filename == "" {
		return "inline"
	}
	return "inline; filename*=UTF-8''" + url.PathEscape(filename)
}
