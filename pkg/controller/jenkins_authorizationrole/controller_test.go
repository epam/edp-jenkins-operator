package jenkins_authorizationrole

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
)

func getTestJenkinsAuthorizationRole() *v1alpha1.JenkinsAuthorizationRole {
	return &v1alpha1.JenkinsAuthorizationRole{
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
}

func TestReconcile_Reconcile(t *testing.T) {
	jar := getTestJenkinsAuthorizationRole()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jar).Build()
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
	jar := getTestJenkinsAuthorizationRole()
	jar.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jar).Build()
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
	jar := getTestJenkinsAuthorizationRole()
	jar.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jar).Build()
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
	jar := getTestJenkinsAuthorizationRole()

	if specUpdated(event.UpdateEvent{ObjectNew: jar, ObjectOld: jar}) {
		t.Fatal("spec is not updated")
	}
}

func TestNewReconciler(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	ps := pmock.PlatformService{}
	lg := helper.LoggerMock{}

	rec := NewReconciler(k8sClient, &lg, &ps)
	if rec == nil {
		t.Fatal("reconciler is not inited")
	}

	if rec.client != k8sClient || rec.log != &lg {
		t.Fatal("wrong reconciler params")
	}
}

func TestReconcile_Reconcile_FailureNotFound(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &v1alpha1.JenkinsAuthorizationRole{})

	k8sClient := fake.NewClientBuilder().Build()
	logger := helper.LoggerMock{}
	r := Reconcile{
		client: k8sClient,
		log:    &logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: "ns1", Name: "name1"},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if logger.LastInfo() != "instance not found" {
		t.Fatal("not found error is not logged")
	}
}

func TestReconcile_Reconcile_FailureInitJenkinsClient(t *testing.T) {
	jar := getTestJenkinsAuthorizationRole()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jar).Build()
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jar.Spec.OwnerName).Return(nil,
		errors.New("make new client fatal"))

	logger := helper.LoggerMock{}
	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jar.Namespace, Name: jar.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcile_Reconcile_FailureAddRole(t *testing.T) {
	jar := getTestJenkinsAuthorizationRole()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jar)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jar).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jar.Spec.OwnerName).Return(&jClient, nil)

	jClient.On("AddRole", jar.Spec.RoleType, jar.Spec.Name, jar.Spec.Pattern, jar.Spec.Permissions).
		Return(errors.New("add role fatal"))

	logger := helper.LoggerMock{}

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jar.Namespace, Name: jar.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	lastErr := logger.LastError()
	if lastErr == nil {
		t.Fatal("no error logged")
	}

	if !strings.Contains(lastErr.Error(), "add role fatal") {
		t.Fatalf("wrong error returned: %s", lastErr.Error())
	}
}
