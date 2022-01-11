package chain

import (
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jfmock "github.com/epam/edp-jenkins-operator/v2/mock/jenkins_folder"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
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
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateCDPipelineFolderChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, wrongPlatform)
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	jenkinsFolderHandler, err := CreateCDPipelineFolderChain(scheme, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateCDPipelineFolderChain(t *testing.T) {
	err := os.Setenv(helper.PlatformType, platform.K8SPlatformType)
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	jenkinsFolderHandler, err := CreateCDPipelineFolderChain(scheme, client)
	assert.NoError(t, err)
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, jenkinsFolderHandler)
}

func TestCreateTriggerBuildProvisionChain_GetPlatformTypeEnvErr(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
	assert.Nil(t, jenkinsFolderHandler)
}

func TestCreateTriggerBuildProvisionChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, wrongPlatform)
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	assert.Nil(t, jenkinsFolderHandler)
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTriggerBuildProvisionChain(t *testing.T) {
	err := os.Setenv(helper.PlatformType, platform.K8SPlatformType)
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	jenkinsFolderHandler, err := CreateTriggerBuildProvisionChain(scheme, client)
	assert.NoError(t, err)
	assert.NotNil(t, jenkinsFolderHandler)
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_nextServeOrNil(t *testing.T) {
	jenkinsFolder := &v1alpha1.JenkinsFolder{}
	jenkinsFolder.Name = "name"
	err := nextServeOrNil(nil, jenkinsFolder)
	assert.NoError(t, err)
}

func Test_nextServeOrNilErr(t *testing.T) {
	jenkinsFolder := &v1alpha1.JenkinsFolder{}
	jenkinsFolderHandler := jfmock.JenkinsFolderHandler{}
	errTest := errors.New("test")
	jenkinsFolderHandler.On("ServeRequest", jenkinsFolder).Return(errTest)
	jenkinsFolder.Name = "name"
	err := nextServeOrNil(&jenkinsFolderHandler, jenkinsFolder)
	assert.Equal(t, err, errTest)
}
