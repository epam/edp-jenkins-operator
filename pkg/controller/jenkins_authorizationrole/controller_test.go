package jenkins_authorizationrole

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
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
	jBuilder.On("MakeNewClient", jar.Spec.OwnerName).Return(&jClient, nil)

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

func TestReconcile_Reconcile_Delete(t *testing.T) {
	jar := v1alpha1.JenkinsAuthorizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRole",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsAuthorizationRoleSpec{
			Name:        "role-name1",
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
	jBuilder.On("MakeNewClient", jar.Spec.OwnerName).Return(&jClient, nil)

	jClient.On("AddRole", jar.Spec.RoleType, jar.Spec.Name, jar.Spec.Pattern, jar.Spec.Permissions).Return(nil)
	jClient.On("RemoveRoles", jar.Spec.RoleType, []string{jar.Spec.Name}).Return(nil)

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

	var checkInstance v1alpha1.JenkinsAuthorizationRole
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: jar.Namespace, Name: jar.Name}, &checkInstance); err != nil {
		t.Fatal(err)
	}

	if len(checkInstance.GetFinalizers()) > 0 {
		t.Fatal("finalizers still exists")
	}
}

func TestReconcile_Reconcile_Delete_FailureRemoveRoles(t *testing.T) {
	jar := v1alpha1.JenkinsAuthorizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRole",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsAuthorizationRoleSpec{
			Name:        "role-name1",
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
	jBuilder.On("MakeNewClient", jar.Spec.OwnerName).Return(&jClient, nil)

	jClient.On("AddRole", jar.Spec.RoleType, jar.Spec.Name, jar.Spec.Pattern, jar.Spec.Permissions).Return(nil)
	jClient.On("RemoveRoles", jar.Spec.RoleType, []string{jar.Spec.Name}).Return(errors.New("remove roles failure"))

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

	var checkInstance v1alpha1.JenkinsAuthorizationRole
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: jar.Namespace, Name: jar.Name}, &checkInstance); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkInstance.Status.Value, "remove roles failure") {
		t.Log(checkInstance.Status.Value)
		t.Fatal("failure status is not set")
	}
}

func TestSpecUpdated(t *testing.T) {
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

	if specUpdated(event.UpdateEvent{ObjectNew: &jar, ObjectOld: &jar}) {
		t.Fatal("spec is not updated")
	}
}
