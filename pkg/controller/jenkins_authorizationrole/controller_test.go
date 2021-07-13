package jenkins_authorizationrole

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
	jar := v1alpha1.JenkinsAuthorizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "nss",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRole",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsAuthorizationRoleSpec{
			Name:        "test",
			RoleType:    "rt",
			Permissions: []string{"foo", "bat"},
			Pattern:     "regex",
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jar).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jar.ObjectMeta, jar.Spec.OwnerName).Return(&jClient, nil)

	jClient.On("AddRole", jar.Spec.RoleType, jar.Spec.Name, jar.Spec.Pattern, jar.Spec.Permissions).Return(nil)

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jar.Namespace, Name: jar.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
}
