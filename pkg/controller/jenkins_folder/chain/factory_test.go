package chain

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jfmock "github.com/epam/edp-jenkins-operator/v2/mock/jenkins_folder"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	wrongPlatform = "test"
)

func TestCreateCDPipelineFolderChain_GetPlatformTypeEnvErr(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateCDPipelineFolderChain(scheme, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment variable PLATFORM_TYPE not found")
	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateCDPipelineFolderChain_NewPlatformServiceErr(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, wrongPlatform))

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateCDPipelineFolderChain(scheme, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "received unknown platform type")

	require.NoError(t, os.Unsetenv(helper.PlatformType))

	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateCDPipelineFolderChain(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, platform.K8SPlatformType))

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateCDPipelineFolderChain(scheme, client)
	require.NoError(t, err)

	require.NoError(t, os.Unsetenv(helper.PlatformType))

	assert.NotNil(t, jenkinsFolderHandler)
}

func TestCreateTriggerBuildProvisionChain_GetPlatformTypeEnvErr(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment variable PLATFORM_TYPE not found")
	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateTriggerBuildProvisionChain_NewPlatformServiceErr(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, wrongPlatform))

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "received unknown platform type")
	assert.Nil(t, jenkinsFolderHandler)

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func TestCreateTriggerBuildProvisionChain(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, platform.K8SPlatformType))

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.NoError(t, err)
	assert.NotNil(t, jenkinsFolderHandler)

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func Test_nextServeOrNil(t *testing.T) {
	jenkinsFolder := &jenkinsApi.JenkinsFolder{}
	jenkinsFolder.Name = "name"

	assert.NoError(t, nextServeOrNil(nil, jenkinsFolder))
}

func Test_nextServeOrNilErr(t *testing.T) {
	jenkinsFolder := &jenkinsApi.JenkinsFolder{}
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	errTest := errors.New("test")
	jenkinsFolderHandler.On("ServeRequest", jenkinsFolder).Return(errTest)
	jenkinsFolder.Name = "name"

	err := nextServeOrNil(&jenkinsFolderHandler, jenkinsFolder)
	assert.Error(t, err)

	assert.Contains(t, err.Error(), "failed to serve next request: test")
}
