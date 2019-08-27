package jenkins

import (
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	"nexus-operator/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("jenkins_service")

// JenkinsService interface for Jenkins EDP component
type JenkinsService interface {
	Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error)
	ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
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
func (n JenkinsServiceImpl) Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	return &instance, nil
}

// ExposeConfiguration performs exposing Jenkins configuration for other EDP components
func (n JenkinsServiceImpl) ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	return &instance, nil
}

// Configure performs self-configuration of Jenkins
func (n JenkinsServiceImpl) Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	return &instance, true, nil
}

// Install performs installation of Jenkins
func (n JenkinsServiceImpl) Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	return &instance, nil
}
