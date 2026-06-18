package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := map[string]Level{
		"error":   LevelError,
		"info":    LevelInfo,
		"debug":   LevelDebug,
		"verbose": LevelDebug,
		"":        LevelInfo,
		"unknown": LevelInfo,
	}

	for input, want := range tests {
		if got := ParseLevel(input); got != want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	logger.Debug("hidden")
	logger.Info("visible")
	logger.Error("visible too")

	output := buf.String()
	if strings.Contains(output, "hidden") {
		t.Fatal("debug message should be filtered at info level")
	}
	if !strings.Contains(output, "visible") || !strings.Contains(output, "visible too") {
		t.Fatalf("expected info and error messages, got %q", output)
	}
}

func TestLoggerStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelDebug, &buf)

	logger.Info("request", "method", "GET", "path", "/api/health", "status", 200)

	output := buf.String()
	for _, part := range []string{"method=GET", "path=/api/health", "status=200"} {
		if !strings.Contains(output, part) {
			t.Fatalf("expected %q in log line, got %q", part, output)
		}
	}
}
