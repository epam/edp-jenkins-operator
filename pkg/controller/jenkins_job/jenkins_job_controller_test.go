package jenkins

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	common "github.com/epam/edp-common/pkg/mock"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mocks "github.com/epam/edp-jenkins-operator/v2/mock"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const name = "name"
const namespace = "namespace"
const URLScheme = "https"

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func createJenkinsJobInstance() *jenkinsApi.JenkinsJob {
	str := name
	return &jenkinsApi.JenkinsJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: jenkinsApi.JenkinsJobSpec{
			StageName: &str,
		},
	}
}

func TestReconcileJenkinsJob_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
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

func TestReconcileJenkinsJob_Reconcile_tryToDeleteJobErr(t *testing.T) {
	ctx := context.Background()
	mc := mocks.Client{}

	instance := createJenkinsJobInstance()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	errTest := errors.New("test")

	mc.On("Get", nsn, &jenkinsApi.JenkinsJob{}).Return(cl)
	mc.On("Update").Return(errTest)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client: &mc,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Equal(t, errTest, err)
	assert.Equal(t, reconcile.Result{}, rs)
	mc.AssertExpectations(t)
}

func TestReconcileJenkinsJob_Reconcile_setOwnersErr(t *testing.T) {
	ctx := context.Background()

	instance := createJenkinsJobInstance()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while setting owner reference"))
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkinsJob_Reconcile_canJenkinsJobBeHandledErr(t *testing.T) {
	ctx := context.Background()
	mc := mocks.Client{}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	nsn2 := types.NamespacedName{
		Namespace: namespace,
		Name:      name + "-cd-pipeline",
	}

	instance := createJenkinsJobInstance()
	str := name
	instance.Spec.JenkinsFolder = &str

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage).WithScheme(s).Build()

	errTest := errors.New("test")

	mc.On("Get", nsn, &jenkinsApi.JenkinsJob{}).Return(cl)
	mc.On("Get", nsn, &cdPipeApi.Stage{}).Return(cl)
	mc.On("Get", nsn2, &jenkinsApi.JenkinsFolder{}).Return(errTest)
	mc.On("Update").Return(cl)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client: &mc,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while checking availability of creating jenkins job"))
	assert.Equal(t, reconcile.Result{}, rs)
	mc.AssertExpectations(t)
}

func TestReconcileJenkinsJob_Reconcile_canJenkinsJobBeHandledFalse(t *testing.T) {
	ctx := context.Background()

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	folder := &jenkinsApi.JenkinsFolder{
		ObjectMeta: v1.ObjectMeta{
			Name:      name + "-cd-pipeline",
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsFolderStatus{
			Available: false,
		},
	}

	instance := createJenkinsJobInstance()
	str := name
	instance.Spec.JenkinsFolder = &str

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{}, &jenkinsApi.JenkinsFolder{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage, folder).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client: cl,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: 10 * time.Second}, rs)
}

func TestReconcileJenkinsJob_Reconcile_GetJenkinsInstanceOwnerErr(t *testing.T) {
	ctx := context.Background()

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	instance := createJenkinsJobInstance()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client: cl,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while getting owner jenkins for jenkins job"))
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkinsJob_Reconcile_InitGoJenkinsClientErr(t *testing.T) {
	ctx := context.Background()
	platformMock := pmock.PlatformService{}

	jen := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	instance := createJenkinsJobInstance()

	errTest := errors.New("test")
	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{}, &jenkinsApi.JenkinsList{},
		&jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage, jen).WithScheme(s).Build()
	platformMock.On("GetExternalEndpoint", namespace, name).Return("", "", "", errTest)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client:   cl,
		log:      log,
		scheme:   s,
		platform: &platformMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has occurred while init jenkins client"))
	assert.Equal(t, reconcile.Result{}, rs)
	platformMock.AssertExpectations(t)
}

func TestReconcileJenkinsJob_Reconcile_isJenkinsJobExistErr(t *testing.T) {
	ctx := context.Background()
	platformMock := pmock.PlatformService{}

	jen := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
	}

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	instance := createJenkinsJobInstance()

	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://12/api/json", httpmock.NewStringResponder(200, ""))

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{}, &jenkinsApi.JenkinsList{},
		&jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage, jen).WithScheme(s).Build()
	platformMock.On("GetExternalEndpoint", namespace, name).Return("1", URLScheme, "2", nil)
	platformMock.On("GetSecretData", namespace, "").Return(secretData, nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client:   cl,
		log:      log,
		scheme:   s,
		platform: &platformMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has occurred while retrieving jenkins job"))
	assert.Equal(t, reconcile.Result{}, rs)
	platformMock.AssertExpectations(t)
}

func TestReconcileJenkinsJob_Reconcile_getChainErr(t *testing.T) {
	ctx := context.Background()
	platformMock := pmock.PlatformService{}

	jen := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
	}

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	instance := createJenkinsJobInstance()

	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://12/api/json", httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("GET", "https://12/job/api/json", httpmock.NewStringResponder(200, ""))

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{}, &jenkinsApi.JenkinsList{},
		&jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage, jen).WithScheme(s).Build()
	platformMock.On("GetExternalEndpoint", namespace, name).Return("1", URLScheme, "2", nil)
	platformMock.On("GetSecretData", namespace, "").Return(secretData, nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client:   cl,
		log:      log,
		scheme:   s,
		platform: &platformMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has occurred while selecting chain"))
	assert.Equal(t, reconcile.Result{}, rs)
	platformMock.AssertExpectations(t)
}

func TestReconcileJenkinsJob_Reconcile_ServeRequestErr(t *testing.T) {
	ctx := context.Background()
	platformMock := pmock.PlatformService{}

	jen := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
	}

	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	stage := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	instance := createJenkinsJobInstance()

	err := os.Setenv(helper.PlatformType, platform.K8SPlatformType)
	assert.NoError(t, err)
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://12/api/json", httpmock.NewStringResponder(200, ""))
	httpmock.RegisterResponder("GET", "https://12/job/api/json", httpmock.NewStringResponder(200, ""))

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsJob{}, &cdPipeApi.Stage{}, &jenkinsApi.JenkinsList{},
		&jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, stage, jen).WithScheme(s).Build()
	platformMock.On("GetExternalEndpoint", namespace, name).Return("1", URLScheme, "2", nil)
	platformMock.On("GetSecretData", namespace, "").Return(secretData, nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsJob{
		client:   cl,
		log:      log,
		scheme:   s,
		platform: &platformMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client"))
	assert.Equal(t, reconcile.Result{}, rs)
	platformMock.AssertExpectations(t)
}

func TestNewReconcileJenkinsJob(t *testing.T) {

	cl := fake.NewClientBuilder().Build()
	log := &common.Logger{}
	scheme := runtime.NewScheme()
	platformMock := &pmock.PlatformService{}
	Reconcile := NewReconcileJenkinsJob(cl, scheme, log, platformMock)
	Expected := &ReconcileJenkinsJob{
		client:   cl,
		scheme:   scheme,
		log:      log,
		platform: platformMock,
	}
	assert.Equal(t, Expected, Reconcile)

}
