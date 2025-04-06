package log

import (
	"fmt"
	"log/slog"
)

type SugaredLogger struct {
	*slog.Logger
}

func (s *SugaredLogger) With(args ...any) *SugaredLogger {
	return &SugaredLogger{Logger: s.Logger.With(args...)}
}

func (s *SugaredLogger) Errorf(format string, args ...any) {
	s.Logger.Error(fmt.Sprintf(format, args...))
}

func (s *SugaredLogger) Warnf(format string, args ...any) {
	s.Logger.Warn(fmt.Sprintf(format, args...))
}

func (s *SugaredLogger) Infof(format string, args ...any) {
	s.Logger.Info(fmt.Sprintf(format, args...))
}

func (s *SugaredLogger) Debugf(format string, args ...any) {
	s.Logger.Debug(fmt.Sprintf(format, args...))
}
