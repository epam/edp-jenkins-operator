package jenkins

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"io/ioutil"
	"jenkins-operator/pkg/apis/v2/v1alpha1"
	"jenkins-operator/pkg/helper"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"text/template"

	jenkinsClient "jenkins-operator/pkg/client/jenkins"
	jenkinsDefaultSpec "jenkins-operator/pkg/service/jenkins/spec"
	"jenkins-operator/pkg/service/platform"
	platformHelper "jenkins-operator/pkg/service/platform/helper"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	jenkinsAdminCredentialsSecretPostfix = "admin-password"
	jenkinsAdminTokenSecretPostfix       = "admin-token"
	jenkinsDefaultConfigsAbsolutePath    = "/usr/local/configs/"
	jenkinsDefaultScriptsDirectory       = "scripts"
	jenkinsDefaultSlavesDirectory        = "slaves"
	jenkinsDefaultTemplatesDirectory     = "templates"
	jenkinsDefaultScriptsAbsolutePath    = jenkinsDefaultConfigsAbsolutePath + jenkinsDefaultScriptsDirectory
	jenkinsDefaultSlavesAbsolutePath     = jenkinsDefaultConfigsAbsolutePath + jenkinsDefaultSlavesDirectory
	jenkinsDefaultTemplatesAbsolutePath  = jenkinsDefaultConfigsAbsolutePath + jenkinsDefaultTemplatesDirectory
	localConfigsRelativePath             = "configs"
	jenkinsSlavesConfigMapName           = "jenkins-slaves"
	jenkinsSharedLibrariesConfigFileName = "config-shared-libraries.tmpl"
	jenkinsDefaultScriptConfigMapKey     = "context"
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
func NewJenkinsService(platformService platform.PlatformService, k8sClient client.Client, k8sScheme *runtime.Scheme) JenkinsService {
	return JenkinsServiceImpl{platformService: platformService, k8sClient: k8sClient, k8sScheme: k8sScheme}
}

// JenkinsServiceImpl struct fo Jenkins EDP Component
type JenkinsServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
	k8sScheme       *runtime.Scheme
}

func (j JenkinsServiceImpl) setAdminSecretInStatus(instance *v1alpha1.Jenkins, value *string) (*v1alpha1.Jenkins, error) {
	instance.Status.AdminSecretName = value
	err := j.k8sClient.Status().Update(context.TODO(), instance)
	if err != nil {
		err := j.k8sClient.Update(context.TODO(), instance)
		if err != nil {
			return instance, errors.Wrap(err, "Couldn't set admin secret name in status")
		}
	}
	return instance, nil
}

func (j JenkinsServiceImpl) createJenkinsScript(instance v1alpha1.Jenkins, name string, configMapName string) (*v1alpha1.JenkinsScript, error) {
	jenkinsScriptObject := &v1alpha1.JenkinsScript{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
		},
		Spec: v1alpha1.JenkinsScriptSpec{
			SourceCmName: configMapName,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, jenkinsScriptObject, j.k8sScheme); err != nil {
		return nil, errors.Wrapf(err, "Couldn't set reference for JenkinsScript %v object", jenkinsScriptObject.Name)
	}

	nsn := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      name,
	}

	err := j.k8sClient.Get(context.TODO(), nsn, jenkinsScriptObject)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err := j.k8sClient.Create(context.TODO(), jenkinsScriptObject)
			if err != nil {
				return nil, errors.Wrapf(err, "Couldn't create Jenkins Script object %v", name)
			}
		}
	}

	return jenkinsScriptObject, nil
}

