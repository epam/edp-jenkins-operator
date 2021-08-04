package jenkins_authorizationrolemapping

import (
	"context"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/pkg/errors"
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
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(&jClient, nil)
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

func TestReconcile_Reconcile_Delete(t *testing.T) {
	jarm := v1alpha1.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
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
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).Return(nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[1], jarm.Spec.Group).Return(nil)

	jClient.On("UnAssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).Return(nil)
	jClient.On("UnAssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[1], jarm.Spec.Group).Return(nil)

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

	var checkInstance v1alpha1.JenkinsAuthorizationRoleMapping
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
		&checkInstance); err != nil {
		t.Fatal(err)
	}

	if len(checkInstance.GetFinalizers()) > 0 {
		t.Fatal("finalizers still set")
	}
}

func TestReconcile_Reconcile_Delete_Failure_UnsetRoles(t *testing.T) {
	jarm := v1alpha1.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
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
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).Return(nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[1], jarm.Spec.Group).Return(nil)

	jClient.On("UnAssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).
		Return(errors.New("unset role fatal"))

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

	var checkInstance v1alpha1.JenkinsAuthorizationRoleMapping
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
		&checkInstance); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkInstance.Status.Value, "unset role fatal") {
		t.Log(checkInstance.Status.Value)
		t.Fatal("no unset error in instance status")
	}
}

func TestSpecUpdated(t *testing.T) {
	jarm1 := v1alpha1.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
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

	jarm2 := v1alpha1.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "nss",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRoleMapping",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsAuthorizationRoleMappingSpec{
			RoleType: "rt2",
			Group:    "mke@test.com",
			Roles:    []string{"tolr1", "tooo2"},
		},
	}

	if !specUpdated(event.UpdateEvent{ObjectOld: &jarm1, ObjectNew: &jarm2}) {
		t.Fatal("spec is updated")
	}
}
