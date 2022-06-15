package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kmock "github.com/epam/edp-jenkins-operator/v2/mock/kubernetes"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
)

const (
	name      = "name"
	namespace = "ns"
)

func TestK8SService_CreateConfigMap(t *testing.T) {
	cm := v1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cm)
	utilruntime.Must(jenkinsApi.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&cm).Build()

	svc := K8SService{
		client: client,
		Scheme: scheme,
	}

	ji := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	_, err := svc.CreateConfigMap(ji, "test", map[string]string{"bar": "baz"}, map[string]string{"lol": "lol"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateConfigMapWithUpdate(ji, "test", map[string]string{"bar": "baz"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateConfigMapWithUpdate(ji, "test", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestK8SService_CreateJenkinsScript(t *testing.T) {
	cm := v1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cm)
	utilruntime.Must(jenkinsApi.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&cm).Build()

	svc := K8SService{
		client: client,
		Scheme: scheme,
	}

	if _, err := svc.CreateJenkinsScript("ns", "name", true); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.CreateJenkinsScript("ns", "name", true); err != nil {
		t.Fatal(err)
	}
}

func TestK8SService_Init(t *testing.T) {
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

	service := K8SService{}
	err = service.Init(restConfig, scheme, client)
	assert.NoError(t, err)
}

func TestK8SService_GetExternalEndpointErr(t *testing.T) {
	ingress := &kmock.Ingress{}
	netClient := &kmock.NetworkingClient{}
	errTest := errors.New("test")

	netClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{networkingClient: netClient}
	_, _, _, err := service.GetExternalEndpoint(namespace, name)
	assert.Equal(t, errTest, err)
	ingress.AssertExpectations(t)
	netClient.AssertExpectations(t)
}

func TestK8SService_GetExternalEndpoint(t *testing.T) {
	host := "host"
	ingress := &kmock.Ingress{}
	netClient := &kmock.NetworkingClient{}
	ingressPath := networkingV1.HTTPIngressPath{
		Path: name,
	}
	ingressRule := networkingV1.IngressRule{
		Host: host,
		IngressRuleValue: networkingV1.IngressRuleValue{
			HTTP: &networkingV1.HTTPIngressRuleValue{
				Paths: []networkingV1.HTTPIngressPath{ingressPath},
			},
		},
	}
	ingressInstance := &networkingV1.Ingress{Spec: networkingV1.IngressSpec{Rules: []networkingV1.IngressRule{ingressRule}}}

	netClient.On("Ingresses", namespace).Return(ingress)
	ingress.On("Get", context.TODO(), name, metav1.GetOptions{}).Return(ingressInstance, nil)

	service := K8SService{networkingClient: netClient}
	endpoint, scheme, path, err := service.GetExternalEndpoint(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, host, endpoint)
	assert.Equal(t, jenkinsDefaultSpec.RouteHTTPSScheme, scheme)
	assert.Equal(t, name, path)
	ingress.AssertExpectations(t)
	netClient.AssertExpectations(t)
}

func TestK8SService_IsDeploymentReadyErr(t *testing.T) {
	instance := jenkinsApi.Jenkins{}

	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}
	errTest := errors.New("test")
	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	service := K8SService{appsV1Client: appClient}

	_, err := service.IsDeploymentReady(instance)
	assert.Equal(t, errTest, err)
	appClient.AssertExpectations(t)
	deployment.AssertExpectations(t)
}

func TestK8SService_IsDeploymentReadyFalse(t *testing.T) {
	instance := jenkinsApi.Jenkins{}
	deploymentInstance := &appsv1.Deployment{}

	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	service := K8SService{appsV1Client: appClient}

	ok, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.False(t, ok)
	appClient.AssertExpectations(t)
	deployment.AssertExpectations(t)
}

func TestK8SService_IsDeploymentReadyTrue(t *testing.T) {
	instance := jenkinsApi.Jenkins{}
	deploymentInstance := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		}}

	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	service := K8SService{appsV1Client: appClient}

	ok, err := service.IsDeploymentReady(instance)
	assert.NoError(t, err)
	assert.True(t, ok)
	appClient.AssertExpectations(t)
	deployment.AssertExpectations(t)
}

func TestK8SService_GetSecretData_EmptyClient(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}
	data, err := service.GetSecretData(namespace, name)
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestK8SService_GetSecretData(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{name: []byte(namespace)},
	}
	client := fake.NewClientBuilder().WithObjects(secret).Build()
	service := K8SService{client: client}
	data, err := service.GetSecretData(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, []byte(namespace), data[name])
}

