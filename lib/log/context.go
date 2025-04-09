package log

import (
	"context"
	"log/slog"
	"os"
	"time"
)

var defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	AddSource:   false,
	Level:       slog.LevelDebug,
	ReplaceAttr: replaceAttr,
}))

type loggerCtxKey struct{}

type Config struct {
	Level string `yaml:"level"`
	Text  bool   `yaml:"text"`
}

func With(parent context.Context, fields ...any) context.Context {
	return NewContext(parent, FromContext(parent).With(fields...))
}

func NewContext(parent context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(parent, loggerCtxKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger)
	if !ok {
		l = defaultLogger
	}
	return l
}

func FromContexts(ctx context.Context) *SugaredLogger {
	return &SugaredLogger{Logger: FromContext(ctx)}
}

func Default() *slog.Logger {
	return defaultLogger
}

func Defaults() *SugaredLogger {
	return &SugaredLogger{Logger: defaultLogger}
}

func New(conf Config) *slog.Logger {
	var level = slog.LevelInfo
	_ = level.UnmarshalText([]byte(conf.Level))
	opts := &slog.HandlerOptions{
		AddSource:   false,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}
	var h slog.Handler
	if conf.Text {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	defaultLogger = slog.New(h)
	return defaultLogger
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "time" {
		a.Value = slog.StringValue(a.Value.Time().Format(time.TimeOnly))
	}
	return a
}
