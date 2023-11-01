package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	authV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/model"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
)

var log = ctrl.Log.WithName("platform")

// K8SService struct for K8S platform service.
type K8SService struct {
	Scheme           *runtime.Scheme
	client           client.Client
	authClient       authV1Client.RbacV1Client
	networkingClient networkingV1Client.NetworkingV1Interface
	appsV1Client     appsV1Client.AppsV1Interface
}

// Init initializes K8SService.
func (s *K8SService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient client.Client) error {
	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init auth V1 client for K8S: %w", err)
	}

	s.Scheme = scheme
	s.authClient = *authClient
	s.client = k8sClient

	ncl, err := networkingV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init networking V1 client for K8S: %w", err)
	}

	s.networkingClient = ncl

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init apps V1 client for K8S: %w", err)
	}

	s.appsV1Client = appsClient

	return nil
}

// GetExternalEndpoint returns Ingress object and connection protocol from Kubernetes.
func (s *K8SService) GetExternalEndpoint(namespace, name string) (host, routeScheme, path string, err error) {
	ingress, err := s.networkingClient.
		Ingresses(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", "", "", fmt.Errorf("failed to find ingress %v in namespace %v", name, namespace)
		}

		return "", "", "", fmt.Errorf("failed to get ingress: %w", err)
	}

	specHost := ingress.Spec.Rules[0].Host
	routeHTTPSScheme := jenkinsDefaultSpec.RouteHTTPSScheme
	specPath := strings.TrimRight(ingress.Spec.Rules[0].HTTP.Paths[0].Path, platformHelper.UrlCutset)

	return specHost, routeHTTPSScheme, specPath, nil
}

// AddVolumeToInitContainer adds volume to Jenkins init container.
func (s *K8SService) AddVolumeToInitContainer(instance *jenkinsApi.Jenkins, containerName string,
	vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount,
) error {
	if len(vol) == 0 || len(volMount) == 0 {
		return nil
	}

	deployment, err := s.appsV1Client.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	initContainer, err := selectContainer(deployment.Spec.Template.Spec.InitContainers, containerName)
	if err != nil {
		return err
	}

	initContainer.VolumeMounts = updateVolumeMounts(initContainer.VolumeMounts, volMount)
	deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, initContainer)
	volumes := deployment.Spec.Template.Spec.Volumes
	volumes = updateVolumes(volumes, vol)
	deployment.Spec.Template.Spec.Volumes = volumes

	jsonDc, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	_, err = s.appsV1Client.
		Deployments(deployment.Namespace).
		Patch(context.TODO(), deployment.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch deployment: %w", err)
	}

	return nil
}

func (s *K8SService) IsDeploymentReady(instance *jenkinsApi.Jenkins) (bool, error) {
	deployment, err := s.appsV1Client.
		Deployments(instance.Namespace).
		Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get deployments: %w", err)
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (s *K8SService) CreateSecret(instance *jenkinsApi.Jenkins, name string, data map[string][]byte) error {
	_, err := s.getSecret(name, instance.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			secret := &coreV1Api.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: instance.Namespace,
					Labels:    platformHelper.GenerateLabels(instance.Name),
				},
				Data: data,
				Type: "Opaque",
			}

			return s.createSecretInK8S(instance, secret)
		}

		return fmt.Errorf("failed to get Secret %v object: %w", name, err)
	}

	return nil
}

func (s *K8SService) getSecret(name, namespace string) (*coreV1Api.Secret, error) {
	secret := &coreV1Api.Secret{}

	if err := s.client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		secret,
	); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return secret, nil
}

func (s *K8SService) createSecretInK8S(jenkins *jenkinsApi.Jenkins, secret *coreV1Api.Secret) error {
	if err := controllerutil.SetControllerReference(jenkins, secret, s.Scheme); err != nil {
		return fmt.Errorf("failed to set reference for Secret %v object: %w", secret.Name, err)
	}

	if err := s.client.Create(context.TODO(), secret); err != nil {
		return fmt.Errorf("failed to create Secret %v object: %w", secret.Name, err)
	}

	log.Info(fmt.Sprintf("Secret %v has been created", secret.Name))

	return nil
}

