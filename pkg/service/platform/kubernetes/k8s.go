package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	edpCompClient "github.com/epam/edp-component-operator/pkg/client"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsScriptV1Client "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkinsscript/client"
	"github.com/epam/edp-jenkins-operator/v2/pkg/model"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"io/ioutil"
	coreV1 "k8s.io/api/core/v1"
	coreV1Api "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsV1Client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	authV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

var log = ctrl.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme                *runtime.Scheme
	coreClient            coreV1Client.CoreV1Client
	jenkinsScriptClient   jenkinsScriptV1Client.EdpV1Client
	k8sUnstructuredClient client.Client
	authClient            authV1Client.RbacV1Client
	extensionsV1Client    extensionsV1Client.ExtensionsV1beta1Client
	appsV1Client          appsV1Client.AppsV1Client
	edpCompClient         edpCompClient.EDPComponentV1Client
}

// Init initializes K8SService
func (service *K8SService) Init(config *rest.Config, Scheme *runtime.Scheme, k8sClient *client.Client) error {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init core client for K8S")
	}

	jenkinsScriptClient, err := jenkinsScriptV1Client.NewForConfig(config)
	if err != nil {
		return err
	}

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init auth V1 client for K8S")
	}

	service.jenkinsScriptClient = *jenkinsScriptClient
	service.coreClient = *coreClient
	service.k8sUnstructuredClient = *k8sClient
	service.Scheme = Scheme
	service.authClient = *authClient

	extensionsClient, err := extensionsV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init extensions V1 client for K8S")
	}
	service.extensionsV1Client = *extensionsClient

	appsClient, err := appsV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init apps V1 client for K8S")
	}
	service.appsV1Client = *appsClient

	compCl, err := edpCompClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to init edp component client")
	}
	service.edpCompClient = *compCl
	return nil
}

// GetExternalEndpoint returns Ingress object and connection protocol from Kubernetes
func (service K8SService) GetExternalEndpoint(namespace string, name string) (string, string, string, error) {
	ingress, err := service.extensionsV1Client.Ingresses(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", "", "", errors.New(fmt.Sprintf("Ingress %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return "", "", "", err
	}

	return ingress.Spec.Rules[0].Host, jenkinsDefaultSpec.RouteHTTPSScheme,
		strings.TrimRight(ingress.Spec.Rules[0].HTTP.Paths[0].Path, platformHelper.UrlCutset), nil
}

// AddVolumeToInitContainer adds volume to Jenkins init container
func (service K8SService) AddVolumeToInitContainer(instance v1alpha1.Jenkins, containerName string,
	vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error {

	if len(vol) == 0 || len(volMount) == 0 {
		return nil
	}

	deployment, err := service.appsV1Client.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
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
		return err
	}

	_, err = service.appsV1Client.Deployments(deployment.Namespace).Patch(context.TODO(), deployment.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (service K8SService) IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error) {
	deployment, err := service.appsV1Client.Deployments(instance.Namespace).Get(context.TODO(), instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

//CreateSecret creates secret object in K8s cluster
func (service K8SService) CreateSecret(instance v1alpha1.Jenkins, name string, data map[string][]byte) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)
	labels := platformHelper.GenerateLabels(instance.Name)

	secretObject := &coreV1Api.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: data,
		Type: "Opaque",
	}

	if err := controllerutil.SetControllerReference(&instance, secretObject, service.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Secret %v object", secretObject.Name)
	}

	secret, err := service.coreClient.Secrets(secretObject.Namespace).Get(context.TODO(), secretObject.Name, metav1.GetOptions{})

	if err != nil && k8sErrors.IsNotFound(err) {
		secret, err = service.coreClient.Secrets(secretObject.Namespace).Create(context.TODO(), secretObject, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Secret %v object", secret.Name)
		}
		reqLogger.Info(fmt.Sprintf("Secret %v has been created", secret.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Couldn't get Secret %v object", secretObject.Name)
	}

	return nil
}

// GetSecret return data field of Secret
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	secret, err := service.coreClient.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (service K8SService) CreateConfigMap(instance v1alpha1.Jenkins, configMapName string, configMapData map[string]string, labels ...map[string]string) error {
	resultLabels := platformHelper.GenerateLabels(instance.Name)
	if len(labels) != 0 {
		resultLabels = labels[0]
	}
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    resultLabels,
		},
		Data: configMapData,
	}

	if err := controllerutil.SetControllerReference(&instance, configMapObject, service.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Config Map %v object", configMapObject.Name)
	}

	cm, err := service.coreClient.ConfigMaps(instance.Namespace).Get(context.TODO(), configMapObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			cm, err = service.coreClient.ConfigMaps(configMapObject.Namespace).Create(context.TODO(), configMapObject, metav1.CreateOptions{})
			if err != nil {
				return errors.Wrapf(err, "Couldn't create Config Map %v object", configMapObject.Name)
			}
			log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, configMapObject.Name))
		}
		return errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMapObject.Name)
	}
	return nil
}

