package jenkins

import (
	"github.com/bndr/gojenkins"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClientMock struct {
	mock.Mock
}

func (j *ClientMock) GetJobByName(jobName string) (*gojenkins.Job, error) {
	called := j.Called(jobName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gojenkins.Job), nil
}

func (j *ClientMock) BuildJob(jobName string, parameters map[string]string) (*int64, error) {
	called := j.Called(jobName, parameters)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*int64), nil
}

func (j *ClientMock) GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error) {
	called := j.Called(job)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gojenkins.Build), nil
}

func (j *ClientMock) BuildIsRunning(build *gojenkins.Build) bool {
	return j.Called(build).Bool(0)
}

func (j *ClientMock) AddRole(roleType, name, pattern string, permissions []string) error {
	return j.Called(roleType, name, pattern, permissions).Error(0)
}

func (j *ClientMock) RemoveRoles(roleType string, roleNames []string) error {
	return j.Called(roleType, roleNames).Error(0)
}

func (j *ClientMock) AssignRole(roleType, roleName, subject string) error {
	return j.Called(roleType, roleName, subject).Error(0)
}

func (j *ClientMock) GetRole(roleType, roleName string) (*Role, error) {
	called := j.Called(roleType, roleName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*Role), nil
}

func (j *ClientMock) UnAssignRole(roleType, roleName, subject string) error {
	return j.Called(roleType, roleName, subject).Error(0)
}

type ClientBuilderMock struct {
	mock.Mock
}

func (j *ClientBuilderMock) MakeNewClient(om *metav1.ObjectMeta, ownerName *string) (ClientInterface, error) {
	called := j.Called(ownerName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(ClientInterface), nil
}