// GetSecretData return data field of Secret.
func (s *K8SService) GetSecretData(namespace, name string) (map[string][]byte, error) {
	secret, err := s.getSecret(name, namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))

			return nil, nil
		}

		return nil, err
	}

	return secret.Data, nil
}

func (s *K8SService) CreateConfigMapWithUpdate(instance *jenkinsApi.Jenkins, name string, data map[string]string,
	labels ...map[string]string,
) (isUpdated bool, err error) {
	currentConfigMap, err := s.CreateConfigMap(instance, name, data, labels...)
	if err != nil {
		return false, fmt.Errorf("failed to create configmap: %w", err)
	}

	if reflect.DeepEqual(data, currentConfigMap.Data) {
		return false, nil
	}

	currentConfigMap.Data = data
	if err := s.client.Update(context.TODO(), currentConfigMap); err != nil {
		return false, fmt.Errorf("failed to update config map: %w", err)
	}

	return true, nil
}

func (s *K8SService) CreateConfigMap(instance *jenkinsApi.Jenkins, name string, data map[string]string,
	labels ...map[string]string,
) (*coreV1Api.ConfigMap, error) {
	currentConfigMap, err := s.getConfigMap(name, instance.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			resultLabels := platformHelper.GenerateLabels(instance.Name)

			if len(labels) != 0 {
				resultLabels = labels[0]
			}

			currentConfigMap = &coreV1Api.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: instance.Namespace,
					Labels:    resultLabels,
				},
				Data: data,
			}

			if createMapErr := s.createConfigMapInK8S(instance, currentConfigMap); createMapErr != nil {
				return nil, fmt.Errorf("failed to create config map: %w", err)
			}

			return currentConfigMap, nil
		}

		return nil, fmt.Errorf("failed to get ConfigMap %v object: %w", name, err)
	}

	return currentConfigMap, nil
}

func (s *K8SService) getConfigMap(name, namespace string) (*coreV1Api.ConfigMap, error) {
	configMap := &coreV1Api.ConfigMap{}

	if err := s.client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		configMap,
	); err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	return configMap, nil
}

func (s *K8SService) createConfigMapInK8S(jenkins *jenkinsApi.Jenkins, cm *coreV1Api.ConfigMap) error {
	if err := controllerutil.SetControllerReference(jenkins, cm, s.Scheme); err != nil {
		return fmt.Errorf("failed to set reference for Config Map %v object: %w", cm.Name, err)
	}

	if err := s.client.Create(context.TODO(), cm); err != nil {
		return fmt.Errorf("failed to create Config Map %v object: %w", cm.Name, err)
	}

	log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))

	return nil
}

// CreateConfigMapFromFileOrDir performs creating ConfigMap in K8S.
func (s *K8SService) CreateConfigMapFromFileOrDir(instance *jenkinsApi.Jenkins, configMapName string,
	configMapKey *string, path string, _ metav1.Object, customLabels ...map[string]string,
) error {
	configMapData, err := s.fillConfigMapData(path, configMapKey)
	if err != nil {
		return fmt.Errorf("failed to generate Config Map data for %v: %w", configMapName, err)
	}

	labels := platformHelper.GenerateLabels(instance.Name)

	if len(customLabels) == 1 {
		for key, value := range customLabels[0] {
			labels[key] = value
		}
	}

	_, err = s.CreateConfigMap(instance, configMapName, configMapData, labels)
	if err != nil {
		return fmt.Errorf("failed to create Config Map %v: %w", configMapName, err)
	}

	return nil
}

