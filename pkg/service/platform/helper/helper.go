package helper

import (
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	authV1Api "github.com/openshift/api/authorization/v1"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	ClusterRole string = "clusterrole"
)

// GenerateLabels returns map with labels for k8s objects
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func GetNewRoleObject(instance v1alpha1.Jenkins, name string, binding string, kind string) (*authV1Api.RoleBinding, error) {
	switch strings.ToLower(kind) {
	case ClusterRole:
		return NewClusterRoleBindingObject(instance, name, binding), nil
	default:
		return &authV1Api.RoleBinding{}, errors.New(fmt.Sprintf("Wrong role kind %s! Cant't create rolebinding", kind))
	}
}

func NewClusterRoleBindingObject(instance v1alpha1.Jenkins, name string, binding string) *authV1Api.RoleBinding {
	return &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
		},
		RoleRef: coreV1Api.ObjectReference{
			APIVersion: "rbac.authorization.k8s.io",
			Kind:       "ClusterRole",
			Name:       binding,
		},
		Subjects: []coreV1Api.ObjectReference{
			{
				Kind: "ServiceAccount",
				Name: instance.Name,
			},
		},
	}
}
