package jenkins_jobbuildrun

import (
	"github.com/bndr/gojenkins"
	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/stretchr/testify/mock"
)

type jenkinsClientMock struct {
	mock.Mock
}

func (j *jenkinsClientMock) GetJobByName(jobName string) (*gojenkins.Job, error) {
	called := j.Called(jobName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gojenkins.Job), nil
}

func (j *jenkinsClientMock) BuildJob(jobName string, parameters map[string]string) (*int64, error) {
	called := j.Called(jobName, parameters)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*int64), nil
}

func (j *jenkinsClientMock) GetLastBuild(job *gojenkins.Job) (*gojenkins.Build, error) {
	called := j.Called(job)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gojenkins.Build), nil
}

func (j *jenkinsClientMock) BuildIsRunning(build *gojenkins.Build) bool {
	return j.Called(build).Bool(0)
}

type jenkinsClientBuilderMock struct {
	mock.Mock
}

func (j *jenkinsClientBuilderMock) MakeNewJenkinsClient(jf *v2v1alpha1.JenkinsJobBuildRun) (jenkinsClient, error) {
	called := j.Called(jf)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(jenkinsClient), nil
}