func (s *K8SService) CreateEDPComponentIfNotExist(jen *jenkinsApi.Jenkins, url, icon string) error {
	c, err := s.getEDPComponent(jen.Name, jen.Namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return s.createEDPComponent(jen, url, icon)
		}

		return fmt.Errorf("failed to get edp component, with parans (%v, %v): %w",
			jen.Name, jen.Namespace, err)
	}

	log.V(1).Info("edp component already exists", "name", c.Name)

	return nil
}

func (s *K8SService) getEDPComponent(name, namespace string) (*edpCompApi.EDPComponent, error) {
	edpComponent := &edpCompApi.EDPComponent{}

	if err := s.client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		edpComponent,
	); err != nil {
		return nil, fmt.Errorf("failed to get EDP component: %w", err)
	}

	return edpComponent, nil
}

func (s *K8SService) createEDPComponent(jen *jenkinsApi.Jenkins, url, icon string) error {
	edpComponent := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jen.Name,
			Namespace: jen.Namespace,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "jenkins",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}

	if err := controllerutil.SetControllerReference(jen, edpComponent, s.Scheme); err != nil {
		return fmt.Errorf("failed to set Controller reference: %w", err)
	}

	if err := s.client.Create(context.TODO(), edpComponent); err != nil {
		return fmt.Errorf("failed to create EDPComponent: %w", err)
	}

	return nil
}

func (s *K8SService) fillConfigMapData(path string, configMapKey *string) (map[string]string, error) {
	var configMapData map[string]string

	pathInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open path %v: %w", path, err)
	}

	if pathInfo.Mode().IsDir() {
		configMapData, err = s.fillConfigMapFromDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to generate config map data from directory %v: %w", path, err)
		}

		return configMapData, nil
	}

	configMapData, err = s.fillConfigMapFromFile(path, configMapKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config map data from file %v: %w", path, err)
	}

	return configMapData, nil
}

func (*K8SService) fillConfigMapFromFile(path string, configMapKey *string) (map[string]string, error) {
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read file %v: %w", path, err)
	}

	key := filepath.Base(path)
	if configMapKey != nil {
		key = *configMapKey
	}

	configMapData := map[string]string{
		key: string(content),
	}

	return configMapData, nil
}

func (*K8SService) fillConfigMapFromDir(path string) (map[string]string, error) {
	configMapData := make(map[string]string)

	directory, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open path %v: %w", path, err)
	}

	for _, file := range directory {
		content, err := os.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to open path %v: %w", path, err)
		}

		configMapData[file.Name()] = string(content)
	}

	return configMapData, nil
}

// GetConfigMapData return data field of ConfigMap.
func (s *K8SService) GetConfigMapData(namespace, name string) (map[string]string, error) {
	cm, err := s.getConfigMap(name, namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to find config map %s in namespace %s", name, namespace)
		}

		return nil, fmt.Errorf("failed to get ConfigMap %s object: %w", name, err)
	}

	return cm.Data, nil
}

func (s *K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	if err := s.client.Get(context.TODO(), nsn, kc); err != nil {
		if k8sErrors.IsNotFound(err) {
			if err = s.client.Create(context.TODO(), kc); err != nil {
				return fmt.Errorf("failed to create Keycloak client %s/%s: %w", kc.Namespace, kc.Name, err)
			}

			log.Info(fmt.Sprintf("Keycloak client %s/%s created", kc.Namespace, kc.Name))

			return nil
		}

		return fmt.Errorf("failed to create Keycloak client %s/%s: %w", kc.Namespace, kc.Name, err)
	}

	return nil
}

func (s *K8SService) GetKeycloakClient(name, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := s.client.Get(context.TODO(), nsn, &out); err != nil {
		return out, fmt.Errorf("failed to get KeycloakClient: %w", err)
	}

	return out, nil
}

