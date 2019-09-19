package helper

import (
	"github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	authV1Api "github.com/openshift/api/authorization/v1"
	coreV1Api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	ClusterRole string = "clusterrole"
	Role        string = "role"
)

// GenerateLabels returns map with labels for k8s objects
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

func GetNewRoleBindingObject(instance v1alpha1.Jenkins, roleBindingName string, roleName string, kind string) (*authV1Api.RoleBinding, error) {
	var roleNamespace string
	switch strings.ToLower(kind) {
	case ClusterRole:
		roleNamespace = ""
	case Role:
		roleNamespace = instance.Namespace
	}
	return &authV1Api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: instance.Namespace,
		},
		RoleRef: coreV1Api.ObjectReference{
			APIVersion: "rbac.authorization.k8s.io",
			Kind:       kind,
			Name:       roleName,
			Namespace:  roleNamespace,
		},
		Subjects: []coreV1Api.ObjectReference{
			{
				Kind: "ServiceAccount",
				Name: instance.Name,
			},
		},
	}, nil
}
