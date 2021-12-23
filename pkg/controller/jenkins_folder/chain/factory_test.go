package chain

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

func TestCreateCDPipelineFolderChain(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := CreateCDPipelineFolderChain(s, &client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}

func TestCreateTriggerBuildProvisionChain(t *testing.T) {
	s := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	_, err := CreateTriggerBuildProvisionChain(s, &client)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
}
