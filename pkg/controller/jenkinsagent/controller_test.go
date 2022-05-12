package jenkinsagent

import (
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
)

func TestSpecUpdate(t *testing.T) {
	agent1 := jenkinsApi.JenkinsAgent{
		Spec: jenkinsApi.JenkinsAgentSpec{
			Name: "1",
		},
	}

	agent2 := jenkinsApi.JenkinsAgent{
		Spec: jenkinsApi.JenkinsAgentSpec{
			Name: "2",
		},
	}

	if !specUpdated(event.UpdateEvent{
		ObjectOld: &agent1,
		ObjectNew: &agent2,
	}) {
		t.Fatal("spec must be updated")
	}
}

func TestReconcile_Reconcile(t *testing.T) {
	agent := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent1",
			Namespace: "ns",
		},
		Spec: jenkinsApi.JenkinsAgentSpec{
			Name:     "agent1",
			Template: "agent-cm",
		},
	}

	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	slavesCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jenkins.SlavesTemplateName,
			Namespace: agent.Namespace,
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}

	agentCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-cm",
			Namespace: agent.Namespace,
		},
		Data: map[string]string{
			"template": "test321",
		},
	}

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&agent, &slavesCM, &agentCM).Build()

	r := Reconcile{
		client: k8sClient,
		log:    &helper.LoggerMock{},
	}

	nn := types.NamespacedName{Namespace: agent.Namespace, Name: agent.Name}

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})
	if err != nil {
		t.Fatal(err)
	}

	var checkAgent jenkinsApi.JenkinsAgent
	if err := k8sClient.Get(context.Background(), nn, &checkAgent); err != nil {
		t.Fatal(err)
	}

	if checkAgent.Status.Value != helper.StatusSuccess {
		t.Log(checkAgent.Status.Value)
		t.Fatal("wrong instance status")
	}

	var checkSlavesCM corev1.ConfigMap
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: slavesCM.Name, Namespace: slavesCM.Namespace}, &checkSlavesCM); err != nil {
		t.Fatal(err)
	}

	tpl, ok := checkSlavesCM.Data[checkAgent.Spec.SalvesKey()]
	if !ok {
		t.Fatal("slaves CM is not updated")
	}

	if tpl != agent.Spec.Template {
		t.Fatal("wrong value of agent template in slaves cm")
	}
}

func TestReconcile_Reconcile_Delete(t *testing.T) {
	agent := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "agent1",
			Namespace:         "ns",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: jenkinsApi.JenkinsAgentSpec{
			Name:     "agent1",
			Template: "foo-bar",
		},
	}

	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	slavesCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jenkins.SlavesTemplateName,
			Namespace: agent.Namespace,
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}

	agentCM := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-cm",
			Namespace: agent.Namespace,
		},
		Data: map[string]string{
			"template": "test321",
		},
	}

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&agent, &slavesCM, &agentCM).Build()

	r := Reconcile{
		client: k8sClient,
		log:    &helper.LoggerMock{},
	}

	nn := types.NamespacedName{Namespace: agent.Namespace, Name: agent.Name}

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})
	if err != nil {
		t.Fatal(err)
	}

	var checkAgent jenkinsApi.JenkinsAgent
	if err := k8sClient.Get(context.Background(), nn, &checkAgent); err != nil {
		t.Fatal(err)
	}

	if checkAgent.Status.Value != helper.StatusSuccess {
		t.Log(checkAgent.Status.Value)
		t.Fatal("wrong instance status")
	}

	var checkSlavesCM corev1.ConfigMap
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: slavesCM.Name, Namespace: slavesCM.Namespace}, &checkSlavesCM); err != nil {
		t.Fatal(err)
	}

	_, ok := checkSlavesCM.Data[checkAgent.Spec.SalvesKey()]
	if ok {
		t.Fatal("slaves CM must not contain agent template")
	}
}

func TestReconcile_Reconcile_FailureNotFound(t *testing.T) {
	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().Build()
	mockLogger := helper.LoggerMock{}

	r := Reconcile{
		client: k8sClient,
		log:    &mockLogger,
	}

	nn := types.NamespacedName{Namespace: "foo", Name: "bar"}

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})
	if err != nil {
		t.Fatal(err)
	}

	if mockLogger.LastInfo() != "JenkinsAgent is not found" {
		t.Fatal("not found error is not logged")
	}
}

func TestReconcile_Reconcile_FailureSlavesNoConfigMap(t *testing.T) {
	agent := jenkinsApi.JenkinsAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "agent1",
			Namespace:         "ns",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
		Spec: jenkinsApi.JenkinsAgentSpec{
			Name:     "agent1",
			Template: "agent-cm",
		},
	}

	s := scheme.Scheme
	utilruntime.Must(jenkinsApi.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&agent).Build()

	r := NewReconciler(k8sClient, &helper.LoggerMock{})

	nn := types.NamespacedName{Namespace: agent.Namespace, Name: agent.Name}

	_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: nn})
	if err != nil {
		t.Fatal(err)
	}

	var checkAgent jenkinsApi.JenkinsAgent
	if err := k8sClient.Get(context.Background(), nn, &checkAgent); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(checkAgent.Status.Value, "configmaps \"jenkins-slaves\" not found") {
		t.Log(checkAgent.Status.Value)
		t.Fatal("no error in instance status")
	}
}
