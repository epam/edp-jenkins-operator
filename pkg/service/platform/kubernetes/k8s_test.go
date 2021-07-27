package kubernetes

import (
	"testing"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestK8SService_CreateConfigMap(t *testing.T) {
	cm := v1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cm)
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&cm).Build()

	svc := K8SService{
		client: client,
		Scheme: scheme,
	}

	ji := v1alpha1.Jenkins{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	_, err := svc.CreateConfigMap(ji, "test", map[string]string{"bar": "baz"}, map[string]string{"lol": "lol"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateConfigMap(ji, "test", map[string]string{"bar": "baz"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.CreateConfigMap(ji, "test", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestK8SService_CreateJenkinsScript(t *testing.T) {
	cm := v1.ConfigMap{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cm)
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&cm).Build()

	svc := K8SService{
		client: client,
		Scheme: scheme,
	}

	if _, err := svc.CreateJenkinsScript("ns", "name", true); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.CreateJenkinsScript("ns", "name", true); err != nil {
		t.Fatal(err)
	}
}
