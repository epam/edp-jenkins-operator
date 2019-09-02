package jenkins

import (
	"context"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	jenkinsDefaultSpec "jenkins-operator/pkg/service/jenkins/spec"
	"jenkins-operator/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	jenkinsAdminCredentialsSecretPostfix = "admin-password"
)

var log = logf.Log.WithName("jenkins_service")

// JenkinsService interface for Jenkins EDP component
type JenkinsService interface {
	Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error)
	ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	IsDeploymentConfigReady(instance v1alpha1.Jenkins) (bool, error)
}

// NewJenkinsService function that returns JenkinsService implementation
func NewJenkinsService(platformService platform.PlatformService, k8sClient client.Client) JenkinsService {
	return JenkinsServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

// JenkinsServiceImpl struct fo Jenkins EDP Component
type JenkinsServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
}

// Integration performs integration Jenkins with other EDP components
func (j JenkinsServiceImpl) Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	return &instance, nil
}

// ExposeConfiguration performs exposing Jenkins configuration for other EDP components
func (j JenkinsServiceImpl) ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	return &instance, nil
}

// Configure performs self-configuration of Jenkins
func (j JenkinsServiceImpl) Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	return &instance, true, nil
}

// Install performs installation of Jenkins
func (j JenkinsServiceImpl) Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	adminSecret := map[string][]byte{
		"username": []byte(jenkinsDefaultSpec.JenkinsDefaultAdminUser),
		"password": []byte(uniuri.New()),
	}

	adminSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsAdminCredentialsSecretPostfix)
	err := j.platformService.CreateSecret(instance, adminSecretName, adminSecret)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Secret %v", adminSecretName)
	}

	instance.Status.AdminSecretName = &adminSecretName
	err = j.k8sClient.Status().Update(context.TODO(), &instance)
	if err != nil {
		err := j.k8sClient.Update(context.TODO(), &instance)
		if err != nil {
			return &instance, errors.Wrap(err, "Couldn't set admin secret name in status")
		}
	}

	err = j.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Service Account %v", instance.Name)
	}

	err = j.platformService.CreatePersistentVolumeClaim(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Volume for %v", instance.Name)
	}

	err = j.platformService.CreateService(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Service for %v/%v", instance.Namespace, instance.Name)
	}

	err = j.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to create Route.")
	}

	err = j.platformService.CreateDeployConf(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "Failed to create Deployment Config.")
	}

	return &instance, nil
}

// IsDeploymentConfigReady check if DC for Nexus is ready
func (j JenkinsServiceImpl) IsDeploymentConfigReady(instance v1alpha1.Jenkins) (bool, error) {
	nexusIsReady := false

	nexusDc, err := j.platformService.GetDeploymentConfig(instance)
	if err != nil {
		return nexusIsReady, err
	}

	if nexusDc.Status.AvailableReplicas == 1 {
		nexusIsReady = true
	}

	return nexusIsReady, nil
}
