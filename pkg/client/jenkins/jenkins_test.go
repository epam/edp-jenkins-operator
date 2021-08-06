package jenkins

import (
	"testing"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/jarcoal/httpmock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInitGoJenkinsClient(t *testing.T) {
	httpmock.Activate()
	httpmock.RegisterResponder("GET", "http://hostpath/api/json",
		httpmock.NewStringResponder(200, ""))

	ji := v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: v1alpha1.JenkinsSpec{},
		Status: v1alpha1.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}
	ps := platform.Mock{}

	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).Return(map[string][]byte{
		"username": []byte("tester"),
		"password": []byte("pwd"),
	}, nil)
	if _, err := InitGoJenkinsClient(&ji, &ps); err != nil {
		t.Fatal(err)
	}
}

func TestInitJenkinsClient(t *testing.T) {
	ji := v1alpha1.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "name1",
		},
		Spec: v1alpha1.JenkinsSpec{},
		Status: v1alpha1.JenkinsStatus{
			AdminSecretName: "admin-secret",
		},
	}
	ps := platform.Mock{}
	ps.On("GetExternalEndpoint", ji.Namespace, ji.Name).Return("host", "http", "path", nil)
	ps.On("GetSecretData", ji.Namespace, ji.Status.AdminSecretName).Return(map[string][]byte{
		"username": []byte("tester"),
		"password": []byte("pwd"),
	}, nil)

	if _, err := InitJenkinsClient(&ji, &ps); err != nil {
		t.Fatal(err)
	}
}