func (s *K8SService) CreateJenkinsScript(namespace, name string, forceRecreate bool) (*jenkinsApi.JenkinsScript, error) {
	js, err := s.getJenkinsScript(name, namespace)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			newJenkinsScript := &jenkinsApi.JenkinsScript{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: jenkinsApi.JenkinsScriptSpec{
					SourceCmName: name,
				},
			}

			if err = s.client.Create(context.TODO(), newJenkinsScript); err != nil {
				return nil, fmt.Errorf("failed to create jenkins script: %w", err)
			}

			return newJenkinsScript, nil
		}

		return nil, fmt.Errorf("failed to getJenkinsScript: %w", err)
	}

	if forceRecreate {
		if err := s.client.Delete(context.TODO(), js); err != nil {
			return nil, fmt.Errorf("failed to delete jenkins script: %w", err)
		}

		js = &jenkinsApi.JenkinsScript{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: jenkinsApi.JenkinsScriptSpec{
				SourceCmName: name,
			},
		}

		if err := s.client.Create(context.TODO(), js); err != nil {
			return nil, fmt.Errorf("failed to create jenkins script: %w", err)
		}
	}

	return js, nil
}

func (s *K8SService) getJenkinsScript(name, namespace string) (*jenkinsApi.JenkinsScript, error) {
	js := &jenkinsApi.JenkinsScript{}

	if err := s.client.Get(
		context.TODO(),
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		js,
	); err != nil {
		return nil, fmt.Errorf("failed to get JenkinsScript: %w", err)
	}

	return js, nil
}

func selectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	for i := 0; i < len(containers); i++ {
		if containers[i].Name == name {
			return containers[i], nil
		}
	}

	return coreV1Api.Container{}, fmt.Errorf("failed to find matching container in spec")
}

func updateVolumes(existing, vol []coreV1Api.Volume) []coreV1Api.Volume {
	var (
		out     []coreV1Api.Volume
		covered []string
	)

	for i := 0; i < len(existing); i++ {
		newer, ok := findVolume(vol, existing[i].Name)

		if ok {
			covered = append(covered, existing[i].Name)
			out = append(out, newer)

			continue
		}

		out = append(out, existing[i])
	}

	for i := 0; i < len(vol); i++ {
		if helpers.IsStringInSlice(vol[i].Name, covered) {
			continue
		}

		covered = append(covered, vol[i].Name)
		out = append(out, vol[i])
	}

	return out
}

func updateVolumeMounts(existing, volMount []coreV1Api.VolumeMount) []coreV1Api.VolumeMount {
	var (
		out     []coreV1Api.VolumeMount
		covered []string
	)

	for _, v := range existing {
		newer, ok := findVolumeMount(volMount, v.Name)
		if ok {
			covered = append(covered, v.Name)
			out = append(out, newer)

			continue
		}

		out = append(out, v)
	}

	for _, v := range volMount {
		if helpers.IsStringInSlice(v.Name, covered) {
			continue
		}

		covered = append(covered, v.Name)
		out = append(out, v)
	}

	return out
}

func findVolumeMount(volMount []coreV1Api.VolumeMount, name string) (coreV1Api.VolumeMount, bool) {
	for i := 0; i < len(volMount); i++ {
		if volMount[i].Name == name {
			return volMount[i], true
		}
	}

	return coreV1Api.VolumeMount{}, false
}

func findVolume(vol []coreV1Api.Volume, name string) (coreV1Api.Volume, bool) {
	for i := 0; i < len(vol); i++ {
		if vol[i].Name == name {
			return vol[i], true
		}
	}

	return coreV1Api.Volume{}, false
}

func (*K8SService) CreateStageJSON(stage *cdPipeApi.Stage) (string, error) {
	j := []model.PipelineStage{
		{
			Name:     "deploy-helm",
			StepName: "deploy-helm",
		},
	}

	for _, ps := range stage.Spec.QualityGates {
		i := model.PipelineStage{
			Name:     ps.QualityGateType,
			StepName: ps.StepName,
		}

		j = append(j, i)
	}

	j = append(j, model.PipelineStage{Name: "promote-images", StepName: "promote-images"})

	o, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("failed to marshal: %w", err)
	}

	return string(o), err
}
