package cdstagejenkinsdeployment

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	common "github.com/epam/edp-common/pkg/mock"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

const name = "name"
const namespace = "namespace"

var nsn = types.NamespacedName{
	Namespace: namespace,
	Name:      name,
}

func TestReconcileCDStageJenkinsDeployment_Reconcile(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.CDStageJenkinsDeployment{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileCDStageJenkinsDeployment{
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

func TestReconcileCDStageJenkinsDeployment_setOwnerReferenceErr(t *testing.T) {
	ctx := context.Background()
	instance := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.CDStageJenkinsDeployment{}, &codebaseApi.CDStageDeploy{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileCDStageJenkinsDeployment{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "cannot set owner ref for"))
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileCDStageJenkinsDeployment_GetPlatformTypeEnvErr(t *testing.T) {
	ctx := context.Background()
	CDStageDeploy := &codebaseApi.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance.Labels = map[string]string{
		consts.CdStageDeployKey: name,
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.CDStageJenkinsDeployment{}, &codebaseApi.CDStageDeploy{})
	cl := fake.NewClientBuilder().WithObjects(instance, CDStageDeploy).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileCDStageJenkinsDeployment{
		client: cl,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Environment variable PLATFORM_TYPE no found"))
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileCDStageJenkinsDeployment_NewPlatformServiceErr(t *testing.T) {
	ctx := context.Background()
	CDStageDeploy := &codebaseApi.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance.Labels = map[string]string{
		consts.CdStageDeployKey: name,
	}
	err := os.Setenv(helper.PlatformType, "test")
	assert.NoError(t, err)

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.CDStageJenkinsDeployment{}, &codebaseApi.CDStageDeploy{})
	cl := fake.NewClientBuilder().WithObjects(instance, CDStageDeploy).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileCDStageJenkinsDeployment{
		client: cl,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "couldn't create platform service"))
	assert.Equal(t, reconcile.Result{}, rs)
	err = os.Unsetenv(helper.PlatformType)
	assert.NoError(t, err)
}

func TestReconcileCDStageJenkinsDeployment_CreateDefChainErr(t *testing.T) {
	ctx := context.Background()
	CDStageDeploy := &codebaseApi.CDStageDeploy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance := &jenkinsApi.CDStageJenkinsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	jenkins := &jenkinsApi.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	instance.Labels = map[string]string{
		consts.CdStageDeployKey: name,
		chain.JenkinsKey:        name,
	}
	err := os.Setenv(helper.PlatformType, platform.K8SPlatformType)
	assert.NoError(t, err)

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.CDStageJenkinsDeployment{}, &codebaseApi.CDStageDeploy{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, CDStageDeploy, jenkins).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileCDStageJenkinsDeployment{
		client: cl,
		log:    log,
		scheme: s,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.NoError(t, err)

	result := &jenkinsApi.CDStageJenkinsDeployment{}
	err = cl.Get(ctx, nsn, result)
	assert.NoError(t, err)
	assert.Equal(t, "failed", result.Status.Status)
	assert.Equal(t, reconcile.Result{RequeueAfter: 500 * time.Millisecond}, rs)
	err = os.Unsetenv(helper.PlatformType)
	assert.NoError(t, err)
}

func TestNewReconcileCDStageJenkinsDeployment(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	log := &common.Logger{}
	scheme := runtime.NewScheme()
	Reconcile := NewReconcileCDStageJenkinsDeployment(cl, scheme, log)
	Expected := &ReconcileCDStageJenkinsDeployment{
		client: cl,
		scheme: scheme,
		log:    log,
	}
	assert.Equal(t, Expected, Reconcile)
}