func TestK8SService_CreateSecretErr(t *testing.T) {
	data := map[string][]byte{name: []byte(namespace)}
	scheme := runtime.NewScheme()
	instance := &jenkinsApi.Jenkins{}
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateSecret(instance, namespace, data)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Couldn't set reference for Secret ns object"))
}

func TestK8SService_CreateSecret(t *testing.T) {
	data := map[string][]byte{name: []byte(namespace)}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{}, &v1.Secret{})
	instance := &jenkinsApi.Jenkins{}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateSecret(instance, namespace, data)
	assert.NoError(t, err)
}

func TestK8SService_AddVolumeToInitContainer_EmptyArgs(t *testing.T) {
	instance := &jenkinsApi.Jenkins{}
	var vol []v1.Volume
	var volMount []v1.VolumeMount
	service := K8SService{}
	err := service.AddVolumeToInitContainer(instance, name, vol, volMount)
	assert.NoError(t, err)
}

func TestK8SService_AddVolumeToInitContainer_DeploymentGetErr(t *testing.T) {
	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}
	instance := &jenkinsApi.Jenkins{}

	errTest := errors.New("test")

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(nil, errTest)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	service := K8SService{
		appsV1Client: appClient,
	}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.NoError(t, err)
}

func TestK8SService_AddVolumeToInitContainer_NotFoundErr(t *testing.T) {
	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}
	instance := &jenkinsApi.Jenkins{}
	deploymentInstance := &appsv1.Deployment{}

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(deploymentInstance, nil)

	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	service := K8SService{
		appsV1Client: appClient,
	}
	err := service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "No matching container in spec found!"))
}

func TestK8SService_AddVolumeToInitContainer_SelectContainerErr(t *testing.T) {
	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}
	instance := &jenkinsApi.Jenkins{}
	deploymentInstance := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: name}}}},
		},
	}
	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}
	errTest := errors.New("test")

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)

	deploymentInstanceModified := deploymentInstance
	container := deploymentInstanceModified.Spec.Template.Spec.InitContainers[0]
	container.VolumeMounts = volMounts
	deploymentInstanceModified.Spec.Template.Spec.InitContainers = append(deploymentInstanceModified.Spec.Template.Spec.InitContainers, container)
	deploymentInstanceModified.Spec.Template.Spec.Volumes = vols
	raw, err := json.Marshal(deploymentInstanceModified)
	if err != nil {
		t.Fatal(err)
	}

	deployment.On("Patch", context.TODO(), deploymentInstance.Name, types.StrategicMergePatchType, raw, metav1.PatchOptions{}).Return(nil, errTest)

	service := K8SService{
		appsV1Client: appClient,
	}
	err = service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.Error(t, err)
}

func TestK8SService_AddVolumeToInitContainer(t *testing.T) {
	appClient := &kmock.AppsV1Client{}
	deployment := &kmock.Deployment{}
	instance := &jenkinsApi.Jenkins{}
	deploymentInstance := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: name}}}},
		},
	}
	vols := []v1.Volume{{Name: name}}
	volMounts := []v1.VolumeMount{{Name: name}}

	appClient.On("Deployments", "").Return(deployment)
	deployment.On("Get", context.TODO(), instance.Name, metav1.GetOptions{}).Return(&deploymentInstance, nil)

	deploymentInstanceModified := deploymentInstance
	container := deploymentInstanceModified.Spec.Template.Spec.InitContainers[0]
	container.VolumeMounts = volMounts
	deploymentInstanceModified.Spec.Template.Spec.InitContainers = append(deploymentInstanceModified.Spec.Template.Spec.InitContainers, container)
	deploymentInstanceModified.Spec.Template.Spec.Volumes = vols
	raw, err := json.Marshal(deploymentInstanceModified)
	if err != nil {
		t.Fatal(err)
	}

	deployment.On("Patch", context.TODO(), deploymentInstance.Name, types.StrategicMergePatchType, raw, metav1.PatchOptions{}).Return(nil, nil)

	service := K8SService{
		appsV1Client: appClient,
	}
	err = service.AddVolumeToInitContainer(instance, name, vols, volMounts)
	assert.NoError(t, err)
}

