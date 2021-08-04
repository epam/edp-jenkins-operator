package jenkins

import (
	"errors"
	"strings"
	"testing"
)

func TestClientBuilderMock_MakeNewClient(t *testing.T) {
	mk := ClientBuilderMock{}
	var owner *string
	mk.On("MakeNewClient", owner).Return(nil, errors.New("fatal mock")).Once()
	_, err := mk.MakeNewClient(nil, nil)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "fatal mock") {
		t.Log(err)
		t.Fatal("wrong error returned")
	}

	jClient := ClientMock{}
	mk.On("MakeNewClient", owner).Return(&jClient, nil).Once()
	if _, err := mk.MakeNewClient(nil, nil); err != nil {
		t.Fatal(err)
	}
}