func (j JenkinsServiceImpl) createSecret(instance v1alpha1.Jenkins, secretName string, username string, password *string) error {
	var secretPassword string
	if password == nil {
		secretPassword = uniuri.New()
	} else {
		secretPassword = *password
	}
	secretData := map[string][]byte{
		"username": []byte(username),
		"password": []byte(secretPassword),
	}

	err := j.platformService.CreateSecret(instance, secretName, secretData)
	if err != nil {
		return errors.Wrapf(err, "Failed to create Secret %v", secretName)
	}
	return nil
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
	jc, err := jenkinsClient.InitJenkinsClient(&instance, j.platformService)
	if err != nil {
		return &instance, false, errors.Wrap(err, "Failed to init Jenkins REST client")
	}
	if jc == nil {
		return &instance, false, errors.Wrap(err, "Jenkins returns nil client")
	}

	adminTokenSecretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsAdminTokenSecretPostfix)
	adminTokenSecret, err := j.platformService.GetSecretData(instance.Namespace, adminTokenSecretName)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "Unable to get admin token secret for %v", instance.Name)
	}

	if adminTokenSecret == nil {
		token, err := jc.GetAdminToken()
		if err != nil {
			return &instance, false, errors.Wrap(err, "Failed to get token from admin user")
		}

		err = j.createSecret(instance, adminTokenSecretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, token)
		if err != nil {
			return &instance, false, err
		}

		updatedInstance, err := j.setAdminSecretInStatus(&instance, &adminTokenSecretName)
		if err != nil {
			return &instance, false, err
		}
		instance = *updatedInstance
	}

	executableFilePath := helper.GetExecutableFilePath()
	jenkinsScriptsDirectoryPath := jenkinsDefaultScriptsAbsolutePath

	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		jenkinsScriptsDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, jenkinsDefaultScriptsDirectory)
	}

	directory, err := ioutil.ReadDir(jenkinsScriptsDirectoryPath)
	if err != nil {
		return &instance, false, errors.Wrapf(err, fmt.Sprintf("Couldn't read directory %v", jenkinsScriptsDirectoryPath))
	}

	for _, file := range directory {
		configMapName := fmt.Sprintf("%v-%v", instance.Name, file.Name())
		configMapKey := jenkinsDefaultScriptConfigMapKey

		jenkinsScript, err := j.createJenkinsScript(instance, file.Name(), configMapName)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "Couldn't create Jenkins Script %v", file.Name())
		}
		err = j.platformService.CreateConfigMapFromFileOrDir(instance, configMapName, &configMapKey, fmt.Sprintf("%v/%v", jenkinsScriptsDirectoryPath, file.Name()), jenkinsScript)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "Couldn't create configs-map %v in namespace %v.", configMapName, instance.Namespace)
		}
	}

	jenkinsSlavesDirectoryPath := jenkinsDefaultSlavesAbsolutePath

	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		jenkinsSlavesDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, jenkinsDefaultSlavesDirectory)
	}

	directory, err = ioutil.ReadDir(jenkinsSlavesDirectoryPath)
	if err != nil {
		return nil, false, errors.Wrapf(err, fmt.Sprintf("Couldn't read directory %v", jenkinsScriptsDirectoryPath))
	}

	JenkinsSlavesConfigmapLabels := map[string]string{
		"role": "jenkins-slave",
	}

	err = j.platformService.CreateConfigMapFromFileOrDir(instance, jenkinsSlavesConfigMapName, nil,
		jenkinsSlavesDirectoryPath, &instance, JenkinsSlavesConfigmapLabels)
	if err != nil {
		return nil, false, errors.Wrapf(err, "Couldn't create configs-map %v in namespace %v.",
			jenkinsSlavesConfigMapName, instance.Namespace)
	}

	jenkinsTemplatesDirectoryPath := jenkinsDefaultTemplatesAbsolutePath
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		jenkinsTemplatesDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, jenkinsDefaultTemplatesDirectory)
	}

	var sharedLibrariesScriptContext bytes.Buffer
	templateAbsolutePath := fmt.Sprintf("%v/%v", jenkinsTemplatesDirectoryPath, jenkinsSharedLibrariesConfigFileName)
	t := template.Must(template.New(jenkinsSharedLibrariesConfigFileName).ParseFiles(templateAbsolutePath))
	err = t.Execute(&sharedLibrariesScriptContext, instance.Spec.SharedLibraries)
	if err != nil {
		return nil, false, errors.Wrapf(err, "Couldn't parse template %v", jenkinsSharedLibrariesConfigFileName)
	}

	jenkinsScriptName := "config-shared-libraries"
	configMapName := fmt.Sprintf("%v-%v", instance.Name, jenkinsScriptName)

	jenkinsScript, err := j.createJenkinsScript(instance, jenkinsScriptName, configMapName)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "Couldn't create Jenkins Script %v", jenkinsScriptName)
	}
	labels := platformHelper.GenerateLabels(instance.Name)
	configMapData := map[string]string{jenkinsDefaultScriptConfigMapKey: sharedLibrariesScriptContext.String()}
	err = j.platformService.CreateConfigMapFromData(instance, configMapName, configMapData, labels, jenkinsScript)
	if err != nil {
		return &instance, false, err
	}

	return &instance, true, nil
}

// Install performs installation of Jenkins
func (j JenkinsServiceImpl) Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error) {
	secretName := fmt.Sprintf("%v-%v", instance.Name, jenkinsAdminCredentialsSecretPostfix)
	err := j.createSecret(instance, secretName, jenkinsDefaultSpec.JenkinsDefaultAdminUser, nil)
	if err != nil {
		return &instance, err
	}
	if instance.Status.AdminSecretName == nil {
		updatedInstance, err := j.setAdminSecretInStatus(&instance, &secretName)
		if err != nil {
			return &instance, err
		}
		instance = *updatedInstance
	}

	err = j.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Service Account %v", instance.Name)
	}

	err = j.platformService.CreateUserRoleBinding(instance, instance.Name, "edit", platformHelper.ClusterRole)
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to create Role Binding %v", instance.Name)
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
