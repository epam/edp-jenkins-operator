package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	edpCompApi "github.com/epmd-edp/edp-component-operator/pkg/apis/v1/v1alpha1"
	edpCompClient "github.com/epmd-edp/edp-component-operator/pkg/client"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	helperController "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	jenkinsScriptV1Client "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsscript/client"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/model"
	jenkinsDefaultSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/helper"
	platformHelper "github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/pkg/errors"
	"io/ioutil"
	appsApi "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	coreV1Api "k8s.io/api/core/v1"
	extensionsApi "k8s.io/api/extensions/v1beta1"
	authV1Api "k8s.io/api/rbac/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsV1Client "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	authV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

var log = logf.Log.WithName("platform")

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

	serviceAccount, err := service.coreClient.ServiceAccounts(serviceAccountObject.Namespace).Get(serviceAccountObject.Name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		serviceAccount, err = service.coreClient.ServiceAccounts(serviceAccountObject.Namespace).Create(serviceAccountObject)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Service Account %v object", serviceAccountObject.Name)
		}
		reqLogger.Info(fmt.Sprintf("Service Account %v has been created", serviceAccount.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Couldn't get Service Account %v object", serviceAccountObject.Name)
	}

	return nil
}

// GetExternalEndpoint returns Ingress object and connection protocol from Kubernetes
func (service K8SService) GetExternalEndpoint(namespace string, name string) (string, string, string, error) {
	ingress, err := service.extensionsV1Client.Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", "", "", errors.New(fmt.Sprintf("Ingress %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return "", "", "", err
	}

	return ingress.Spec.Rules[0].Host, jenkinsDefaultSpec.RouteHTTPSScheme,
		strings.TrimRight(ingress.Spec.Rules[0].HTTP.Paths[0].Path, "/"), nil
}

// AddVolumeToInitContainer adds volume to Jenkins init container
func (service K8SService) AddVolumeToInitContainer(instance v1alpha1.Jenkins, containerName string,
	vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error {

	if len(vol) == 0 || len(volMount) == 0 {
		return nil
	}

	deployment, err := service.appsV1Client.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
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

	_, err = service.appsV1Client.Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
		return err
	}
	return nil
}

