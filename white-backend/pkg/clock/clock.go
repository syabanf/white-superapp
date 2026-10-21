// Package clock lets use cases ask for "now" without reaching for time.Now,
// so tests can pin the day.
package clock

import "time"

type Clock interface {
	Now() time.Time
}

type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed always returns the same instant.
type Fixed struct{ T time.Time }

func (f Fixed) Now() time.Time { return f.T }

func At(s string) Fixed {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return Fixed{T: t.UTC()}
}
