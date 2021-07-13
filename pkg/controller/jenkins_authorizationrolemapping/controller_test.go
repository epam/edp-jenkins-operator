package jenkins_authorizationrolemapping

import (
	"context"
	"testing"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_Reconcile(t *testing.T) {
	jarm := v1alpha1.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "nss",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRoleMapping",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsAuthorizationRoleMappingSpec{
			RoleType: "rt",
			Group:    "mke@test.com",
			Roles:    []string{"tolr1", "tooo2"},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jarm).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jarm.ObjectMeta, jarm.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).Return(nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[1], jarm.Spec.Group).Return(nil)

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
}
