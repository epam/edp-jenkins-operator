package chain

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
)

const (
	wrongPlatform = "test"
	validPlatform = "kubernetes"
)

func TestInitDefChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := InitDefChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}

func TestInitDefChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, wrongPlatform)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitDefChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitDefChain(t *testing.T) {
	err := os.Setenv(helper.PlatformType, validPlatform)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitDefChain(s, client)
	assert.NoError(t, err)
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitTriggerJobProvisionChain_PlatformTypeErr(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}

func TestInitTriggerJobProvisionChain_NewPlatformServiceErr(t *testing.T) {
	err := os.Setenv(helper.PlatformType, wrongPlatform)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitTriggerJobProvisionChain(s, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitTriggerJobProvisionChain(t *testing.T) {
	err := os.Setenv(helper.PlatformType, validPlatform)
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err = InitTriggerJobProvisionChain(s, client)
	assert.NoError(t, err)
	err = os.Unsetenv(helper.PlatformType)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_nextServeOrNil(t *testing.T) {
	jj := &v1alpha1.JenkinsJob{}
	jj.Name = "name"
	err := nextServeOrNil(nil, jj)
	assert.NoError(t, err)
}
