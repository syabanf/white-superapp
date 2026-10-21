// Package logging builds the structured logger every binary uses.
package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/pkg/config"
)

// New returns a JSON logger tagged with the service name. LOG_LEVEL picks the
// level (debug, info, warn, error); LOG_FORMAT=text switches to text output.
func New(service string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(config.Env("LOG_LEVEL", "info")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if config.Env("LOG_FORMAT", "json") == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(h).With("service", service)
}