// CreateConfigMapFromFile performs creating ConfigMap in K8S
func (service K8SService) CreateConfigMapFromFileOrDir(instance v1alpha1.Jenkins, configMapName string,
	configMapKey *string, path string, ownerReference metav1.Object, customLabels ...map[string]string) error {
	configMapData, err := service.fillConfigMapData(path, configMapKey)
	if err != nil {
		return errors.Wrapf(err, "Couldn't generate Config Map data for %v", configMapName)
	}

	labels := platformHelper.GenerateLabels(instance.Name)
	if len(customLabels) == 1 {
		for key, value := range customLabels[0] {
			labels[key] = value
		}
	}

	err = service.CreateConfigMap(instance, configMapName, configMapData, labels)
	if err != nil {
		return errors.Wrapf(err, "Failed to create Config Map %v", configMapName)
	}

	return nil
}

func (service K8SService) CreateEDPComponentIfNotExist(jen v1alpha1.Jenkins, url string, icon string) error {
	comp, err := service.edpCompClient.
		EDPComponents(jen.Namespace).
		Get(jen.Name, metav1.GetOptions{})
	if err == nil {
		log.V(1).Info("edp component already exists", "name", comp.Name)
		return nil
	}
	if k8sErrors.IsNotFound(err) {
		return service.createEDPComponent(jen, url, icon)
	}
	return errors.Wrapf(err, "failed to get edp component: %v", jen.Name)
}

func (service K8SService) createEDPComponent(jen v1alpha1.Jenkins, url string, icon string) error {
	obj := &edpCompApi.EDPComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: jen.Name,
		},
		Spec: edpCompApi.EDPComponentSpec{
			Type:    "jenkins",
			Url:     url,
			Icon:    icon,
			Visible: true,
		},
	}
	if err := controllerutil.SetControllerReference(&jen, obj, service.Scheme); err != nil {
		return err
	}
	_, err := service.edpCompClient.
		EDPComponents(jen.Namespace).
		Create(obj)
	return err
}

func (service K8SService) fillConfigMapData(path string, configMapKey *string) (map[string]string, error) {
	configMapData := make(map[string]string)
	pathInfo, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
	}
	if pathInfo.Mode().IsDir() {
		configMapData, err = service.fillConfigMapFromDir(path)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't generate config map data from directory %v", path))
		}
	} else {
		configMapData, err = service.fillConfigMapFromFile(path, configMapKey)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't generate config map data from file %v", path))
		}
	}
	return configMapData, nil
}

func (service K8SService) fillConfigMapFromFile(path string, configMapKey *string) (map[string]string, error) {
	configMapData := make(map[string]string)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't read file %v.", path))
	}
	key := filepath.Base(path)
	if configMapKey != nil {
		key = *configMapKey
	}
	configMapData = map[string]string{
		key: string(content),
	}
	return configMapData, nil
}

func (service K8SService) fillConfigMapFromDir(path string) (map[string]string, error) {
	configMapData := make(map[string]string)
	directory, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
	}
	for _, file := range directory {
		content, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("Couldn't open path %v.", path))
		}
		configMapData[file.Name()] = string(content)
	}
	return configMapData, nil
}

// GetConfigMapData return data field of ConfigMap
func (service K8SService) GetConfigMapData(namespace string, name string) (map[string]string, error) {
	configMap, err := service.coreClient.ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "Config map %v in namespace %v not found", name, namespace)
		}
		return nil, errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMap.Name)
	}
	return configMap.Data, nil
}

