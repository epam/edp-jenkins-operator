package jenkins

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsScriptHelper "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkinsscript/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/helper"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"io/ioutil"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"text/template"

	gerritApi "github.com/epmd-edp/gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	gerritSpec "github.com/epmd-edp/gerrit-operator/v2/pkg/service/gerrit/spec"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	jenkinsDefaultSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform/helper"
	keycloakApi "github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	coreV1Api "k8s.io/api/core/v1"
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
	jenkinsKeycloakConfigFileName        = "config-keycloak.tmpl"
	jenkinsDefaultScriptConfigMapKey     = "context"
	sshKeyDefaultMountPath               = "/var/lib/jenkins/.ssh"
	gerritDefaultName                    = "gerrit"
)

var log = logf.Log.WithName("jenkins_service")

// JenkinsService interface for Jenkins EDP component
type JenkinsService interface {
	Install(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Configure(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error)
	ExposeConfiguration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, error)
	Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error)
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

func (j JenkinsServiceImpl) setAdminSecretInStatus(instance *v1alpha1.Jenkins, value string) (*v1alpha1.Jenkins, error) {
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

func (j JenkinsServiceImpl) createKeycloakClient(instance v1alpha1.Jenkins, name string) (*keycloakApi.KeycloakClient, error) {
	keycloakClientObject := &keycloakApi.KeycloakClient{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1.edp.epam.com/v1alpha1",
			Kind:       "KeycloakClient",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: keycloakApi.KeycloakClientSpec{
			TargetRealm: instance.Spec.KeycloakSpec.Realm,
			Public:      true,
			ClientId:    instance.Name,
		},
	}

	if err := controllerutil.SetControllerReference(&instance, keycloakClientObject, j.k8sScheme); err != nil {
		return nil, errors.Wrapf(err, "Couldn't set reference for JenkinsScript %v object", keycloakClientObject.Name)
	}

	nsn := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      name,
	}

	err := j.k8sClient.Get(context.TODO(), nsn, keycloakClientObject)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err := j.k8sClient.Create(context.TODO(), keycloakClientObject)
			if err != nil {
				return nil, errors.Wrapf(err, "Couldn't create Keycloak client object %v", name)
			}
			log.Info("Keycloak client CR created")
			return keycloakClientObject, nil
		}
		return nil, errors.Wrapf(err, "Couldn't get Keycloak client object %v", name)
	}

	return keycloakClientObject, nil
}

func (j JenkinsServiceImpl) mountGerritCredentials(instance v1alpha1.Jenkins, gerritName string) error {
	options := client.ListOptions{Namespace: instance.Namespace}
	list := &gerritApi.GerritList{}

	err := j.k8sClient.List(context.TODO(), &options, list)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))
			return nil
		} else {
			return errors.Wrapf(err, fmt.Sprintf("Unable to get Gerrit CRs in namespace %v", instance.Namespace))
		}
	}

	if len(list.Items) == 0 {
		log.Info(fmt.Sprintf("Gerrit installation is not found in namespace %v", instance.Namespace))
		return nil
	}

	gerritCrObject := &list.Items[0]
	gerritSpecName := fmt.Sprintf("%v/%v", gerritSpec.EdpAnnotationsPrefix, gerritSpec.EdpCiUSerSshKeySuffix)
	if val, ok := gerritCrObject.ObjectMeta.Annotations[gerritSpecName]; ok {
		dcJenkins, err := j.platformService.GetDeploymentConfig(instance)
		if err != nil {
			return err
		}

		volMount := []coreV1Api.VolumeMount{
			{
				Name:      val,
				MountPath: sshKeyDefaultMountPath,
				ReadOnly:  true,
			},
		}

		mode := int32(400)
		vol := []coreV1Api.Volume{
			{
				Name: val,
				VolumeSource: coreV1Api.VolumeSource{
					Secret: &coreV1Api.SecretVolumeSource{
						SecretName:  val,
						DefaultMode: &mode,
						Items: []coreV1Api.KeyToPath{
							{
								Key:  "id_rsa",
								Path: "id_rsa",
								Mode: &mode,
							},
						},
					},
				},
			},
		}

		err = j.platformService.PatchDeployConfVol(instance, dcJenkins, vol, volMount)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("Unable to patch Jenkins DC in namespace %v", instance.Namespace))
		}
	}
	return nil
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

//setAnnotation add key:value to current resource annotation
func (j JenkinsServiceImpl) setAnnotation(instance *v1alpha1.Jenkins, key string, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}

