package helper

import "testing"

func TestLoggerMock_Info(t *testing.T) {
	l := LoggerMock{}

	if l.LastInfo() != "" {
		t.Fatal("last info must be empty")
	}

	l.Info("test")

	if len(l.Infos()) == 0 {
		t.Fatal("infos array is not set")
	}

	if l.LastInfo() != "test" {
		t.Fatal("wrong value of info")
	}
}
