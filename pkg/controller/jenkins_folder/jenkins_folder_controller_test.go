package jenkins

import (
	"context"
	"os"
	"strings"
	"testing"

	common "github.com/epam/edp-common/pkg/mock"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

func TestReconcileJenkinsFolder_Reconcile_EmptyClient(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{})
	cl := fake.NewClientBuilder().WithObjects().WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsFolder{
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

func TestReconcileJenkinsFolder_Reconcile_tryToDeleteJenkinsFolderErr(t *testing.T) {
	ctx := context.Background()

	s := runtime.NewScheme()
	instance := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{}, &jenkinsApi.JenkinsList{})
	cl := fake.NewClientBuilder().WithObjects(instance).WithScheme(s).Build()

	log := &common.Logger{}
	rg := ReconcileJenkinsFolder{
		client: cl,
		log:    log,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)

	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while creating gojenkins client"))
	assert.Equal(t, reconcile.Result{}, rs)
}

func TestReconcileJenkinsFolder_Reconcile_ServeRequestErr(t *testing.T) {
	ctx := context.Background()
	platformMock := pmock.PlatformService{}
	first := "first"
	third := "third"
	secretData := map[string][]byte{
		"username": {'a'},
		"password": {'k'},
	}

	s := runtime.NewScheme()
	instance := &jenkinsApi.JenkinsFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	jenkins := &jenkinsApi.Jenkins{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
		Status: jenkinsApi.JenkinsStatus{AdminSecretName: name},
	}

	err := os.Setenv(helper.PlatformType, platform.K8SPlatformType)
	assert.NoError(t, err)
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https://firstthird/api/json", httpmock.NewStringResponder(200, ""))

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{}, &jenkinsApi.JenkinsList{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	platformMock.On("GetExternalEndpoint", namespace, name).Return(first, URLScheme, third, nil)
	platformMock.On("GetSecretData", namespace, name).Return(secretData, nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsFolder{
		client:   cl,
		log:      log,
		platform: &platformMock,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while setting owner reference"))
	assert.Equal(t, reconcile.Result{}, rs)
	platformMock.AssertExpectations(t)
}

func TestNewReconcileJenkinsFolder(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	log := &common.Logger{}
	scheme := runtime.NewScheme()
	platformMock := &pmock.PlatformService{}
	Reconcile := NewReconcileJenkinsFolder(cl, scheme, log, platformMock)
	Expected := &ReconcileJenkinsFolder{
		client:   cl,
		scheme:   scheme,
		log:      log,
		platform: platformMock,
	}
	assert.Equal(t, Expected, Reconcile)
}