func TestK8SService_GetKeycloakClientErr(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}
	_, err := service.GetKeycloakClient(name, namespace)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered for the type"))
}

func TestK8SService_GetKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance).Build()
	service := K8SService{client: client}
	out, err := service.GetKeycloakClient(name, namespace)
	assert.NoError(t, err)
	assert.Equal(t, *instance, out)
}

func TestK8SService_CreateKeycloakClientErr(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{}
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}
	err := service.CreateKeycloakClient(instance)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered for the type"))
}

func TestK8SService_CreateKeycloakClient(t *testing.T) {
	instance := &keycloakV1Api.KeycloakClient{
		TypeMeta: metav1.TypeMeta{Kind: "KeycloakClient", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &keycloakV1Api.KeycloakClient{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	service := K8SService{client: client}
	err := service.CreateKeycloakClient(instance)
	assert.NoError(t, err)
}

func TestK8SService_CreateEDPComponentIfNotExist_GetErr(t *testing.T) {
	instance := jenkinsApi.Jenkins{}
	client := fake.NewClientBuilder().Build()

	service := K8SService{client: client}
	err := service.CreateEDPComponentIfNotExist(instance, "", "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered"))
}

func TestK8SService_CreateEDPComponentIfNotExist_AlreadyExist(t *testing.T) {
	instance := jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}}
	EDPcomponent := edpCompApi.EDPComponent{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}}
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &edpCompApi.EDPComponent{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&EDPcomponent).Build()

	service := K8SService{client: client}
	err := service.CreateEDPComponentIfNotExist(instance, "", "")
	assert.NoError(t, err)
}

func TestK8SService_CreateEDPComponentIfNotExist(t *testing.T) {
	instance := jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &edpCompApi.EDPComponent{}, &jenkinsApi.Jenkins{})

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()

	service := K8SService{client: client, Scheme: scheme}
	err := service.CreateEDPComponentIfNotExist(instance, "test.com", "icon.png")
	assert.NoError(t, err)
}

func TestK8SService_GetConfigMapData_NotFound(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	service := K8SService{client: client}
	_, err := service.GetConfigMapData(namespace, name)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Config map name in namespace"))
}

func TestK8SService_GetConfigMapData(t *testing.T) {
	data := map[string]string{name: namespace}
	configMap := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
		Data: data}
	client := fake.NewClientBuilder().WithObjects(configMap).Build()
	service := K8SService{client: client}
	configData, err := service.GetConfigMapData(namespace, name)
	assert.NoError(t, err)
	assert.Equal(t, data, configData)
}

func TestK8SService_GetConfigMapData_NotRegistered(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	service := K8SService{client: client}
	_, err := service.GetConfigMapData(namespace, name)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no kind is registered"))
}

func TestK8SService_CreateStageJSON_Empty(t *testing.T) {
	stage := cdPipeApi.Stage{}
	service := K8SService{}
	expected := `[{"name":"deploy-helm","step_name":"deploy-helm"},{"name":"promote-images","step_name":"promote-images"}]`
	jsonData, err := service.CreateStageJSON(stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}

func TestK8SService_CreateStageJSON(t *testing.T) {
	stage := cdPipeApi.Stage{
		Spec: cdPipeApi.StageSpec{
			QualityGates: []cdPipeApi.QualityGate{{QualityGateType: "type", StepName: name}},
		}}
	service := K8SService{}
	expected := `[{"name":"deploy-helm","step_name":"deploy-helm"},{"name":"type","step_name":"name"},{"name":"promote-images","step_name":"promote-images"}]`
	jsonData, err := service.CreateStageJSON(stage)
	assert.NoError(t, err)
	assert.Equal(t, expected, jsonData)
}
