package openshift

import (
	"encoding/json"
	"fmt"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/model"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/kubernetes"
	edpv1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epmd-edp/gerrit-operator/v2/pkg/service/helpers"
	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"

	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"

	projectV1 "github.com/openshift/api/project/v1"
)

var log = logf.Log.WithName("platform")

// OpenshiftService struct for Openshift platform service
type OpenshiftService struct {
	kubernetes.K8SService

	appClient     appsV1client.AppsV1Client
	routeClient   routeV1Client.RouteV1Client
	projectClient projectV1Client.ProjectV1Client
}

// Init initializes OpenshiftService
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient *client.Client) error {
	err := service.K8SService.Init(config, scheme, k8sClient)
	if err != nil {
		return errors.Wrap(err, "Failed to init K8S platform service")
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init apps V1 client for Openshift")
	}

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init route V1 client for Openshift")
	}

	pc, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to init project client for Openshift")
	}

	service.appClient = *appClient
	service.routeClient = *routeClient
	service.projectClient = *pc

	return nil
}

// GetExternalEndpoint returns hostname and protocol for Route
func (service OpenshiftService) GetExternalEndpoint(namespace string, name string) (string, string, string, error) {
	route, err := service.routeClient.Routes(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", "", "", errors.New(fmt.Sprintf("Route %v in namespace %v not found", name, namespace))
	} else if err != nil {
		return "", "", "", err
	}

	var routeScheme = jenkinsDefaultSpec.RouteHTTPScheme
	if route.Spec.TLS.Termination != "" {
		routeScheme = jenkinsDefaultSpec.RouteHTTPSScheme
	}
	return route.Spec.Host, routeScheme, strings.TrimRight(route.Spec.Path, platformHelper.UrlCutset), nil
}

func (service OpenshiftService) IsDeploymentReady(instance v1alpha1.Jenkins) (bool, error) {
	deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1 {
		return true, nil
	}

	return false, nil
}

func (service OpenshiftService) AddVolumeToInitContainer(instance v1alpha1.Jenkins,
	containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount) error {

	if len(vol) == 0 || len(volMount) == 0 {
		return nil
	}

	dc, err := service.appClient.DeploymentConfigs(instance.Namespace).Get(instance.Name, metav1.GetOptions{})
	if err != nil {
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

func (s OpenshiftService) CreateStageJSON(cr edpv1alpha1.Stage) (string, error) {
	j := []model.PipelineStage{
		{
			Name:     "deploy",
			StepName: "deploy",
		},
	}

	for _, ps := range cr.Spec.QualityGates {
		i := model.PipelineStage{
			Name:     ps.QualityGateType,
			StepName: ps.StepName,
		}

		j = append(j, i)
	}
	j = append(j, model.PipelineStage{Name: "promote-images", StepName: "promote-images"})

	o, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(o), err
}

func (s OpenshiftService) CreateProject(name string) error {
	log.V(2).Info("start sending request to create project...", "name", name)
	_, err := s.projectClient.ProjectRequests().Create(
		&projectV1.ProjectRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Description: "deploy project for stage",
		},
	)
	return err
}

func (s OpenshiftService) DeleteProject(name string) error {
	return s.projectClient.Projects().Delete(name, metav1.NewDeleteOptions(0))
}
