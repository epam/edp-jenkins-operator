package jenkins

import (
	"errors"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/stretchr/testify/require"
)

func TestClientBuilderMock_MakeNewClient(t *testing.T) {
	mk := ClientBuilderMock{}
	var owner *string
	mk.On("MakeNewClient", owner).Return(nil, errors.New("fatal mock")).Once()

	_, err := mk.MakeNewClient(nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fatal mock")

	jClient := ClientMock{}
	mk.On("MakeNewClient", owner).Return(&jClient, nil).Once()

	_, err = mk.MakeNewClient(nil, nil)
	require.NoError(t, err)
}

func TestClientMock_OneLiners(t *testing.T) {
	m := ClientMock{}

	m.On("AddRole", "rt", "name", "pattern", []string{""}).Return(nil)
	require.NoError(t, m.AddRole("rt", "name", "pattern", []string{""}))

	build := gojenkins.Build{}
	m.On("BuildIsRunning", &build).Return(false)
	require.False(t, m.BuildIsRunning(&build))

	m.On("RemoveRoles", "rt", []string{"rn"}).Return(nil)
	require.NoError(t, m.RemoveRoles("rt", []string{"rn"}))

	m.On("AssignRole", "rt", "rn", "subject").Return(nil)
	require.NoError(t, m.AssignRole("rt", "rn", "subject"))

	m.On("UnAssignRole", "rt", "rn", "s").Return(nil)
	require.NoError(t, m.UnAssignRole("rt", "rn", "s"))
}

func TestClientMock_GetJobByName(t *testing.T) {
	m := ClientMock{}
	m.On("GetJobByName", "name").Return(nil, errors.New("fatal"))

	_, err := m.GetJobByName("name")
	require.Error(t, err)

	m.On("GetJobByName", "job11").Return(&gojenkins.Job{}, nil)
	_, err = m.GetJobByName("job11")
	require.NoError(t, err)
}

func TestClientMock_BuildJob(t *testing.T) {
	m := ClientMock{}
	m.On("BuildJob", "job1", map[string]string{"foo": "bar"}).
		Return(nil, errors.New("fatal"))

	_, err := m.BuildJob("job1", map[string]string{"foo": "bar"})
	require.Error(t, err)

	var ret int64 = 10
	m.On("BuildJob", "job2", map[string]string{"foo": "bar"}).Return(&ret, nil)

	_, err = m.BuildJob("job2", map[string]string{"foo": "bar"})
	require.NoError(t, err)
}

func TestClientMock_GetLastBuild(t *testing.T) {
	m := ClientMock{}

	job1 := gojenkins.Job{}
	m.On("GetLastBuild", &job1).Return(nil, errors.New("fatal")).Once()

	_, err := m.GetLastBuild(&job1)
	require.Error(t, err)

	m.On("GetLastBuild", &job1).Return(&gojenkins.Build{}, nil)

	_, err = m.GetLastBuild(&job1)
	require.NoError(t, err)
}

func TestClientMock_GetRole(t *testing.T) {
	m := ClientMock{}

	m.On("GetRole", "rt", "rn").Return(nil, errors.New("fatal"))

	_, err := m.GetRole("rt", "rn")
	require.Error(t, err)

	m.On("GetRole", "rt1", "rn1").Return(&Role{}, nil)

	_, err = m.GetRole("rt1", "rn1")
	require.NoError(t, err)
}
