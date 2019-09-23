package openshift

import (
	"encoding/json"
	"fmt"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsDefaultSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/kubernetes"
	appsV1Api "github.com/openshift/api/apps/v1"
	authV1Api "github.com/openshift/api/authorization/v1"
	routeV1Api "github.com/openshift/api/route/v1"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authV1Client "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	authClient  authV1Client.AuthorizationV1Client
	appClient   appsV1client.AppsV1Client
	routeClient routeV1Client.RouteV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme,k8sClient *client.Client) error {
	err := service.K8SService.Init(config, scheme,k8sClient)
	if err != nil {
		return errors.Wrap(err, "Failed to init K8S platform service")
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init apps V1 client for Openshift")
	}
	service.appClient = *appClient

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init route V1 client for Openshift")
	}
	service.routeClient = *routeClient

	authClient, err := authV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init auth V1 client for Openshift")
	}
	service.authClient = *authClient

	return nil
}

// GetRoute returns Route object and connection protocol from Openshift
func (service OpenshiftService) GetRoute(namespace string, name string) (*routeV1Api.Route, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil, "", errors.New(fmt.Sprintf("Route %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return nil, "", err
	}

	var routeScheme = "http"
	if route.Spec.TLS.Termination != "" {
		routeScheme = "https"
	}
	return route, routeScheme, nil
}

// CreateDeployConf - creates deployment configs for Jenkins instance
func (service OpenshiftService) CreateDeployConf(instance v1alpha1.Jenkins) error {

	activeDeadlineSecond := int64(21600)
	terminationGracePeriod := int64(30)
	jenkinsRoute, routeScheme, err := service.GetRoute(instance.Namespace, instance.Name)
	if err != nil {
		return err
	}

	jenkinsUiUrl := fmt.Sprintf("%v://%v", routeScheme, jenkinsRoute.Spec.Host)

	// Can't assign pointer to constant, that is why — create an intermediate var.
	timeout := jenkinsDefaultSpec.JenkinsRecreateTimeout
	command := []string{"sh", "-c", fmt.Sprintf(
		"JENKINS_HOME=\"/var/lib/jenkins\"; mkdir -p $JENKINS_HOME/.ssh; if [ -d /tmp/ssh ];" +
			"then chmod 777 -R $JENKINS_HOME/.ssh; cat /tmp/ssh/id_rsa >> $JENKINS_HOME/.ssh/id_rsa;" +
			"chmod 400 $JENKINS_HOME/.ssh/id_rsa; fi")}

	labels := helper.GenerateLabels(instance.Name)
	jenkinsDcObject := &appsV1Api.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsV1Api.DeploymentConfigSpec{
			Replicas: 1,
			Triggers: []appsV1Api.DeploymentTriggerPolicy{
				{
					Type: appsV1Api.DeploymentTriggerOnConfigChange,
				},
			},
			Strategy: appsV1Api.DeploymentStrategy{
				Type: appsV1Api.DeploymentStrategyTypeRecreate,
				RecreateParams: &appsV1Api.RecreateDeploymentStrategyParams{
					TimeoutSeconds: &timeout,
				},
				ActiveDeadlineSeconds: &activeDeadlineSecond,
			},
			Selector: labels,
			Template: &coreV1Api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: coreV1Api.PodSpec{
					SecurityContext:               &coreV1Api.PodSecurityContext{},
					RestartPolicy:                 coreV1Api.RestartPolicyAlways,
					DeprecatedServiceAccount:      instance.Name,
					DNSPolicy:                     coreV1Api.DNSClusterFirst,
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					SchedulerName:                 coreV1Api.DefaultSchedulerName,
					InitContainers: []coreV1Api.Container{
						{
							Image:                    "busybox",
							ImagePullPolicy:          coreV1Api.PullIfNotPresent,
							Name:                     "grant-permissions",
							Command:                  command,
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
							Env: []coreV1Api.EnvVar{
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
									Value: jenkinsUiUrl,
								},
								{
									Name:  "JENKINS_OPTS",
									Value: "--requestHeaderSize=32768",
								},
							},
							SecurityContext: nil,
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
										Path:   "/login",
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

	if err := controllerutil.SetControllerReference(&instance, jenkinsDcObject, service.Scheme); err != nil {
		return err
	}

	jenkinsDc, err := service.appClient.DeploymentConfigs(jenkinsDcObject.Namespace).Get(jenkinsDcObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		log.V(1).Info(fmt.Sprintf("Creating a new DeploymentConfig %s/%s for Jenkins %s", jenkinsDcObject.Namespace, jenkinsDcObject.Name, instance.Name))

		jenkinsDc, err = service.appClient.DeploymentConfigs(jenkinsDcObject.Namespace).Create(jenkinsDcObject)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("DeploymentConfig %s/%s has been created", jenkinsDc.Namespace, jenkinsDc.Name))
	} else if err != nil {
		return err
	}

	return nil
}

func (service OpenshiftService) CreateExternalEndpoint(instance v1alpha1.Jenkins) error {

	labels := helper.GenerateLabels(instance.Name)

	routeObject := &routeV1Api.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: routeV1Api.RouteSpec{
			TLS: &routeV1Api.TLSConfig{
				Termination:                   routeV1Api.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routeV1Api.InsecureEdgeTerminationPolicyRedirect,
			},
			To: routeV1Api.RouteTargetReference{
				Name: instance.Name,
				Kind: "Service",
			},
			Port: &routeV1Api.RoutePort{
				TargetPort: intstr.IntOrString{IntVal: jenkinsDefaultSpec.JenkinsDefaultUiPort},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&instance, routeObject, service.Scheme); err != nil {
		return err
	}

	route, err := service.routeClient.Routes(routeObject.Namespace).Get(routeObject.Name, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		route, err = service.routeClient.Routes(routeObject.Namespace).Create(routeObject)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Route %s/%s has been created", route.Namespace, route.Name))
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(routeObject.Spec, route.Spec) {

		route.Spec = routeObject.Spec
		_, err = service.routeClient.Routes(routeObject.Namespace).Update(route)
		if err != nil {
			return errors.Wrapf(err, "Failed to update DeploymentConfig %v !", routeObject.Name)
		}
	}

	return nil
}

//noinspection GoUnresolvedReference
func (service OpenshiftService) CreateRole(instance v1alpha1.Jenkins, roleName string, rules []authV1Api.PolicyRule) error {
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

	consoleRole, err := service.authClient.Roles(roleObject.Namespace).Get(roleObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
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

func (service OpenshiftService) CreateUserRoleBinding(instance v1alpha1.Jenkins, roleBindingName string, roleName string, kind string) error {
	bindingObject, err := helper.GetNewRoleBindingObject(instance, roleBindingName, roleName, kind)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(&instance, bindingObject, service.Scheme); err != nil {
		return err
	}

	binding, err := service.authClient.RoleBindings(bindingObject.Namespace).Get(bindingObject.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
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

// GetDeploymentConfig returns DeploymentConfig object from Openshift
func (service OpenshiftService) GetDeploymentConfig(instance v1alpha1.Jenkins) (*appsV1Api.DeploymentConfig, error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return deploymentConfig, nil
}

func (service OpenshiftService) AddVolumeToInitContainer(instance v1alpha1.Jenkins, dc *appsV1Api.DeploymentConfig,
	containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error {

	if len(vol) == 0 || len(volMount) == 0 {
		return nil
	}

	initContainer, err := selectContainer(dc.Spec.Template.Spec.InitContainers, containerName)
	if err != nil {
		return err
	}

	initContainer.VolumeMounts = updateVolumeMounts(initContainer.VolumeMounts, volMount)
	dc.Spec.Template.Spec.InitContainers = append(dc.Spec.Template.Spec.InitContainers, initContainer)
	volumes := dc.Spec.Template.Spec.Volumes
	volumes = updateVolumes(volumes, vol)
	dc.Spec.Template.Spec.Volumes = volumes

	jsonDc, err := json.Marshal(dc)
	if err != nil {
		return err
	}

	_, err = service.appClient.DeploymentConfigs(dc.Namespace).Patch(dc.Name, types.StrategicMergePatchType, jsonDc)
	if err != nil {
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
