package openshift

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	appv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	omock "github.com/epam/edp-jenkins-operator/v2/mock/openshift"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
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
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().Build()

	service := OpenshiftService{}
	err = service.Init(restConfig, scheme, client)
	assert.NoError(t, err)
}

func TestOpenshiftService_CreateStageJSON_Empty(t *testing.T) {
	stage := cdPipeApi.Stage{}
	service := OpenshiftService{}
	expected := `[{"name":"deploy","step_name":"deploy"},{"name":"promote-images","step_name":"promote-images"}]`
	jsonData, err := service.CreateStageJSON(stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}

func TestOpenshiftService_CreateStageJSON(t *testing.T) {
	stage := cdPipeApi.Stage{
		Spec: cdPipeApi.StageSpec{
			QualityGates: []cdPipeApi.QualityGate{{QualityGateType: "type", StepName: name}},
		}}
	service := OpenshiftService{}
	expected := `[{"name":"deploy","step_name":"deploy"},{"name":"type","step_name":"name"},{"name":"promote-images","step_name":"promote-images"}]`
	jsonData, err := service.CreateStageJSON(stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}

func TestOpenshiftService_GetExternalEndpoint_GetErr(t *testing.T) {
	routeClient := omock.RouteV1Client{}
	route := omock.Route{}
	errTest := errors.New("test")

	routeClient.On("Routes", namespace).Return(&route)
	route.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := OpenshiftService{routeClient: &routeClient}
	_, _, _, err := service.GetExternalEndpoint(namespace, name)
	assert.Error(t, err)
	assert.Equal(t, errTest, err)
	route.AssertExpectations(t)
	routeClient.AssertExpectations(t)
}

func TestOpenshiftService_GetExternalEndpoint(t *testing.T) {
	routeClient := omock.RouteV1Client{}
	route := omock.Route{}
	routeInstance := routev1.Route{Spec: routev1.RouteSpec{TLS: &routev1.TLSConfig{Termination: name}}}

	routeClient.On("Routes", namespace).Return(&route)
	route.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(&routeInstance, nil)

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
	instance := v1alpha1.Jenkins{}
	var vol []v1.Volume
	var volMount []v1.VolumeMount
	service := OpenshiftService{}
	err := service.AddVolumeToInitContainer(instance, name, vol, volMount)
	assert.NoError(t, err)
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
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}
	errTest := errors.New("test")
	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	service := OpenshiftService{appClient: appClient}

	_, err := service.IsDeploymentReady(instance)

	assert.Equal(t, errTest, err)
	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)

}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_IsDeploymentReadyFalse() {
	t := s.T()
	deploymentConfInstance := appv1.DeploymentConfig{}
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(instance)
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
		}}

	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	service := OpenshiftService{appClient: appClient}

	ok, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.True(t, ok)
	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_EmptyArgs2() {
	t := s.T()
	instance := v1alpha1.Jenkins{}
	var vol []v1.Volume
	var volMount []v1.VolumeMount
	service := OpenshiftService{}
	err := service.AddVolumeToInitContainer(instance, name, vol, volMount)
	assert.NoError(t, err)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_CantGet() {
	t := s.T()
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}
	errTest := errors.New("test")

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	service := OpenshiftService{appClient: appClient}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.NoError(t, err)

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)
}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_NotFoundErr() {
	t := s.T()
	deploymentConfInstance := appv1.DeploymentConfig{Spec: appv1.DeploymentConfigSpec{
		Template: &v1.PodTemplateSpec{Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: ""}}}},
	}}
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	service := OpenshiftService{appClient: appClient}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "No matching container in spec found!"))

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)

}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer_PatchErr() {
	t := s.T()
	errTest := errors.New("test")
	deploymentConfInstance := appv1.DeploymentConfig{Spec: appv1.DeploymentConfigSpec{
		Template: &v1.PodTemplateSpec{Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: name}}}},
	}}
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	deploymentConf.On("Patch", context.TODO(), deploymentConfInstance.Name, types.StrategicMergePatchType, metav1.PatchOptions{}).Return(nil, errTest)

	service := OpenshiftService{appClient: appClient}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.Error(t, err)

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)

}

func (s *TestOpenShiftAlternativeSuite) TestOpenshiftService_AddVolumeToInitContainer() {
	t := s.T()
	deploymentConfInstance := appv1.DeploymentConfig{Spec: appv1.DeploymentConfigSpec{
		Template: &v1.PodTemplateSpec{Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: name}}}},
	}}
	instance := v1alpha1.Jenkins{}
	appClient := &omock.AppsV1Client{}
	deploymentConf := &omock.DeploymentConfig{}

	appClient.On("DeploymentConfigs", "").Return(deploymentConf)
	deploymentConf.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentConfInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	deploymentConf.On("Patch", context.TODO(), deploymentConfInstance.Name, types.StrategicMergePatchType, metav1.PatchOptions{}).Return(nil, nil)

	service := OpenshiftService{appClient: appClient}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.NoError(t, err)

	appClient.AssertExpectations(t)
	deploymentConf.AssertExpectations(t)

}

func TestOpenshiftTestSuite(t *testing.T) {
	suite.Run(t, new(TestOpenShiftAlternativeSuite))
}
