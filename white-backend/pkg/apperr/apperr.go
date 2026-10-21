// Package apperr is the one error vocabulary every service speaks. Use cases
// return *Error; the gRPC layer maps Code to a status code and carries the
// message and field errors as details so the gateway can hand the UI the
// same {code, error, fieldErrors} shape the Next.js server actions used.
package apperr

import (
	"errors"
	"fmt"
)

type Code string

const (
	Validation      Code = "VALIDATION"
	Unauthenticated Code = "UNAUTHENTICATED"
	Unauthorized    Code = "UNAUTHORIZED"
	NotFound        Code = "NOT_FOUND"
	Conflict        Code = "CONFLICT"
	RateLimit       Code = "RATE_LIMIT"
	TokenExpired    Code = "TOKEN_EXPIRED"
	Permission      Code = "PERMISSION"
	Network         Code = "NETWORK"
	Unknown         Code = "UNKNOWN"
)

// Messages shown to users (UI copy is Bahasa Indonesia).
var messages = map[Code]string{
	Validation:      "Periksa kembali data yang dimasukkan.",
	Unauthenticated: "Silakan masuk terlebih dahulu.",
	Unauthorized:    "Anda tidak punya akses.",
	NotFound:        "Data tidak ditemukan.",
	Conflict:        "Data sudah ada.",
	RateLimit:       "Batas permintaan API tercapai. Coba lagi beberapa menit.",
	TokenExpired:    "Koneksi akun kedaluwarsa. Hubungkan ulang akun.",
	Permission:      "Izin tidak mencukupi. Periksa akses akun yang dihubungkan.",
	Network:         "Gangguan jaringan. Periksa koneksi dan coba lagi.",
	Unknown:         "Terjadi kesalahan tak terduga.",
}

func Message(code Code) string {
	if m, ok := messages[code]; ok {
		return m
	}
	return messages[Unknown]
}

type Error struct {
	Code        Code
	Message     string
	FieldErrors map[string][]string
	// Provider that produced the error (meta, google, dataforseo, ...), when any.
	Provider string
	// RequiresReconnect tells the UI to offer "Hubungkan ulang".
	RequiresReconnect bool
	cause             error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// New builds an error with a custom message. An empty message falls back to
// the default copy for the code.
func New(code Code, message string) *Error {
	if message == "" {
		message = Message(code)
	}
	return &Error{Code: code, Message: message}
}

func Newf(code Code, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap keeps the cause for logs while the user sees `message`.
func Wrap(code Code, message string, cause error) *Error {
	e := New(code, message)
	e.cause = cause
	return e
}

// Fields attaches per-field validation messages (key = field name).
func (e *Error) Fields(fields map[string][]string) *Error {
	e.FieldErrors = fields
	return e
}

// Field is the one-field shortcut: the error message doubles as the field message.
func Field(field, message string) *Error {
	return New(Validation, message).Fields(map[string][]string{field: {message}})
}

// FromProvider mirrors the old ProviderError: reconnect is implied for
// expired tokens and missing permissions.
func FromProvider(provider string, code Code, message string, cause error) *Error {
	e := Wrap(code, message, cause)
	e.Provider = provider
	e.RequiresReconnect = code == TokenExpired || code == Permission
	return e
}

// As extracts an *Error from any error chain. Plain errors become Unknown.
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Wrap(Unknown, "", err)
}

func Is(err error, code Code) bool {
	e := As(err)
	return e != nil && e.Code == code
}
