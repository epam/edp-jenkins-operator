package kubernetes

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	jenkinsDefaultSpec "jenkins-operator/pkg/service/jenkins/spec"
	platformHelper "jenkins-operator/pkg/service/platform/helper"
	coreV1Api "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// K8SService struct for K8S platform service
type K8SService struct {
	Scheme     *runtime.Scheme
	CoreClient coreV1Client.CoreV1Client
}

// Init initializes K8SService
func (service *K8SService) Init(config *rest.Config, Scheme *runtime.Scheme) error {
	CoreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init core client for K8S")
	}
	service.CoreClient = *CoreClient
	service.Scheme = Scheme
	return nil
}

// CreateServiceAccount performs creating ServiceAccount in K8S
func (service K8SService) CreateServiceAccount(instance v1alpha1.Jenkins) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)
	labels := platformHelper.GenerateLabels(instance.Name)

	serviceAccountObject := &coreV1Api.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceAccountObject, service.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Service Account %v object", serviceAccountObject.Name)
	}

	serviceAccount, err := service.CoreClient.ServiceAccounts(serviceAccountObject.Namespace).Get(serviceAccountObject.Name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		serviceAccount, err = service.CoreClient.ServiceAccounts(serviceAccountObject.Namespace).Create(serviceAccountObject)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Service Account %v object", serviceAccountObject.Name)
		}
		reqLogger.Info(fmt.Sprintf("Service Account %v has been created", serviceAccount.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Couldn't get Service Account %v object", serviceAccountObject.Name)
	}

	return nil
}

// CreateVolume performs creating PersistentVolumeClaim in K8S
func (service K8SService) CreatePersistentVolumeClaim(instance v1alpha1.Jenkins) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)
	labels := platformHelper.GenerateLabels(instance.Name)

	for _, volume := range instance.Spec.Volumes {
		volumeObject := &coreV1Api.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name + "-" + volume.Name,
				Namespace: instance.Namespace,
				Labels:    labels,
			},
			Spec: coreV1Api.PersistentVolumeClaimSpec{
				AccessModes: []coreV1Api.PersistentVolumeAccessMode{
					coreV1Api.ReadWriteOnce,
				},
				StorageClassName: &volume.StorageClass,
				Resources: coreV1Api.ResourceRequirements{
					Requests: map[coreV1Api.ResourceName]resource.Quantity{
						coreV1Api.ResourceStorage: resource.MustParse(volume.Capacity),
					},
				},
			},
		}

		if err := controllerutil.SetControllerReference(&instance, volumeObject, service.Scheme); err != nil {
			return errors.Wrapf(err, "Couldn't set reference for Persistent Volume Claim %v object", volumeObject.Name)
		}

		volume, err := service.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Get(volumeObject.Name, metav1.GetOptions{})

		if err != nil && k8serr.IsNotFound(err) {
			volume, err = service.CoreClient.PersistentVolumeClaims(volumeObject.Namespace).Create(volumeObject)
			if err != nil {
				return errors.Wrapf(err, "Couldn't create Persistent Volume Claim %v object", volume.Name)
			}
			reqLogger.Info(fmt.Sprintf("Persistant Volume Claim %v has been created", volume.Name))
		} else if err != nil {
			return errors.Wrapf(err, "Couldn't get Persistent Volume Claim %v object", volumeObject.Name)
		}
	}
	return nil
}

// CreateService performs creating Service in K8S
func (service K8SService) CreateService(instance v1alpha1.Jenkins) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)
	labels := platformHelper.GenerateLabels(instance.Name)

	serviceObject := &coreV1Api.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: coreV1Api.ServiceSpec{
			Selector: labels,
			Ports: []coreV1Api.ServicePort{
				{
					TargetPort: intstr.IntOrString{IntVal: jenkinsDefaultSpec.JenkinsDefaultUiPort},
					Port:       jenkinsDefaultSpec.JenkinsDefaultUiPort,
					Name:       "http",
					Protocol:   coreV1Api.ProtocolTCP,
				},
				{
					TargetPort: intstr.IntOrString{IntVal: jenkinsDefaultSpec.JenkinsDefaultJnlpPort},
					Port:       jenkinsDefaultSpec.JenkinsDefaultJnlpPort,
					Name:       "jnlp",
					Protocol:   coreV1Api.ProtocolTCP,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, serviceObject, service.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Service %v object", serviceObject.Name)
	}

	svc, err := service.CoreClient.Services(instance.Namespace).Get(serviceObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		svc, err = service.CoreClient.Services(serviceObject.Namespace).Create(serviceObject)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Service %v object", svc.Name)
		}
		reqLogger.Info(fmt.Sprintf("Service %v has been created", svc.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Couldn't get Service %v object", serviceObject.Name)
	} else if !reflect.DeepEqual(svc.Spec.Ports, serviceObject.Spec.Ports) {
		svc.Spec.Ports = serviceObject.Spec.Ports
		_, err := service.CoreClient.Services(instance.Namespace).Update(svc)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Couldn't update Service %v object", svc.Name))
		}
		reqLogger.Info(fmt.Sprintf("Service %v has been updated", svc.Name))
	}

	return nil
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

	secret, err := service.CoreClient.Secrets(secretObject.Namespace).Get(secretObject.Name, metav1.GetOptions{})

	if err != nil && k8serr.IsNotFound(err) {
		secret, err = service.CoreClient.Secrets(secretObject.Namespace).Create(secretObject)
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
	secret, err := service.CoreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		log.Info(fmt.Sprintf("Secret %v in namespace %v not found", name, namespace))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (service K8SService) CreateConfigMapFromData(instance v1alpha1.Jenkins, configMapName string,
	configMapData map[string]string, labels map[string]string, ownerReference metav1.Object) error {
	configMapObject := &coreV1Api.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Data: configMapData,
	}

	if err := controllerutil.SetControllerReference(ownerReference, configMapObject, service.Scheme); err != nil {
		return errors.Wrapf(err, "Couldn't set reference for Config Map %v object", configMapObject.Name)
	}

	cm, err := service.CoreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})
	if err != nil && k8serr.IsNotFound(err) {
		cm, err = service.CoreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Config Map %v object", cm.Name)
		}
		log.Info(fmt.Sprintf("ConfigMap %s/%s has been created", cm.Namespace, cm.Name))
	} else if err != nil {
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

	err = service.CreateConfigMapFromData(instance, configMapName, configMapData, labels, ownerReference)
	if err != nil {
		return errors.Wrapf(err, "Failed to create Config Map %v", configMapName)
	}

	return nil
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
	configMap, err := service.CoreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

	if err != nil {
		if k8serr.IsNotFound(err) {
			log.Error(err, fmt.Sprintf("Config map %v in namespace %v not found", name, namespace))
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Couldn't get ConfigMap %v object", configMap.Name)
	}
	return configMap.Data, nil
}
