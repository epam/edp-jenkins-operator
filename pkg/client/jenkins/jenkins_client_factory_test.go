package jenkins

import (
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
)

func TestMakeClientBuilder(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	ps := pmock.PlatformService{}

	cb := MakeClientBuilder(&ps, k8sClient)

	if cb == nil {
		t.Fatal("client builder is not init")
	}

	if cb.client != k8sClient {
		t.Fatal("client builder k8s client is not set")
	}

	if cb.platform != &ps {
		t.Fatal("client builder platform service is not set")
	}
}

func TestClientBuilder_MakeNewClient_Failure_NoJenkins(t *testing.T) {
	var (
		k8sClient = fake.NewClientBuilder().Build()
		ps        pmock.PlatformService
		jar       jenkinsApi.JenkinsAuthorizationRole
		cb        = MakeClientBuilder(&ps, k8sClient)
	)

	_, err := cb.MakeNewClient(&jar.ObjectMeta, nil)
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "couldn't get Jenkins instances in namespace") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestClientBuilder_MakeNewClient(t *testing.T) {
	var (
		ps  pmock.PlatformService
		jar jenkinsApi.JenkinsAuthorizationRole
		ji  = jenkinsApi.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "name1",
			},
			Spec: jenkinsApi.JenkinsSpec{},
			Status: jenkinsApi.JenkinsStatus{
				AdminSecretName: "admin-secret",
			},
		}
	)

	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&ji, &jar).Build()
	cb := MakeClientBuilder(&ps, k8sClient)

	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).Return(map[string][]byte{
		"username": []byte("tester"),
		"password": []byte("pwd"),
	}, nil)

	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://hostpath/api/json",
		httpmock.NewStringResponder(200, ""))

	if _, err := cb.MakeNewClient(&jar.ObjectMeta, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientBuilder_MakeNewClient_FailureGetExternalEndpoint(t *testing.T) {
	var (
		ps  pmock.PlatformService
		jar jenkinsApi.JenkinsAuthorizationRole
		ji  = jenkinsApi.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "name1",
			},
			Spec: jenkinsApi.JenkinsSpec{},
			Status: jenkinsApi.JenkinsStatus{
				AdminSecretName: "admin-secret",
			},
		}
	)

	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&ji, &jar).Build()
	cb := MakeClientBuilder(&ps, k8sClient)

	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).
		Return("host", "http", "path", errors.New("external fatal"))

	_, err := cb.MakeNewClient(&jar.ObjectMeta, nil)
	if err == nil {
		t.Fatal("wrong error returned")
	}

	if !strings.Contains(errors.Cause(err).Error(), "external fatal") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
