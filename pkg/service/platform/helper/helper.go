package helper

import (
	"bytes"
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/helper"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	authV1Api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/template"
)

const (
	ClusterRole                string = "clusterrole"
	Role                       string = "role"
	defaultConfigsAbsolutePath        = "/usr/local/configs/"
	localConfigsRelativePath          = "configs"
)

type JenkinsScriptData struct {
	RealmName              string
	KeycloakUrl            string
	KeycloakClientName     string
	JenkinsUrl             string
	JenkinsSharedLibraries []v1alpha1.JenkinsSharedLibraries
}

// GenerateLabels returns map with labels for k8s objects
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func GetNewRoleBindingObject(instance v1alpha1.Jenkins, roleBindingName string, roleName string, kind string) (*authV1Api.RoleBinding, error) {
	return &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: instance.Namespace,
		},
		RoleRef: authV1Api.RoleRef{
			Kind: kind,
			Name: roleName,
		},
		Subjects: []authV1Api.Subject{
			{
				Kind: "ServiceAccount",
				Name: instance.Name,
			},
		},
	}, nil
}

func GetNewClusterRoleBindingObject(instance v1alpha1.Jenkins, clusterRoleBindingName string, clusterRoleName string) (*authV1Api.ClusterRoleBinding, error) {
	return &authV1Api.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterRoleBindingName,
			Namespace: instance.Namespace,
		},
		RoleRef: authV1Api.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRoleName,
		},
		Subjects: []authV1Api.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
		},
	}, nil
}

func createPath(directory string, localRun bool) (string, error) {
	if localRun {
		executableFilePath, err := helper.GetExecutableFilePath()
		if err != nil {
			return "", errors.Wrapf(err, "Unable to get executable file path")
		}
		templatePath := fmt.Sprintf("%v/../%v/%v", executableFilePath, localConfigsRelativePath, directory)
		return templatePath, nil
	}

	templatePath := fmt.Sprintf("%s/%s", defaultConfigsAbsolutePath, directory)
	return templatePath, nil

}

func checkIfRunningLocally() bool {
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		return true
	}
	return false
}

func CreatePathToTemplateDirectory(directory string) (string, error) {
	localRun := checkIfRunningLocally()
	return createPath(directory, localRun)
}

func ParseTemplate(data JenkinsScriptData, pathToTemplate string, templateName string) (bytes.Buffer, error) {
	var ScriptContext bytes.Buffer

	if !helper.FileExists(pathToTemplate) {
		errMsg := fmt.Sprintf("Template file not found in pathToTemplate specificed! Path: %s", pathToTemplate)
		return bytes.Buffer{}, errors.New(errMsg)
	}
	t := template.Must(template.New(templateName).ParseFiles(pathToTemplate))
	err := t.Execute(&ScriptContext, data)
	if err != nil {
		return bytes.Buffer{}, errors.Wrapf(err, "Couldn't parse template %v", templateName)
	}

	return ScriptContext, nil
}
