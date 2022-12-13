package jenkins_authorizationrolemapping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
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
	require.NoError(t, err)

	var checkInstance jenkinsApi.JenkinsAuthorizationRoleMapping

	require.NoError(t,
		k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: jarm.Namespace,
				Name:      jarm.Name,
			},
			&checkInstance,
		),
	)

	require.Emptyf(t, checkInstance.GetFinalizers(), "finalizers still set")
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
		NamespacedName: types.NamespacedName{
			Namespace: jarm.Namespace,
			Name:      jarm.Name,
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	var checkInstance jenkinsApi.JenkinsAuthorizationRoleMapping

	require.NoError(t,
		k8sClient.Get(
			context.Background(),
			types.NamespacedName{
				Namespace: jarm.Namespace,
				Name:      jarm.Name,
			},
			&checkInstance,
		),
	)

	require.Containsf(t, checkInstance.Status.Value, "unset role fatal", "no unset error in instance status")
}

func TestSpecUpdated(t *testing.T) {
	jarm1 := getTestJenkinsAuthorizationRoleMapping()
	jarm2 := getTestJenkinsAuthorizationRoleMapping()
	jarm2.Spec.RoleType = "rt123"

	require.True(t, specUpdated(event.UpdateEvent{ObjectOld: jarm1, ObjectNew: jarm2}))
}

func TestNewReconciler(t *testing.T) {
	lg := helper.LoggerMock{}
	k8sClient := fake.NewClientBuilder().Build()
	ps := pmock.PlatformService{}

	rec := NewReconciler(k8sClient, &lg, &ps)
	require.NotNilf(t, rec, "reconciler is not inited")

	require.Falsef(t, rec.log != &lg || rec.client != k8sClient, "wrong reconciler params")
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
		NamespacedName: types.NamespacedName{
			Namespace: jarm.Namespace,
			Name:      "baz",
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.Equalf(t, "instance not found", logger.LastInfo(), "not found error is not logged")
}

func TestReconcile_Reconcile_FailureMakeJenkinsClient(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
	logger := helper.LoggerMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).
		Return(nil, errors.New("client fatal"))

	r := Reconcile{
		client:               k8sClient,
		log:                  &logger,
		jenkinsClientFactory: &jBuilder,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: jarm.Namespace,
			Name:      jarm.Name,
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.Error(t, err)

	require.Contains(t, err.Error(), "client fatal")
}

func TestReconcile_Reconcile_AssignRoleFailure(t *testing.T) {
	jarm := getTestJenkinsAuthorizationRoleMapping()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jarm)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jarm).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jarm.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("AssignRole", jarm.Spec.RoleType, jarm.Spec.Roles[0], jarm.Spec.Group).
		Return(errors.New("assign fatal"))

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
	require.NoError(t, err)

	lastErr := logger.LastError()
	require.Error(t, lastErr)

	require.Contains(t, lastErr.Error(), "assign fatal")
}
