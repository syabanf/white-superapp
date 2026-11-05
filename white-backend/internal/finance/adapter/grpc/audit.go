package grpcapi

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

const auditBodyLimit = 20_000

var secretField = regexp.MustCompile(`(?i)password|token|secret|api_?key`)

// AuditInterceptor records every Finance call that can change data, read from its
// HTTP route, so a new RPC is covered without touching this file. It runs after the
// call: a failed write goes to the log and does not undo the change.
func AuditInterceptor(store domain.AuditRepo, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if !isFinanceChange(info.FullMethod) {
			return resp, err
		}
		e := domain.AuditEntry{Action: info.FullMethod[strings.LastIndex(info.FullMethod, "/")+1:], Result: domain.AuditOK}
		if p, ok := authn.FromContext(ctx); ok {
			e.ActorID, e.ActorName = p.UserID, p.Name
		}
		if err != nil {
			e.Result, e.Message = domain.AuditRefused, apperr.As(err).Message
		}
		if m, ok := req.(proto.Message); ok {
			e.Request, e.DocID = auditBody(m), docField(m.ProtoReflect(), "id", "invoice_id", "advance_id", "payment_id", "line_id")
		}
		if m, ok := resp.(proto.Message); ok && err == nil {
			e.Response = auditBody(m)
			if e.DocID == "" {
				e.DocID = docField(m.ProtoReflect(), "id")
			}
			e.DocNo = docNumber(m.ProtoReflect(), 2)
		}
		// The caller may have gone away, and the change still needs its record.
		if werr := store.InsertAuditEntry(context.WithoutCancel(ctx), e); werr != nil && logger != nil {
			logger.Error("audit trail", "action", e.Action, "err", werr)
		}
		return resp, err
	}
}

// isFinanceChange reads the route: everything except GET changes data.
func isFinanceChange(fullMethod string) bool {
	if !strings.HasPrefix(fullMethod, "/white.finance.") {
		return false
	}
	name := protoreflect.FullName(strings.ReplaceAll(strings.TrimPrefix(fullMethod, "/"), "/", "."))
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return true
	}
	rule, _ := proto.GetExtension(d.Options(), annotations.E_Http).(*annotations.HttpRule)
	return rule == nil || rule.GetGet() == ""
}

// auditBody is the message as JSON without file contents or secrets.
func auditBody(m proto.Message) string {
	clean := proto.Clone(m)
	scrub(clean.ProtoReflect())
	// Default values stay in, so switching a setting off records false.
	body, err := protojson.MarshalOptions{EmitDefaultValues: true}.Marshal(clean)
	if err != nil {
		return ""
	}
	if len(body) > auditBodyLimit {
		return string(body[:auditBodyLimit]) + "…"
	}
	return string(body)
}

func scrub(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.Kind() == protoreflect.BytesKind || secretField.MatchString(string(fd.Name())):
			m.Clear(fd)
		case fd.IsMap():
		case fd.Kind() == protoreflect.MessageKind && fd.IsList():
			for i := range v.List().Len() {
				scrub(v.List().Get(i).Message())
			}
		case fd.Kind() == protoreflect.MessageKind:
			scrub(v.Message())
		}
		return true
	})
}

func docField(m protoreflect.Message, names ...string) string {
	fields := m.Descriptor().Fields()
	for _, n := range names {
		if fd := fields.ByName(protoreflect.Name(n)); fd != nil && fd.Kind() == protoreflect.StringKind {
			if v := m.Get(fd).String(); v != "" {
				return v
			}
		}
	}
	return ""
}

// docNumber finds the document number a response carries, such as invoice_no, and
// looks inside wrappers like a payment result.
func docNumber(m protoreflect.Message, depth int) string {
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		if fd.Kind() == protoreflect.StringKind && !fd.IsList() && strings.HasSuffix(string(fd.Name()), "_no") {
			if v := m.Get(fd).String(); v != "" {
				return v
			}
		}
	}
	if depth == 0 {
		return ""
	}
	for i := range fields.Len() {
		fd := fields.Get(i)
		if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() && m.Has(fd) {
			if v := docNumber(m.Get(fd).Message(), depth-1); v != "" {
				return v
			}
		}
	}
	return ""
}

func auditFilter(r *financev1.ListAuditTrailRequest) domain.AuditFilter {
	page, size := max(int(r.Page), 1), int(r.PageSize)
	if size < 1 || size > 200 {
		size = 50
	}
	return domain.AuditFilter{Search: r.Q, Result: r.Result, From: r.DateFrom, To: r.DateTo, Limit: size, Offset: (page - 1) * size}
}

func (s *periodEndServer) ListAuditTrail(ctx context.Context, r *financev1.ListAuditTrailRequest) (*financev1.ListAuditTrailResponse, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	rows, total, err := s.svc.Ledgers.ListAuditTrail(ctx, auditFilter(r))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListAuditTrailResponse{Total: int32(total)}
	for _, e := range rows {
		out.Entries = append(out.Entries, &financev1.AuditEntry{Id: e.ID, At: e.At, ActorId: e.ActorID, ActorName: e.ActorName, Action: e.Action, DocId: e.DocID, DocNo: e.DocNo, Result: e.Result, Message: e.Message, Request: e.Request, Response: e.Response})
	}
	return out, nil
}

func (s *periodEndServer) ExportAuditTrail(ctx context.Context, r *financev1.ListAuditTrailRequest) (*financev1.TaxExportFile, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	data, err := s.svc.Ledgers.ExportAuditTrail(ctx, auditFilter(r))
	if err != nil {
		return nil, err
	}
	return &financev1.TaxExportFile{FileName: "audit-trail.csv", ContentType: "text/csv; charset=utf-8", Data: data}, nil
}
