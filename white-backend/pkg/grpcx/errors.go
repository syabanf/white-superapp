package grpcx

import (
	"context"
	"errors"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

const errorDomain = "white"

var codeToGRPC = map[apperr.Code]codes.Code{
	apperr.Validation:      codes.InvalidArgument,
	apperr.Unauthenticated: codes.Unauthenticated,
	apperr.Unauthorized:    codes.PermissionDenied,
	apperr.NotFound:        codes.NotFound,
	apperr.Conflict:        codes.AlreadyExists,
	apperr.RateLimit:       codes.ResourceExhausted,
	apperr.TokenExpired:    codes.FailedPrecondition,
	apperr.Permission:      codes.PermissionDenied,
	apperr.Network:         codes.Unavailable,
	apperr.Unknown:         codes.Internal,
}

var grpcToCode = map[codes.Code]apperr.Code{
	codes.InvalidArgument:    apperr.Validation,
	codes.Unauthenticated:    apperr.Unauthenticated,
	codes.PermissionDenied:   apperr.Unauthorized,
	codes.NotFound:           apperr.NotFound,
	codes.AlreadyExists:      apperr.Conflict,
	codes.ResourceExhausted:  apperr.RateLimit,
	codes.FailedPrecondition: apperr.TokenExpired,
	codes.Unavailable:        apperr.Network,
	codes.DeadlineExceeded:   apperr.Network,
}

// ToStatus turns an application error into a gRPC status. The apperr.Code
// travels in ErrorInfo.Reason so the gateway can restore it; field errors
// travel as BadRequest violations.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "dibatalkan")
	}
	e := apperr.As(err)
	gc, ok := codeToGRPC[e.Code]
	if !ok {
		gc = codes.Internal
	}
	st := status.New(gc, e.Message)
	info := &errdetails.ErrorInfo{
		Reason:   string(e.Code),
		Domain:   errorDomain,
		Metadata: map[string]string{},
	}
	if e.Provider != "" {
		info.Metadata["provider"] = e.Provider
	}
	if e.RequiresReconnect {
		info.Metadata["requiresReconnect"] = "true"
	}
	details := []protoadapt.MessageV1{info}
	if len(e.FieldErrors) > 0 {
		br := &errdetails.BadRequest{}
		for field, msgs := range e.FieldErrors {
			for _, m := range msgs {
				br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{Field: field, Description: m})
			}
		}
		details = append(details, br)
	}
	withDetails, derr := st.WithDetails(details...)
	if derr != nil {
		return st.Err()
	}
	return withDetails.Err()
}

// FromStatus is the inverse used by gRPC clients and the gateway.
func FromStatus(err error) *apperr.Error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return apperr.As(err)
	}
	code, known := grpcToCode[st.Code()]
	if !known {
		code = apperr.Unknown
	}
	e := apperr.New(code, st.Message())
	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.ErrorInfo:
			if v.Domain == errorDomain && v.Reason != "" {
				e.Code = apperr.Code(v.Reason)
			}
			e.Provider = v.Metadata["provider"]
			e.RequiresReconnect = v.Metadata["requiresReconnect"] == "true"
		case *errdetails.BadRequest:
			e.FieldErrors = map[string][]string{}
			for _, fv := range v.FieldViolations {
				e.FieldErrors[fv.Field] = append(e.FieldErrors[fv.Field], fv.Description)
			}
		}
	}
	return e
}
