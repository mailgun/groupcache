package groupcache

import (
	"github.com/sirupsen/logrus"
)

// Logger is a minimal interface that will allow us to use structured loggers,
// including (but not limited to) logrus.
type Logger interface {
	// Error logging level
	Error() Logger

	// Warn logging level
	Warn() Logger

	// Info logging level
	Info() Logger

	// Debug logging level
	Debug() Logger

	// ErrorField is a field with an error value
	ErrorField(label string, err error) Logger

	// StringField is a field with a string value
	StringField(label string, val string) Logger

	// WithFields is willy-nilly key value pairs.
	WithFields(fields map[string]interface{}) Logger

	// Printf is called last to emit the log at the given
	// level.
	Printf(format string, args ...interface{})
}

// LogrusLogger is an implementation of Logger that wraps logrus... who knew?
type LogrusLogger struct {
	Entry *logrus.Entry
	level logrus.Level
}

func (l LogrusLogger) Info() Logger {
	return LogrusLogger{
		Entry: l.Entry,
		level: logrus.InfoLevel,
	}
}

func (l LogrusLogger) Debug() Logger {
	return LogrusLogger{
		Entry: l.Entry,
		level: logrus.DebugLevel,
	}
}

func (l LogrusLogger) Warn() Logger {
	return LogrusLogger{
		Entry: l.Entry,
		level: logrus.WarnLevel,
	}
}

func (l LogrusLogger) Error() Logger {
	return LogrusLogger{
		Entry: l.Entry,
		level: logrus.ErrorLevel,
	}
}

func (l LogrusLogger) WithFields(fields map[string]interface{}) Logger {
	return LogrusLogger{
		Entry: l.Entry.WithFields(fields),
		level: l.level,
	}
}

// ErrorField - create a field for an error
func (l LogrusLogger) ErrorField(label string, err error) Logger {
	return LogrusLogger{
		Entry: l.Entry.WithField(label, err),
		level: l.level,
	}

}

// StringField - create a field for a string.
func (l LogrusLogger) StringField(label string, val string) Logger {
	return LogrusLogger{
		Entry: l.Entry.WithField(label, val),
		level: l.level,
	}
}

func (l LogrusLogger) Printf(format string, args ...interface{}) {
	l.Entry.Logf(l.level, format, args...)
}
