package chain

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	wrongPlatform = "test"
)

func TestInitDefChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitDefChain(s, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment variable PLATFORM_TYPE not found")
}

func TestInitDefChain_NewPlatformServiceErr(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, wrongPlatform))

	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitDefChain(s, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown platform type")

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func TestInitDefChain(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, platform.K8SPlatformType))

	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitDefChain(s, client)
	assert.NoError(t, err)

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func TestInitTriggerJobProvisionChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment variable PLATFORM_TYPE not found")
}

func TestInitTriggerJobProvisionChain_NewPlatformServiceErr(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, wrongPlatform))

	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown platform type")

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func TestInitTriggerJobProvisionChain(t *testing.T) {
	require.NoError(t, os.Setenv(helper.PlatformType, platform.K8SPlatformType))

	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	_, err := InitTriggerJobProvisionChain(s, client)
	assert.NoError(t, err)

	require.NoError(t, os.Unsetenv(helper.PlatformType))
}

func Test_nextServeOrNil(t *testing.T) {
	jj := &jenkinsApi.JenkinsJob{}
	jj.Name = "name"

	assert.NoError(t, nextServeOrNil(nil, jj))
}
