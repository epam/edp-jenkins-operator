package helper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerMock_Info(t *testing.T) {
	l := LoggerMock{}
	infoLogger := l.V(2)

	require.Emptyf(t, l.LastInfo(), "last info must be empty")

	infoLogger.Info("test")

	require.NotEmptyf(t, l.Infos(), "infos array is not set")

	require.Equalf(t, "test", l.LastInfo(), "wrong value of info")
}

func TestLoggerMock_Error(t *testing.T) {
	l := LoggerMock{}
	logger := l.WithValues("foo", "bar").WithName("test")
	require.NoError(t, l.LastError())

	logger.Error(errors.New("fatal"), "err msg")

	require.Error(t, l.LastError())

	require.Truef(t, l.Enabled(), "logger must be enabled")
}
