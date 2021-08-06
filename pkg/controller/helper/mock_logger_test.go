package helper

import (
	"testing"

	"github.com/pkg/errors"
)

func TestLoggerMock_Info(t *testing.T) {
	l := LoggerMock{}
	infoLogger := l.V(2)

	if l.LastInfo() != "" {
		t.Fatal("last info must be empty")
	}

	infoLogger.Info("test")

	if len(l.Infos()) == 0 {
		t.Fatal("infos array is not set")
	}

	if l.LastInfo() != "test" {
		t.Fatal("wrong value of info")
	}
}

func TestLoggerMock_Error(t *testing.T) {
	l := LoggerMock{}
	logger := l.WithValues("foo", "bar").WithName("test")
	if err := l.LastError(); err != nil {
		t.Fatal(err)
	}

	logger.Error(errors.New("fatal"), "err msg")

	err := l.LastError()
	if err == nil {
		t.Fatal("no error returned")
	}

	if !l.Enabled() {
		t.Fatal("logger must be enabled")
	}
}
