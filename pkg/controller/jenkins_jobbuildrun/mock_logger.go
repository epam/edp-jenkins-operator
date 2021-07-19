package jenkins_jobbuildrun

import "github.com/go-logr/logr"

type logger struct {
	errors []error
}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {}

func (l *logger) Enabled() bool { return true }

func (l *logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.errors = append(l.errors, err)
}

func (l *logger) LastError() error {
	if len(l.errors) == 0 {
		return nil
	}

	return l.errors[len(l.errors)-1]
}

func (l *logger) V(level int) logr.InfoLogger {
	return l
}

func (l *logger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return l
}

func (l *logger) WithName(name string) logr.Logger {
	return l
}