func (service K8SService) CreateKeycloakClient(kc *keycloakV1Api.KeycloakClient) error {
	nsn := types.NamespacedName{
		Namespace: kc.Namespace,
		Name:      kc.Name,
	}

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, kc)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			err := service.k8sUnstructuredClient.Create(context.TODO(), kc)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
			}
			log.Info(fmt.Sprintf("Keycloak client %s/%s created", kc.Namespace, kc.Name))
		}
		return errors.Wrapf(err, "Failed to create Keycloak client %s/%s", kc.Namespace, kc.Name)
	}

	return nil
}

func (service K8SService) GetKeycloakClient(name string, namespace string) (keycloakV1Api.KeycloakClient, error) {
	out := keycloakV1Api.KeycloakClient{}
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	err := service.k8sUnstructuredClient.Get(context.TODO(), nsn, &out)
	if err != nil {
		return out, err
	}

	return out, nil
}

func (service K8SService) CreateJenkinsScript(namespace string, configMap string) (*v1alpha1.JenkinsScript, error) {
	jso := &v1alpha1.JenkinsScript{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMap,
			Namespace: namespace,
		},
		Spec: v1alpha1.JenkinsScriptSpec{
			SourceCmName: configMap,
		},
	}

	js, err := service.jenkinsScriptClient.Get(context.TODO(), configMap, namespace, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			js, err := service.jenkinsScriptClient.Create(context.TODO(), jso, namespace)
			if err != nil {
				return nil, err
			}
			// Success
			return js, nil
		}
		// Error occurred
		return nil, err
	}
	// Nothing to do
	return js, nil

}

func selectContainer(containers []coreV1Api.Container, name string) (coreV1Api.Container, error) {
	for _, c := range containers {
		if c.Name == name {
			return c, nil
		}
	}

	return coreV1Api.Container{}, errors.New("No matching container in spec found!")
}

func updateVolumes(existing []coreV1Api.Volume, vol []coreV1Api.Volume) []coreV1Api.Volume {
	var out []coreV1Api.Volume
	var covered []string

	for _, v := range existing {
		newer, ok := findVolume(vol, v.Name)
		if ok {
			covered = append(covered, v.Name)
			out = append(out, newer)
			continue
		}
		out = append(out, v)
	}
	for _, v := range vol {
		if helpers.IsStringInSlice(v.Name, covered) {
			continue
		}
		covered = append(covered, v.Name)
		out = append(out, v)
	}
	return out
}

func updateVolumeMounts(existing []coreV1Api.VolumeMount, volMount []coreV1Api.VolumeMount) []coreV1Api.VolumeMount {
	var out []coreV1Api.VolumeMount
	var covered []string

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
	for _, v := range volMount {
		if v.Name == name {
			return v, true
		}
	}
	return coreV1Api.VolumeMount{}, false
}

func findVolume(vol []coreV1Api.Volume, name string) (coreV1Api.Volume, bool) {
	for _, v := range vol {
		if v.Name == name {
			return v, true
		}
	}
	return coreV1Api.Volume{}, false
}

func (s K8SService) CreateProject(name string) error {
	log.V(2).Info("start sending request to create namespace...", "name", name)
	_, err := s.coreClient.Namespaces().Create(context.TODO(),
		&coreV1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
		metav1.CreateOptions{},
	)
	return err
}

// CreateRoleBinding creates RoleBinding
func (s K8SService) CreateRoleBinding(edpName string, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error {
	log.V(2).Info("start creating role binding", "edp name", edpName, "namespace", namespace, "role name", roleRef)
	_, err := s.authClient.RoleBindings(namespace).Create(context.TODO(),
		&rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s", edpName, roleRef.Name),
			},
			RoleRef:  roleRef,
			Subjects: subjects,
		},
		metav1.CreateOptions{},
	)
	return err
}

func (s K8SService) CreateStageJSON(stage cdPipeApi.Stage) (string, error) {
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
	j = append(j, model.PipelineStage{Name: "promote-images-ecr", StepName: "promote-images-ecr"})

	o, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(o), err
}

func (s K8SService) DeleteProject(name string) error {
	var grace int64 = 0
	return s.coreClient.Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
	})
}

// GetRoleBinding get RoleBinding
func (s K8SService) GetRoleBinding(roleBindingName, namespace string) (*rbacV1.RoleBinding, error) {
	rb, err := s.authClient.RoleBindings(namespace).Get(context.TODO(), roleBindingName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return rb, nil
}
