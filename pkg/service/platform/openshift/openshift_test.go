package openshift

import (
	"context"
	"fmt"
	"os"
	"testing"

	appv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"

	omock "github.com/epam/edp-jenkins-operator/v2/mock/openshift"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

const (
	name      = "name"
	namespace = "ns"
)

func TestOpenshiftService_Init(t *testing.T) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restConfig, err := config.ClientConfig()
	require.NoError(t, err)

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()
	service := OpenshiftService{}

	assert.NoError(t, service.Init(restConfig, scheme, client))
}

func TestOpenshiftService_CreateStageJSON_Empty(t *testing.T) {
	stage := cdPipeApi.Stage{}
	service := OpenshiftService{}
	expected := `[{"name":"deploy","step_name":"deploy"},{"name":"promote-images","step_name":"promote-images"}]`

	jsonData, err := service.CreateStageJSON(&stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}

func TestOpenshiftService_CreateStageJSON(t *testing.T) {
	stage := cdPipeApi.Stage{
		Spec: cdPipeApi.StageSpec{
			QualityGates: []cdPipeApi.QualityGate{{QualityGateType: "type", StepName: name}},
		},
	}
	service := OpenshiftService{}
	expected := `[{"name":"deploy","step_name":"deploy"},{"name":"type","step_name":"name"},{"name":"promote-images","step_name":"promote-images"}]`

	jsonData, err := service.CreateStageJSON(&stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}

func TestOpenshiftService_GetExternalEndpoint_GetErr(t *testing.T) {
	routeClient := omock.RouteV1Client{}
	route := omock.Route{}

	routeClient.On("Routes", namespace).Return(&route)
	route.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{routeClient: &routeClient}

	_, _, _, err := service.GetExternalEndpoint(namespace, name) //nolint:dogsled // the values are irrelevant to the test.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Routes: test")

	route.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetExternalEndpoint(t *testing.T) {
	routeClient := omock.RouteV1Client{}
	route := omock.Route{}
	routeInstance := routev1.Route{Spec: routev1.RouteSpec{TLS: &routev1.TLSConfig{Termination: name}}}

	routeClient.On("Routes", namespace).Return(&route)
	route.On("Get", context.TODO(), name, metav1.GetOptions{}).
		Return(&routeInstance, nil)

	service := OpenshiftService{routeClient: &routeClient}

	endpoint, scheme, s2, err := service.GetExternalEndpoint(namespace, name)
	assert.NoError(t, err)
	assert.Empty(t, endpoint)
	assert.Equal(t, "https", scheme)
	assert.Empty(t, s2)

	route.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_AddVolumeToInitContainer_EmptyArgs(t *testing.T) {
	instance := jenkinsApi.Jenkins{}

	var (
		vol      []v1.Volume
		volMount []v1.VolumeMount
	)

	service := OpenshiftService{}

	assert.NoError(t, service.AddVolumeToInitContainer(&instance, name, vol, volMount))
}

type TestOpenShiftAlternativeSuite struct {
	suite.Suite
}

func (s *TestOpenShiftAlternativeSuite) BeforeTest(suiteName, testName string) {
	err := os.Setenv(deploymentTypeEnvName, deploymentConfigsDeploymentType)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *TestOpenShiftAlternativeSuite) AfterTest(suiteName, testName string) {
	err := os.Unsetenv(deploymentTypeEnvName)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyErr() {
	t := s.T()

	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{appClient: appClient}

	_, err := service.IsDeploymentReady(&instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get DeploymentConfigs: test")

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyFalse() {
	t := s.T()

	deploymentConfInstance := appv1.DeploymentConfig{}
	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(&instance)
	assert.NoError(t, err)
	assert.False(t, ok)

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyTrue() {
	t := s.T()

	deploymentConfInstance := appv1.DeploymentConfig{
		Status: appv1.DeploymentConfigStatus{
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		},
	}

	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(&instance)
	assert.NoError(t, err)
	assert.True(t, ok)

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_EmptyArgs2() {
	t := s.T()

	instance := jenkinsApi.Jenkins{}

	var (
		vol      []v1.Volume
		volMount []v1.VolumeMount
	)

	service := OpenshiftService{}

	assert.NoError(t, service.AddVolumeToInitContainer(&instance, name, vol, volMount))
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_CantGet() {
	t := s.T()

	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(nil, fmt.Errorf("test"))

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}
	service := OpenshiftService{appClient: appClient}

	assert.NoError(t, service.AddVolumeToInitContainer(&instance, name, vols, volMounts))

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_NotFoundErr() {
	t := s.T()

	deploymentConfInstance := appv1.DeploymentConfig{
		Spec: appv1.DeploymentConfigSpec{
			Template: &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{Name: ""},
					},
				},
			},
		},
	}
	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	service := OpenshiftService{appClient: appClient}

	err := service.AddVolumeToInitContainer(&instance, name, vols, volMounts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find matching container in spec")

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_PatchErr() {
	t := s.T()

	deploymentConfInstance := appv1.DeploymentConfig{
		Spec: appv1.DeploymentConfigSpec{
			Template: &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{Name: name},
					},
				},
			},
		},
	}
	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	deploymentConf.On(
		"Patch",
		context.TODO(),
		deploymentConfInstance.Name,
		types.StrategicMergePatchType,
		metav1.PatchOptions{},
	).Return(nil, fmt.Errorf("test"))

	service := OpenshiftService{appClient: appClient}

	assert.Error(t, service.AddVolumeToInitContainer(&instance, name, vols, volMounts))

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer() {
	t := s.T()

	deploymentConfInstance := appv1.DeploymentConfig{
		Spec: appv1.DeploymentConfigSpec{
			Template: &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{Name: name},
					},
				},
			},
		},
	}
	instance := jenkinsApi.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).
		Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	deploymentConf.On(
		"Patch",
		context.TODO(),
		deploymentConfInstance.Name,
		types.StrategicMergePatchType,
		metav1.PatchOptions{},
	).Return(nil, nil)

	service := OpenshiftService{appClient: appClient}

	assert.NoError(t, service.AddVolumeToInitContainer(&instance, name, vols, volMounts))

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func TestOpenshiftTestSuite(t *testing.T) {
	suite.Run(t, new(TestOpenShiftAlternativeSuite))
}
