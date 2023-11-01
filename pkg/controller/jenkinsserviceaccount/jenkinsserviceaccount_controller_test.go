package jenkinsserviceaccount

import (
	"context"
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

func TestReconcileJenkinsServiceAccount_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsServiceAccount{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsServiceAccount{
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

func TestReconcileJenkinsServiceAccount_Reconcile_Unreg(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion)
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsServiceAccount{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no kind is registered")
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkinsServiceAccount_Reconcile_getOrCreateInstanceOwnerErr(t *testing.T) {
	ctx := context.Background()

	instance := &jenkinsApi.JenkinsServiceAccount{
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
	}

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      "",
			Namespace: namespace,
		},
	}
	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsServiceAccount{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(jenkins, instance).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsServiceAccount{
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

func TestReconcileJenkinsServiceAccount_Reconcile_InitJenkinsClientNil(t *testing.T) {
	ctx := context.Background()
	platform := pmock.PlatformService{}

	instance := &jenkinsApi.JenkinsServiceAccount{
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
	}

	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsServiceAccount{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(jenkins, instance).WithScheme(s).Build()

	platform.On("GetExternalEndpoint", namespace, name).Return("", "", "", nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsServiceAccount{
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

func TestNewReconcileJenkinsFolder(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	log := &common.Logger{}
	scheme := runtime.NewScheme()
	platform := &pmock.PlatformService{}
	Reconcile := NewReconcileJenkinsServiceAccount(cl, scheme, log, platform)
	Expected := &ReconcileJenkinsServiceAccount{
		client:   cl,
		scheme:   scheme,
		log:      log,
		platform: platform,
	}
	assert.Equal(t, Expected, Reconcile)
}
