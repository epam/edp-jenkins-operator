package jenkins

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/epam/edp-common/pkg/mock"

	mocks "github.com/epam/edp-jenkins-operator/v2/mock"
	smock "github.com/epam/edp-jenkins-operator/v2/mock/service"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
)

const (
	name      = "name"
	namespace = "namespace"
)

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func createJenkinsByStatus(status string) *jenkinsApi.Jenkins {
	return &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsStatus{
			Status: status,
		},
	}
}

func TestReconcileJenkins_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	_, isMsgFound := log.InfoMessages["instance not found"]

	assert.True(t, isMsgFound)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkins_Reconcile_UpdateEmptyStatusErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus("")

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update status from ")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_UpdateStatusInstallErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusInstall)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update status from ")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_CreateAdminPasswordErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusInstall)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create admin password secret creation: test")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_IsDeploymentReadyErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusInstall)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(nil)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(false, errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if Deployment configs are ready")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_IsDeploymentReadyFalse(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusInstall)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(false, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)

	_, ok := log.InfoMessages["Deployment configs is not ready for configuration yet"]
	assert.True(t, ok)
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_UpdateStatusCreatedErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusCreated)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update status from")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_ConfigureErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusCreated)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to finish configuration")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_ConfigureFalse(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusCreated)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)

	_, ok := log.InfoMessages["Configuration is not finished"]
	assert.True(t, ok)
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_UpdateStatusConfiguringErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusConfiguring)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update status from ")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_UpdateStatusConfiguredErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusConfigured)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update status from ")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_ExposeConfigurationErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusConfigured)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to expose configuration")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_updateInstanceStatusErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update instance status")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_IntegrationErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("Integration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, errTest)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration failed")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_IntegrationFalse(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("Integration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)

	_, ok := log.InfoMessages["Integration is not finished"]
	assert.True(t, ok)
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_UpdateStatusIntegrationStartErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, nil)
	serv.On("Integration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)

	_, ok := log.InfoMessages["Couldn't update status"]
	assert.True(t, ok)
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_updateAvailableStatusErr(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")

	sw.On("Update").Return(nil).Once()
	sw.On("Update").Return(errTest)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	mc.On("Update").Return(errTest)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, nil)
	serv.On("Integration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update availability status")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}

func TestReconcileJenkins_Reconcile_AllValid(t *testing.T) {
	ctx := context.Background()
	sw := &mocks.StatusWriter{}
	mc := mocks.Client{}
	serv := smock.JenkinsService{}

	s := runtime.NewScheme()
	instance := createJenkinsByStatus(StatusIntegrationStart)

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	sw.On("Update").Return(nil)
	mc.On("Get", nsn, &jenkinsApi.Jenkins{}).Return(cl)
	mc.On("Status").Return(sw)
	serv.On("CreateAdminPassword", mock.AnythingOfType("*v1.Jenkins")).Return(nil)
	serv.On("IsDeploymentReady", mock.AnythingOfType("*v1.Jenkins")).Return(true, nil)
	serv.On("Configure", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)
	serv.On("ExposeConfiguration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, false, nil)
	serv.On("Integration", mock.AnythingOfType("*v1.Jenkins")).Return(instance, true, nil)

	log := &common.Logger{}
	rg := ReconcileJenkins{
		client:  &mc,
		log:     log,
		service: &serv,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 60 * time.Second}, rs)
	mc.AssertExpectations(t)
	sw.AssertExpectations(t)
	serv.AssertExpectations(t)
}
