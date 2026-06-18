package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelError: "ERROR",
}

func (l Level) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return "INFO"
}

type Logger struct {
	mu    sync.Mutex
	level Level
	out   io.Writer
}

var defaultLogger = New(LevelInfo, os.Stdout)

func Default() *Logger {
	return defaultLogger
}

func SetDefault(l *Logger) {
	defaultLogger = l
}

func ParseLevel(value string) Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "error", "err":
		return LevelError
	case "debug", "verbose":
		return LevelDebug
	case "info", "":
		return LevelInfo
	default:
		return LevelInfo
	}
}

func New(level Level, out io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{level: level, out: out}
}

func NewFromConfig(level Level, logFile string) (*Logger, error) {
	logStdout := os.Getenv("LOG_STDOUT") != "0" && strings.ToLower(os.Getenv("LOG_STDOUT")) != "false"

	var out io.Writer = os.Stdout

	if logFile != "" {
		file, err := openLogFile(logFile)
		if err != nil {
			return nil, err
		}
		if logStdout {
			out = io.MultiWriter(os.Stdout, file)
		} else {
			out = file
		}
	}

	return New(level, out), nil
}

func openLogFile(logFile string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return file, nil
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

func (l *Logger) log(level Level, msg string, args ...any) {
	if level < l.level {
		return
	}

	line := formatLine(level, msg, args...)

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintln(l.out, line)
}

func formatLine(level Level, msg string, args ...any) string {
	if len(args) == 0 {
		return fmt.Sprintf("%s %s %s", time.Now().Format(time.RFC3339), level.String(), msg)
	}

	if len(args)%2 != 0 {
		args = append(args, "MISSING_VALUE")
	}

	pairs := make([]string, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%v=%v", args[i], args[i+1]))
	}

	return fmt.Sprintf("%s %s %s %s", time.Now().Format(time.RFC3339), level.String(), msg, strings.Join(pairs, " "))
}
