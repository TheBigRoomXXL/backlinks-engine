package telemetry

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
)

func init() {
	s, _ := settings.New()
	// TODO: append instead of tunc once tests app is stable
	logFile, err := os.OpenFile(s.LOG_PATH, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		log.Fatal("failed to create log file: ", err)
	}
	handler := NewHandlerWithMetrics(slog.NewTextHandler(logFile, nil))
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

type HandlerWithMetrics struct {
	handler slog.Handler
}

// Just a simple wrapper arond any slog.Handler to count the number of errors and warnings
func NewHandlerWithMetrics(handler slog.Handler) *HandlerWithMetrics {
	return &HandlerWithMetrics{
		handler: handler,
	}
}

func (h *HandlerWithMetrics) Handle(ctx context.Context, record slog.Record) error {
	if record.Level == slog.LevelError {
		Errors.Add(1)
	} else if record.Level == slog.LevelWarn {
		Warnings.Add(1)
	}
	return h.handler.Handle(ctx, record)
}

func (h *HandlerWithMetrics) Enabled(ctx context.Context, l slog.Level) bool {
	return h.handler.Enabled(ctx, l)
}

func (h *HandlerWithMetrics) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := h.handler.WithAttrs(attrs)
	return &HandlerWithMetrics{
		handler: newHandler,
	}
}

func (h *HandlerWithMetrics) WithGroup(name string) slog.Handler {
	newHandler := h.handler.WithGroup(name)
	return &HandlerWithMetrics{
		handler: newHandler,
	}
}
