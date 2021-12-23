package chain

import (
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

func TestInitDefChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := InitDefChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}

func TestInitDefChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, "test")
	assert.NoError(t, err)
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitDefChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	err = os.Unsetenv(helper.PlatformType)
	assert.NoError(t, err)
}

func TestInitTriggerJobProvisionChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}

func TestInitTriggerJobProvisionChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, "test")
	assert.NoError(t, err)
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	err = os.Unsetenv(helper.PlatformType)
	assert.NoError(t, err)
}
