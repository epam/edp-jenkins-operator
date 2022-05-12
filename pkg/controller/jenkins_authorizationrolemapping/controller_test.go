package jenkins_authorizationrolemapping

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
)

func getTestJenkinsAuthorizationRoleMapping() *jenkinsApi.JenkinsAuthorizationRoleMapping {
	return &jenkinsApi.JenkinsAuthorizationRoleMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "nss",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsAuthorizationRoleMapping",
			APIVersion: "apps/v1",
		},
		Spec: jenkinsApi.JenkinsAuthorizationRoleMappingSpec{
			RoleType: "rt",
			Group:    "mke@test.com",
			Roles:    []string{"tolr1", "tooo2"},
		},
	}
}

func TestReconcile_Reconcile(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
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
	jarm := getTestJenkinsAuthorizationRoleMapping()
	jarm.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
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

	var checkInstance jenkinsApi.JenkinsAuthorizationRoleMapping
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
		&checkInstance); err != nil {
		t.Fatal(err)
	}

	if len(checkInstance.GetFinalizers()) > 0 {
		t.Fatal("finalizers still set")
	}
}

func TestReconcile_Reconcile_Delete_Failure_UnsetRoles(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()
	jarm.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
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

	var checkInstance jenkinsApi.JenkinsAuthorizationRoleMapping
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
	jarm1 := getTestJenkinsAuthorizationRoleMapping()
	jarm2 := getTestJenkinsAuthorizationRoleMapping()
	jarm2.Spec.RoleType = "rt123"

	if !specUpdated(event.UpdateEvent{ObjectOld: jarm1, ObjectNew: jarm2}) {
		t.Fatal("spec is updated")
	}
}

func TestNewReconciler(t *testing.T) {
	lg := helper.LoggerMock{}
	k8sClient := fake.NewClientBuilder().Build()
	ps := pmock.PlatformService{}

	rec := NewReconciler(k8sClient, &lg, &ps)
	if rec == nil {
		t.Fatal("reconciler is not inited")
	}

	if rec.log != &lg || rec.client != k8sClient {
		t.Fatal("wrong reconciler params")
	}
}

func TestReconcile_Reconcile_FailureNotFound(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
	logger := helper.LoggerMock{}

	r := Reconcile{
		client: k8sClient,
		log:    &logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jarm.Namespace, Name: "baz"},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if logger.LastInfo() != "instance not found" {
		t.Fatal("not found error is not logged")
	}
}

func TestReconcile_Reconcile_FailureMakeJenkinsClient(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
	logger := helper.LoggerMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(nil,
		errors.New("client fatal"))

	r := Reconcile{
		client:               k8sClient,
		log:                  &logger,
		jenkinsClientFactory: &jBuilder,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatal("error is not returned")
	}

	if !strings.Contains(err.Error(), "client fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestReconcile_Reconcile_AssignRoleFailure(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).Return(errors.New("assign fatal"))
	logger := helper.LoggerMock{}

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &logger,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jarm.Namespace, Name: jarm.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	lastErr := logger.LastError()
	if lastErr == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(lastErr.Error(), "assign fatal") {
		t.Fatalf("wrong error returned: %s", lastErr.Error())
	}
}
