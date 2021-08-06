package jenkins

import (
	"errors"
	"strings"
	"testing"

	"github.com/bndr/gojenkins"
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

func TestClientMock_OneLiners(t *testing.T) {
	m := ClientMock{}

	m.On("AddRole", "rt", "name", "pattern", []string{""}).Return(nil)
	if err := m.AddRole("rt", "name", "pattern", []string{""}); err != nil {
		t.Fatal(err)
	}

	build := gojenkins.Build{}
	m.On("BuildIsRunning", &build).Return(false)
	if m.BuildIsRunning(&build) {
		t.Fatal("build is running")
	}

	m.On("RemoveRoles", "rt", []string{"rn"}).Return(nil)
	if err := m.RemoveRoles("rt", []string{"rn"}); err != nil {
		t.Fatal(err)
	}

	m.On("AssignRole", "rt", "rn", "subject").Return(nil)
	if err := m.AssignRole("rt", "rn", "subject"); err != nil {
		t.Fatal(err)
	}

	m.On("UnAssignRole", "rt", "rn", "s").Return(nil)
	if err := m.UnAssignRole("rt", "rn", "s"); err != nil {
		t.Fatal(err)
	}
}

func TestClientMock_GetJobByName(t *testing.T) {
	m := ClientMock{}
	m.On("GetJobByName", "name").Return(nil, errors.New("fatal"))
	if _, err := m.GetJobByName("name"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("GetJobByName", "job11").Return(&gojenkins.Job{}, nil)
	if _, err := m.GetJobByName("job11"); err != nil {
		t.Fatal(err)
	}
}

func TestClientMock_BuildJob(t *testing.T) {
	m := ClientMock{}
	m.On("BuildJob", "job1", map[string]string{"foo": "bar"}).
		Return(nil, errors.New("fatal"))
	if _, err := m.BuildJob("job1", map[string]string{"foo": "bar"}); err == nil {
		t.Fatal("no error returned")
	}

	var ret int64 = 10
	m.On("BuildJob", "job2", map[string]string{"foo": "bar"}).Return(&ret, nil)
	if _, err := m.BuildJob("job2", map[string]string{"foo": "bar"}); err != nil {
		t.Fatal(err)
	}
}

func TestClientMock_GetLastBuild(t *testing.T) {
	m := ClientMock{}

	job1 := gojenkins.Job{}
	m.On("GetLastBuild", &job1).Return(nil, errors.New("fatal")).Once()
	if _, err := m.GetLastBuild(&job1); err == nil {
		t.Fatal("no error returned")
	}

	m.On("GetLastBuild", &job1).Return(&gojenkins.Build{}, nil)
	if _, err := m.GetLastBuild(&job1); err != nil {
		t.Fatal(err)
	}
}

func TestClientMock_GetRole(t *testing.T) {
	m := ClientMock{}

	m.On("GetRole", "rt", "rn").Return(nil, errors.New("fatal"))
	if _, err := m.GetRole("rt", "rn"); err == nil {
		t.Fatal("no error returned")
	}

	m.On("GetRole", "rt1", "rn1").Return(&Role{}, nil)
	if _, err := m.GetRole("rt1", "rn1"); err != nil {
		t.Fatal(err)
	}
}