// CreateDeployment performs creating Deployment in K8S
func (service K8SService) CreateDeployment(instance v1alpha1.Jenkins) error {
	reqLog := log.WithValues("jenkins ", instance)
	h, s, p, err := service.GetExternalEndpoint(instance.Namespace, instance.Name)
	if err != nil {
		return err
	}

	t := true
	l := platformHelper.GenerateLabels(instance.Name)
	var uid int64 = 999
	var gid int64 = 998
	var fid int64 = 0
	url := fmt.Sprintf("%v://%v%v", s, h, p)

	jenkinsOptsEnv := "--requestHeaderSize=32768"
	rpPath := "/login"
	if len(instance.Spec.BasePath) != 0 {
		jenkinsOptsEnv = fmt.Sprintf("%v --prefix=/%v", jenkinsOptsEnv, instance.Spec.BasePath)
		rpPath = fmt.Sprintf("%v/login", instance.Spec.BasePath)
	}

	jo := &appsApi.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    l,
		},
		Spec: appsApi.DeploymentSpec{
			Replicas: &jenkinsDefaultSpec.Replicas,
			Strategy: appsApi.DeploymentStrategy{
				Type: "Recreate",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: l,
			},
			Template: coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l,
				},
				Spec: coreV1Api.PodSpec{
					ImagePullSecrets: instance.Spec.ImagePullSecrets,
					SecurityContext: &coreV1Api.PodSecurityContext{
						RunAsNonRoot: &t,
						FSGroup:      &fid,
					},
					RestartPolicy:                 coreV1Api.RestartPolicyAlways,
					DeprecatedServiceAccount:      instance.Name,
					DNSPolicy:                     coreV1Api.DNSClusterFirst,
					TerminationGracePeriodSeconds: &jenkinsDefaultSpec.TerminationGracePeriod,
					SchedulerName:                 coreV1Api.DefaultSchedulerName,
					InitContainers: []coreV1Api.Container{
						{
							SecurityContext: &coreV1Api.SecurityContext{
								RunAsUser:  &uid,
								RunAsGroup: &gid,
							},
							Image:                    instance.Spec.InitImage,
							ImagePullPolicy:          coreV1Api.PullIfNotPresent,
							Name:                     "grant-permissions",
							Command:                  jenkinsDefaultSpec.Command,
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: coreV1Api.TerminationMessageReadFile,
							VolumeMounts: []coreV1Api.VolumeMount{
								{
									MountPath:        "/var/lib/jenkins",
									Name:             fmt.Sprintf("%v-jenkins-data", instance.Name),
									ReadOnly:         false,
									SubPath:          "",
									MountPropagation: nil,
								},
							},
						},
					},
					Containers: []coreV1Api.Container{
						{
							Name:            instance.Name,
							Image:           instance.Spec.Image + ":" + instance.Spec.Version,
							ImagePullPolicy: coreV1Api.PullAlways,
							SecurityContext: &coreV1Api.SecurityContext{
								RunAsUser:  &uid,
								RunAsGroup: &gid,
							},
							Env: []coreV1Api.EnvVar{
								{
									Name: "CI_NAMESPACE",
									ValueFrom: &coreV1Api.EnvVarSource{
										FieldRef: &coreV1Api.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "OPENSHIFT_ENABLE_OAUTH",
									Value: "false",
								},
								{
									Name:  "OPENSHIFT_ENABLE_REDIRECT_PROMPT",
									Value: "true",
								},
								{
									Name:  "KUBERNETES_MASTER",
									Value: "https://kubernetes.default:443",
								},
								{
									Name:  "KUBERNETES_TRUST_CERTIFICATES",
									Value: "true",
								},
								{
									Name:  "JNLP_SERVICE_NAME",
									Value: fmt.Sprintf("%v-jnlp", instance.Name),
								},
								{
									Name: "JENKINS_PASSWORD",
									ValueFrom: &coreV1Api.EnvVarSource{
										SecretKeyRef: &coreV1Api.SecretKeySelector{
											LocalObjectReference: coreV1Api.LocalObjectReference{
												Name: fmt.Sprintf("%v-%v", instance.Name, jenkinsDefaultSpec.JenkinsPasswordSecretName),
											},
											Key: "password",
										},
									},
								},
								{
									Name:  "JENKINS_UI_URL",
									Value: url,
								},
								{
									Name:  "JENKINS_OPTS",
									Value: jenkinsOptsEnv,
								},
								{
									Name:  "PLATFORM_TYPE",
									Value: helperController.GetPlatformTypeEnv(),
								},
							},
							Ports: []coreV1Api.ContainerPort{
								{
									ContainerPort: jenkinsDefaultSpec.JenkinsDefaultUiPort,
									Protocol:      coreV1Api.ProtocolTCP,
								},
							},

							ReadinessProbe: &coreV1Api.Probe{
								TimeoutSeconds:      10,
								InitialDelaySeconds: 60,
								SuccessThreshold:    1,
								PeriodSeconds:       10,
								FailureThreshold:    3,
								Handler: coreV1Api.Handler{
									HTTPGet: &coreV1Api.HTTPGetAction{
										Path:   rpPath,
										Port:   intstr.FromInt(jenkinsDefaultSpec.JenkinsDefaultUiPort),
										Scheme: coreV1Api.URISchemeHTTP,
									},
								},
							},

							VolumeMounts: []coreV1Api.VolumeMount{
								{
									MountPath:        "/var/lib/jenkins",
									Name:             fmt.Sprintf("%v-jenkins-data", instance.Name),
									ReadOnly:         false,
									SubPath:          "",
									MountPropagation: nil,
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: coreV1Api.TerminationMessageReadFile,
							Resources: coreV1Api.ResourceRequirements{
								Requests: map[coreV1Api.ResourceName]resource.Quantity{
									coreV1Api.ResourceMemory: resource.MustParse(jenkinsDefaultSpec.JenkinsDefaultMemoryRequest),
								},
							},
						},
					},
					ServiceAccountName: instance.Name,
					Volumes: []coreV1Api.Volume{
						{
							Name: fmt.Sprintf("%v-jenkins-data", instance.Name),
							VolumeSource: coreV1Api.VolumeSource{
								PersistentVolumeClaim: &coreV1Api.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%v-data", instance.Name),
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, jo, service.Scheme); err != nil {
		return err
	}

	d, err := service.appsV1Client.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if !k8sErrors.IsNotFound(err) {
		return err
	}

	d, err = service.appsV1Client.Deployments(instance.Namespace).Create(jo)
	if err != nil {
		return err
	}

	reqLog.Info("Deployment has been created",
		"Namespace", d.Namespace, "Name", d.Name, "JenkinsName", d.Name)

	return err
}

func (service K8SService) IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error) {
	deployment, err := service.appsV1Client.Deployments(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deployment.Status.UpdatedReplicas == 1 && deployment.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
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

		volume, err := service.coreClient.PersistentVolumeClaims(volumeObject.Namespace).Get(volumeObject.Name, metav1.GetOptions{})

		if err != nil && k8sErrors.IsNotFound(err) {
			volume, err = service.coreClient.PersistentVolumeClaims(volumeObject.Namespace).Create(volumeObject)
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

	svc, err := service.coreClient.Services(instance.Namespace).Get(serviceObject.Name, metav1.GetOptions{})

	if err != nil && k8sErrors.IsNotFound(err) {
		svc, err = service.coreClient.Services(serviceObject.Namespace).Create(serviceObject)
		if err != nil {
			return errors.Wrapf(err, "Couldn't create Service %v object", svc.Name)
		}
		reqLogger.Info(fmt.Sprintf("Service %v has been created", svc.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Couldn't get Service %v object", serviceObject.Name)
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

	secret, err := service.coreClient.Secrets(secretObject.Namespace).Get(secretObject.Name, metav1.GetOptions{})

	if err != nil && k8sErrors.IsNotFound(err) {
		secret, err = service.coreClient.Secrets(secretObject.Namespace).Create(secretObject)
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
	secret, err := service.coreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
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

	cm, err := service.coreClient.ConfigMaps(instance.Namespace).Get(configMapObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			cm, err = service.coreClient.ConfigMaps(configMapObject.Namespace).Create(configMapObject)
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

// CreateRole creates new role in k8s
func (service K8SService) CreateRole(instance v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error {
	roleObject := &authV1Api.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: instance.Namespace,
		},
		Rules: rules,
	}

	if err := controllerutil.SetControllerReference(&instance, roleObject, service.Scheme); err != nil {
		return errors.Wrap(err, "Failed to set Owner Reference")
	}

	consoleRole, err := service.authClient.Roles(roleObject.ObjectMeta.Namespace).Get(roleObject.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			consoleRole, err = service.authClient.Roles(roleObject.Namespace).Create(roleObject)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Role %v", roleObject.Name)
			}
			log.Info(fmt.Sprintf("Role %s is created", consoleRole.Name))
			return nil
		}
		return errors.Wrapf(err, "Getting Role %v failed", roleObject.Name)
	}

	return nil
}

// CreateClusterRole creates new cluster role
func (service K8SService) CreateClusterRole(instance v1alpha1.Jenkins, clusterRoleName string, rules []authV1Api.PolicyRule) error {
	clusterRoleObject := &authV1Api.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: rules,
	}

	if err := controllerutil.SetControllerReference(&instance, clusterRoleObject, service.Scheme); err != nil {
		return err
	}

	clusterRole, err := service.authClient.ClusterRoles().Get(clusterRoleObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			clusterRole, err = service.authClient.ClusterRoles().Create(clusterRoleObject)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Cluster Role %v", clusterRoleObject.Name)
			}
			log.Info(fmt.Sprintf("Cluster Role %s is created", clusterRole.Name))
			return nil
		}
		return errors.Wrapf(err, "Getting Cluster Role %v failed", clusterRoleObject.Name)
	}

	return nil
}

// CreateClusterRolePolicyRules
func (service K8SService) CreateClusterRolePolicyRules() []authV1Api.PolicyRule {
	return []authV1Api.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"podsecuritypolicies"},
			Verbs:     []string{"get", "list", "update"},
		},
		{
			APIGroups: []string{"*"},
			Resources: []string{"namespaces"},
			Verbs:     []string{"create", "get"},
		},
	}
}

// CreateUserClusterRoleBinding binds user to clusterRole
func (service K8SService) CreateUserClusterRoleBinding(instance v1alpha1.Jenkins, clusterRoleBindingName string, clusterRoleName string) error {
	bindingObject, err := helper.GetNewClusterRoleBindingObject(instance, clusterRoleBindingName, clusterRoleName)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(&instance, bindingObject, service.Scheme); err != nil {
		return err
	}

	binding, err := service.authClient.ClusterRoleBindings().Get(bindingObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			binding, err = service.authClient.ClusterRoleBindings().Create(bindingObject)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Cluster Role Binding %v", bindingObject.Name)
			}
			log.Info(fmt.Sprintf("Cluster Role Binding %s has been created", binding.Name))
			return nil
		}
		return errors.Wrapf(err, "Getting Cluster Role Binding %v failed", bindingObject.Name)
	}

	return nil
}

func (service K8SService) CreateUserRoleBinding(instance v1alpha1.Jenkins, roleBindingName string, roleName string, roleKind string) error {
	bindingObject, err := helper.GetNewRoleBindingObject(instance, roleBindingName, roleName, roleKind)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(&instance, bindingObject, service.Scheme); err != nil {
		return err
	}

	binding, err := service.authClient.RoleBindings(bindingObject.Namespace).Get(bindingObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			binding, err = service.authClient.RoleBindings(bindingObject.Namespace).Create(bindingObject)
			if err != nil {
				return errors.Wrapf(err, "Failed to create Role Binding %v", bindingObject.Name)
			}
			log.Info(fmt.Sprintf("Role Binding %s has been created", binding.Name))
			return nil
		}
		return errors.Wrapf(err, "Getting Role Binding %v failed", bindingObject.Name)
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
			Type: "jenkins",
			Url:  url,
			Icon: icon,
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
	configMap, err := service.coreClient.ConfigMaps(namespace).Get(name, metav1.GetOptions{})

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

	js, err := service.jenkinsScriptClient.Get(configMap, namespace, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			js, err := service.jenkinsScriptClient.Create(jso, namespace)
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

// CreateExternalEndpoint creates k8s ingress
func (service K8SService) CreateExternalEndpoint(instance v1alpha1.Jenkins) error {
	labels := helper.GenerateLabels(instance.Name)

	hostname := fmt.Sprintf("%v-%v.%v", instance.Name, instance.Namespace, instance.Spec.EdpSpec.DnsWildcard)
	path := "/"
	if len(instance.Spec.BasePath) != 0 {
		hostname = instance.Spec.EdpSpec.DnsWildcard
		path = fmt.Sprintf("/%v", instance.Spec.BasePath)
	}

	ingressObject := &extensionsApi.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: extensionsApi.IngressSpec{
			Rules: []extensionsApi.IngressRule{
				{
					Host: hostname,
					IngressRuleValue: extensionsApi.IngressRuleValue{
						HTTP: &extensionsApi.HTTPIngressRuleValue{
							Paths: []extensionsApi.HTTPIngressPath{
								{
									Path: path,
									Backend: extensionsApi.IngressBackend{
										ServiceName: instance.Name,
										ServicePort: intstr.IntOrString{IntVal: jenkinsDefaultSpec.JenkinsDefaultUiPort},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, ingressObject, service.Scheme); err != nil {
		return err
	}

	ingress, err := service.extensionsV1Client.Ingresses(ingressObject.Namespace).Get(ingressObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			ingress, err = service.extensionsV1Client.Ingresses(ingressObject.Namespace).Create(ingressObject)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Ingress %s/%s has been created", ingress.Namespace, ingress.Name))
		}
		return err
	}

	return nil
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
	_, err := s.coreClient.Namespaces().Create(
		&coreV1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
			},
		},
	)
	return err
}

// CreateRoleBinding creates RoleBinding
func (s K8SService) CreateRoleBinding(edpName string, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error {
	log.V(2).Info("start creating role binding", "edp name", edpName, "namespace", namespace, "role name", roleRef)
	_, err := s.authClient.RoleBindings(namespace).Create(
		&rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s", edpName, roleRef.Name),
			},
			RoleRef:  roleRef,
			Subjects: subjects,
		},
	)
	return err
}

func (s K8SService) CreateStageJSON(cr edpv1alpha1.Stage) (string, error) {
	j := []model.PipelineStage{
		{
			Name:     "deploy-helm",
			StepName: "deploy-helm",
		},
	}

	for _, ps := range cr.Spec.QualityGates {
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
	return s.coreClient.Namespaces().Delete(name, metav1.NewDeleteOptions(0))
}

// GetRoleBinding get RoleBinding
func (s K8SService) GetRoleBinding(roleBindingName, namespace string) (*rbacV1.RoleBinding, error) {
	rb, err := s.authClient.RoleBindings(namespace).Get(roleBindingName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return rb, nil
}
