package platform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewPlatformService_NonValidPlatform(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	scheme := runtime.NewScheme()
	platformType := "test"
	service, err := NewPlatformService(platformType, scheme, client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Unknown platform type"))
	assert.Nil(t, service)
}

func TestNewPlatformService_K8SPlatform(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	scheme := runtime.NewScheme()
	platformType := K8SPlatformType
	service, err := NewPlatformService(platformType, scheme, client)
	assert.NoError(t, err)
	assert.NotNil(t, service)
}

func TestNewPlatformService_OpenshiftPlatform(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	scheme := runtime.NewScheme()
	platformType := OpenShiftPlatformType
	service, err := NewPlatformService(platformType, scheme, client)
	assert.NoError(t, err)
	assert.NotNil(t, service)
}
