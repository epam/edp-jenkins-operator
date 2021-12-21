package jenkins

import (
	"context"
	common "github.com/epam/edp-common/pkg/mock"
	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"testing"
)

const name = "name"
const namespace = "namespace"
const URLScheme = "https://"

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
	platform := pmock.PlatformService{}
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

	err := os.Setenv(helper.PlatformType, "kubernetes")
	assert.NoError(t, err)
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "https:////firstthird/api/json", httpmock.NewStringResponder(200, ""))

	s.AddKnownTypes(v1.SchemeGroupVersion, &jenkinsApi.JenkinsFolder{}, &jenkinsApi.JenkinsList{}, &jenkinsApi.Jenkins{})
	cl := fake.NewClientBuilder().WithObjects(instance, jenkins).WithScheme(s).Build()

	platform.On("GetExternalEndpoint", namespace, name).Return(first, URLScheme, third, nil)
	platform.On("GetSecretData", namespace, name).Return(secretData, nil)

	log := &common.Logger{}
	rg := ReconcileJenkinsFolder{
		client:   cl,
		log:      log,
		platform: &platform,
	}
	req := reconcile.Request{
		NamespacedName: nsn,
	}
	rs, err := rg.Reconcile(ctx, req)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "an error has been occurred while setting owner reference"))
	assert.Equal(t, reconcile.Result{}, rs)
}