// Integration performs integration Jenkins with other EDP components
func (j JenkinsServiceImpl) Integration(instance v1alpha1.Jenkins) (*v1alpha1.Jenkins, bool, error) {
	if instance.Spec.KeycloakSpec.Enabled {
		_, err := j.createKeycloakClient(instance, instance.Name)
		if err != nil {
			return &instance, false, errors.Wrapf(err, fmt.Sprintf("Failed to create Keycloak Client"))
		}

		jenkinsTemplatesDirectoryPath := jenkinsDefaultTemplatesAbsolutePath
		executableFilePath := helper.GetExecutableFilePath()
		if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
			jenkinsTemplatesDirectoryPath = fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, jenkinsDefaultTemplatesDirectory)
		}

		var keycloakConfigScriptContext bytes.Buffer
		templateAbsolutePath := fmt.Sprintf("%v/%v", jenkinsTemplatesDirectoryPath, jenkinsKeycloakConfigFileName)
		t := template.Must(template.New(jenkinsKeycloakConfigFileName).ParseFiles(templateAbsolutePath))

		err = t.Execute(&keycloakConfigScriptContext, instance)
		if err != nil {
			return nil, false, errors.Wrapf(err, "Couldn't parse template %v", jenkinsKeycloakConfigFileName)
		}

		jenkinsScriptName := "config-keycloak"
		configMapName := fmt.Sprintf("%v-%v", instance.Name, jenkinsScriptName)

		jenkinsScript, err := jenkinsScriptHelper.CreateJenkinsScript(
			jenkinsScriptHelper.K8sClient{Client: j.k8sClient, Scheme: j.k8sScheme},
			jenkinsScriptName,
			configMapName,
			instance.Namespace,
			true,
			&instance)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "Couldn't create Jenkins Script %v", jenkinsScriptName)
		}
		labels := platformHelper.GenerateLabels(instance.Name)
		configMapData := map[string]string{jenkinsDefaultScriptConfigMapKey: keycloakConfigScriptContext.String()}
		err = j.platformService.CreateConfigMapFromData(instance, configMapName, configMapData, labels, jenkinsScript)
		if err != nil {
			return &instance, false, err
		}
	}

	err := j.mountGerritCredentials(instance, gerritDefaultName)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "Failed to mount Gerrit credentials")
	}

	return &instance, true, nil
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

		adminTokenAnnotationKey := helper.GenerateAnnotationKey(jenkinsDefaultSpec.JenkinsTokenAnnotationSuffix)
		j.setAnnotation(&instance, adminTokenAnnotationKey, adminTokenSecretName)

		err = j.k8sClient.Update(context.TODO(), &instance)
		if err != nil {
			return &instance, false, err
		}

		updatedInstance, err := j.setAdminSecretInStatus(&instance, adminTokenSecretName)
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
		configMapKey := jenkinsScriptHelper.JenkinsDefaultScriptConfigMapKey

		jenkinsScript, err := jenkinsScriptHelper.CreateJenkinsScript(
			jenkinsScriptHelper.K8sClient{Client: j.k8sClient, Scheme: j.k8sScheme},
			file.Name(),
			configMapName,
			instance.Namespace,
			true,
			&instance)
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
		return &instance, false, errors.Wrapf(err, "Couldn't parse template %v", jenkinsSharedLibrariesConfigFileName)
	}

	jenkinsScriptName := "config-shared-libraries"
	configMapName := fmt.Sprintf("%v-%v", instance.Name, jenkinsScriptName)
	jenkinsScript, err := jenkinsScriptHelper.CreateJenkinsScript(
		jenkinsScriptHelper.K8sClient{Client: j.k8sClient, Scheme: j.k8sScheme},
		jenkinsScriptName,
		configMapName,
		instance.Namespace,
		true,
		&instance)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "Couldn't create Jenkins Script %v", jenkinsScriptName)
	}
	labels := platformHelper.GenerateLabels(instance.Name)
	configMapData := map[string]string{jenkinsScriptHelper.JenkinsDefaultScriptConfigMapKey: sharedLibrariesScriptContext.String()}
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
	if instance.Status.AdminSecretName == "" {
		updatedInstance, err := j.setAdminSecretInStatus(&instance, secretName)
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

// IsDeploymentConfigReady check if DC for Jenkins is ready
func (j JenkinsServiceImpl) IsDeploymentConfigReady(instance v1alpha1.Jenkins) (bool, error) {
	jenkinsIsReady := false

	jenkinsDc, err := j.platformService.GetDeploymentConfig(instance)
	if err != nil {
		return jenkinsIsReady, err
	}

	if jenkinsDc.Status.UpdatedReplicas == 1 && jenkinsDc.Status.AvailableReplicas == 1 {
		jenkinsIsReady = true
	}

	return jenkinsIsReady, nil
}
