// Package validate collects field errors the way the zod schemas did: one
// message list per top-level field, surfaced as one VALIDATION error.
package validate

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

type Validator struct {
	fields map[string][]string
	order  []string
}

func New() *Validator { return &Validator{fields: map[string][]string{}} }

func (v *Validator) Add(field, message string) *Validator {
	if _, seen := v.fields[field]; !seen {
		v.order = append(v.order, field)
	}
	v.fields[field] = append(v.fields[field], message)
	return v
}

func (v *Validator) HasErrors() bool { return len(v.fields) > 0 }

// Err returns nil when every check passed. The top-level message is the
// first field message, which is what the forms displayed.
func (v *Validator) Err() error {
	if !v.HasErrors() {
		return nil
	}
	first := v.fields[v.order[0]][0]
	return apperr.New(apperr.Validation, first).Fields(v.fields)
}

func (v *Validator) Required(field, value, message string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.Add(field, message)
	}
	return v
}

func (v *Validator) MinLen(field, value string, n int, message string) *Validator {
	if utf8.RuneCountInString(value) < n {
		v.Add(field, message)
	}
	return v
}

func (v *Validator) MaxLen(field, value string, n int, message string) *Validator {
	if utf8.RuneCountInString(value) > n {
		v.Add(field, message)
	}
	return v
}

func (v *Validator) Match(field, value string, re *regexp.Regexp, message string) *Validator {
	if !re.MatchString(value) {
		v.Add(field, message)
	}
	return v
}

func (v *Validator) OneOf(field, value string, allowed []string, message string) *Validator {
	for _, a := range allowed {
		if a == value {
			return v
		}
	}
	return v.Add(field, message)
}

// HTTPURL accepts http(s) URLs with a host. Empty values pass; combine with
// Required when the field is mandatory.
func (v *Validator) HTTPURL(field, value, message string) *Validator {
	if value == "" {
		return v
	}
	if !IsHTTPURL(value) {
		v.Add(field, message)
	}
	return v
}

func IsHTTPURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func (v *Validator) Email(field, value, message string) *Validator {
	if !emailRe.MatchString(value) {
		v.Add(field, message)
	}
	return v
}

// Trim collapses surrounding whitespace; nil-safe helper for optional fields.
func Trim(s string) string { return strings.TrimSpace(s) }

// EmptyToNil returns nil for a blank string, which maps to SQL NULL.
func EmptyToNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
