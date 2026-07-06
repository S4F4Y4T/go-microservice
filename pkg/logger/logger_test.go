package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
		{"  debug  ", slog.LevelDebug},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseLevel(tt.input); got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFromContextFallsBackToDefault(t *testing.T) {
	if got := FromContext(context.Background()); got != L() {
		t.Errorf("FromContext(bg) = %v, want default logger %v", got, L())
	}
	if got := FromContext(nil); got != L() { //nolint:staticcheck
		t.Errorf("FromContext(nil) = %v, want default logger", got)
	}
}

func TestWithContextRoundTrip(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(nil, nil))
	ctx := WithContext(context.Background(), custom)

	if got := FromContext(ctx); got != custom {
		t.Errorf("FromContext() = %v, want %v", got, custom)
	}
}
