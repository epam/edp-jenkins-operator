package helper

import "github.com/go-logr/logr"

type LoggerMock struct {
	errors []error
	infos  []string
}

func (l *LoggerMock) Info(msg string, keysAndValues ...interface{}) {
	l.infos = append(l.infos, msg)
}

func (l *LoggerMock) Enabled() bool { return true }

func (l *LoggerMock) Error(err error, msg string, keysAndValues ...interface{}) {
	l.errors = append(l.errors, err)
}

func (l *LoggerMock) LastError() error {
	if len(l.errors) == 0 {
		return nil
	}

	return l.errors[len(l.errors)-1]
}

func (l *LoggerMock) Infos() []string {
	return l.infos
}

func (l *LoggerMock) LastInfo() string {
	if len(l.infos) == 0 {
		return ""
	}

	return l.infos[len(l.infos)-1]
}

func (l *LoggerMock) V(level int) logr.Logger {
	return l
}

func (l *LoggerMock) WithValues(keysAndValues ...interface{}) logr.Logger {
	return l
}

func (l *LoggerMock) WithName(name string) logr.Logger {
	return l
}
