package openshift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	appsV1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routeV1Client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	coreV1Api "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-gerrit-operator/v2/pkg/service/helpers"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/model"
	jenkinsDefaultSpec "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins/spec"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/kubernetes"
)

// OpenshiftService struct for Openshift platform service.
type OpenshiftService struct {
	kubernetes.K8SService

	appClient     appsV1client.AppsV1Interface
	routeClient   routeV1Client.RouteV1Interface
	projectClient projectV1Client.ProjectV1Interface
}

const (
	deploymentTypeEnvName           = "DEPLOYMENT_TYPE"
	deploymentConfigsDeploymentType = "deploymentConfigs"
)

// Init initializes OpenshiftService.
func (service *OpenshiftService) Init(config *rest.Config, scheme *runtime.Scheme, k8sClient client.Client) error {
	if err := service.K8SService.Init(config, scheme, k8sClient); err != nil {
		return fmt.Errorf("failed to init K8S platform service: %w", err)
	}

	appClient, err := appsV1client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init apps V1 client for Openshift: %w", err)
	}

	routeClient, err := routeV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init route V1 client for Openshift: %w", err)
	}

	pc, err := projectV1Client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to init project client for Openshift: %w", err)
	}

	service.appClient = appClient
	service.routeClient = routeClient
	service.projectClient = pc

	return nil
}

// GetExternalEndpoint returns hostname and protocol for Route.
func (service *OpenshiftService) GetExternalEndpoint(namespace, name string) (host, routeScheme, path string, err error) {
	route, err := service.routeClient.
		Routes(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return "", "", "", fmt.Errorf("failed to find route %v in namespace %v", name, namespace)
		}

		return "", "", "", fmt.Errorf("failed to get Routes: %w", err)
	}

	specHost := route.Spec.Host

	routeHTTPSScheme := jenkinsDefaultSpec.RouteHTTPScheme

	if route.Spec.TLS.Termination != "" {
		routeHTTPSScheme = jenkinsDefaultSpec.RouteHTTPSScheme
	}

	specPath := strings.TrimRight(route.Spec.Path, platformHelper.UrlCutset)

	return specHost, routeHTTPSScheme, specPath, nil
}

func (service *OpenshiftService) IsDeploymentReady(instance *jenkinsApi.Jenkins) (bool, error) {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		deploymentConfig, err := service.appClient.
			DeploymentConfigs(instance.Namespace).
			Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get DeploymentConfigs: %w", err)
		}

		if deploymentConfig.Status.UpdatedReplicas == 1 && deploymentConfig.Status.AvailableReplicas == 1 {
			return true, nil
		}

		return false, nil
	}

	ready, err := service.K8SService.IsDeploymentReady(instance)
	if err != nil {
		return false, fmt.Errorf("failed to check if deployment is ready: %w", err)
	}

	return ready, nil
}

func (service *OpenshiftService) AddVolumeToInitContainer(instance *jenkinsApi.Jenkins,
	containerName string, vol []coreV1Api.Volume, volMount []coreV1Api.VolumeMount,
) error {
	if os.Getenv(deploymentTypeEnvName) == deploymentConfigsDeploymentType {
		if len(vol) == 0 || len(volMount) == 0 {
			return nil
		}

		deploymentConfig, err := service.appClient.DeploymentConfigs(instance.Namespace).
			Get(context.TODO(), instance.Name, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		initContainer, err := selectContainer(deploymentConfig.Spec.Template.Spec.InitContainers, containerName)
		if err != nil {
			return err
		}

		initContainer.VolumeMounts = updateVolumeMounts(initContainer.VolumeMounts, volMount)
		deploymentConfig.Spec.Template.Spec.InitContainers = append(deploymentConfig.Spec.Template.Spec.InitContainers, initContainer)
		volumes := deploymentConfig.Spec.Template.Spec.Volumes
		volumes = updateVolumes(volumes, vol)
		deploymentConfig.Spec.Template.Spec.Volumes = volumes

		jsonDc, err := json.Marshal(deploymentConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal deployment config: %w", err)
		}

		_, err = service.appClient.
			DeploymentConfigs(deploymentConfig.Namespace).
			Patch(context.TODO(), deploymentConfig.Name, types.StrategicMergePatchType, jsonDc, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to patch DeploymentConfigs: %w", err)
		}

		return nil
	}

	if err := service.K8SService.AddVolumeToInitContainer(instance, containerName, vol, volMount); err != nil {
		return fmt.Errorf("failed to add volume to init container: %w", err)
	}

	return nil
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

	for i := 0; i < len(existing); i++ {
		newer, ok := findVolumeMount(volMount, existing[i].Name)
		if ok {
			covered = append(covered, existing[i].Name)
			out = append(out, newer)

			continue
		}

		out = append(out, existing[i])
	}

	for i := 0; i < len(volMount); i++ {
		if helpers.IsStringInSlice(volMount[i].Name, covered) {
			continue
		}

		covered = append(covered, volMount[i].Name)
		out = append(out, volMount[i])
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

func (*OpenshiftService) CreateStageJSON(stage *cdPipeApi.Stage) (string, error) {
	j := []model.PipelineStage{
		{
			Name:     "deploy",
			StepName: "deploy",
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
		return "", fmt.Errorf("failed to marshal stages: %w", err)
	}

	return string(o), err
}
