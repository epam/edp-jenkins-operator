package jenkinsscript

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/epam/edp-common/pkg/mock"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
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

func createJenkinsScript() *jenkinsApi.JenkinsScript {
	return &jenkinsApi.JenkinsScript{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion:         "",
					Kind:               "Jenkins",
					Name:               name,
					UID:                "",
					Controller:         nil,
					BlockOwnerDeletion: nil,
				},
			},
			Name:      name,
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsScriptStatus{
			Executed: true,
		},
	}
}

func TestReconcileJenkinsScript_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
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

func TestReconcileJenkinsScript_Reconcile_getOrCreateInstanceOwnerErr(t *testing.T) {
	ctx := context.Background()

	instance := createJenkinsScript()

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      "",
			Namespace: namespace,
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get owner for")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
}

func TestReconcileJenkinsScript_Reconcile_StatusExecuted(t *testing.T) {
	ctx := context.Background()

	instance := createJenkinsScript()

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkinsScript_Reconcile_InitJenkinsClientErr(t *testing.T) {
	ctx := context.Background()
	platform := pmock.PlatformService{}

	instance := createJenkinsScript()
	instance.Status.Executed = false

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	errTest := errors.New("test")
	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	platform.On("GetExternalEndpoint", namespace, name).Return("", "", "", errTest)

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
		client:   cl,
		log:      log,
		platform: &platform,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to init jenkins client for")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	platform.AssertExpectations(t)
}

func TestReconcileJenkinsScript_Reconcile_InitJenkinsClientNil(t *testing.T) {
	ctx := context.Background()
	platform := pmock.PlatformService{}

	instance := createJenkinsScript()
	instance.Status.Executed = false

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: "",
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	platform.On("GetExternalEndpoint", namespace, name).Return("", "", "", nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
		client:   cl,
		log:      log,
		platform: &platform,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	platform.AssertExpectations(t)
}

func TestReconcileJenkinsScript_Reconcile_GetConfigMapDataErr(t *testing.T) {
	ctx := context.Background()
	platform := pmock.PlatformService{}

	instance := createJenkinsScript()
	instance.Status.Executed = false
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: jenkinsApi.JenkinsStatus{
			AdminSecretName: name,
		},
	}
	errTest := errors.New("test")

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsScript{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	platform.On("GetExternalEndpoint", namespace, name).Return("", "", "", nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)
	platform.On("GetConfigMapData", namespace, "").Return(nil, errTest)

	log := &common.Logger{}
	rg := ReconcileJenkinsScript{
		client:   cl,
		log:      log,
		platform: &platform,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get config map for")
	assert.Equal(t, reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, rs)
	platform.AssertExpectations(t)
}

func TestNewReconcileJenkinsScript(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	log := &common.Logger{}
	scheme := runtime.NewScheme()
	platform := &pmock.PlatformService{}
	Reconcile := NewReconcileJenkinsScript(cl, scheme, log, platform)
	Expected := &ReconcileJenkinsScript{
		client:   cl,
		scheme:   scheme,
		log:      log,
		platform: platform,
	}
	assert.Equal(t, Expected, Reconcile)
}
