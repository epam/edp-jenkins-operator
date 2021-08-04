package helper

import (
	"strings"
	"testing"
	"time"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestTryToDelete_AddFinalizers(t *testing.T) {
	ja := v1alpha1.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ja1",
			Namespace: "ns1",
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	if _, err := TryToDelete(&ja, "fint", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(ja.GetFinalizers()) == 0 {
		t.Fatal("no finalizers added")
	}
}

func TestTryToDelete_RemoveFinalizers(t *testing.T) {
	ja := v1alpha1.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ja1",
			Namespace:         "ns1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"fint"},
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	if _, err := TryToDelete(&ja, "fint", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(ja.GetFinalizers()) > 0 {
		t.Fatal("finalizers is not removed")
	}
}

func TestTryToDelete_DeleteFuncFailure(t *testing.T) {
	ja := v1alpha1.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ja1",
			Namespace:         "ns1",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &ja)

	_, err := TryToDelete(&ja, "fint", func() error {
		return errors.New("del func fatal")
	})
	if err == nil {
		t.Fatal("no error func returned")
	}

	if !strings.Contains(err.Error(), "del func fatal") {
		t.Log(err)
		t.Fatal("wrong func returned")
	}
}
