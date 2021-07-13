package jenkins_jobbuildrun

import (
	"context"
	"testing"
	"time"

	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"

	"github.com/bndr/gojenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile_ReconcileJobNotFound(t *testing.T) {
	jbr := v1alpha1.JenkinsJobBuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsJobBuildRun",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsJobBuildRunSpec{
			JobPath: "path/job",
			Retry:   2,
		},
		Status: v1alpha1.JenkinsJobBuildRunStatus{
			BuildNumber: 5,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jbr.ObjectMeta, jbr.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("GetJobByName", "path/job").Return(nil, errors.New("404"))

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jbr.Namespace, Name: jbr.Name},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter != 0 {
		t.Fatal("RequeueAfter is set")
	}

	var checkJenkinsJobBuildRun v1alpha1.JenkinsJobBuildRun
	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err != nil {
		t.Fatal(err)
	}

	if checkJenkinsJobBuildRun.Status.Status != v1alpha1.JobBuildRunStatusNotFound {
		t.Fatalf("wrong job status: %s", checkJenkinsJobBuildRun.Status.Status)
	}
}

func TestReconcile_ReconcileNewBuild(t *testing.T) {
	jbr := v1alpha1.JenkinsJobBuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsJobBuildRun",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsJobBuildRunSpec{
			JobPath: "path/job",
			Retry:   2,
		},
		Status: v1alpha1.JenkinsJobBuildRunStatus{
			BuildNumber: 5,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jbr.ObjectMeta, jbr.Spec.OwnerName).Return(&jClient, nil)

	job := gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			InQueue: false,
		},
	}
	jClient.On("GetJobByName", "path/job").Return(&job, nil)

	jobBuild := gojenkins.Build{
		Raw: &gojenkins.BuildResponse{
			Number: 1,
		},
	}
	jClient.On("GetLastBuild", &job).Return(&jobBuild, nil)
	jClient.On("BuildIsRunning", &jobBuild).Return(false)

	var buildNum int64 = 5
	jClient.On("BuildJob", jbr.Spec.JobPath, jbr.Spec.Params).Return(&buildNum, nil)

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jbr.Namespace, Name: jbr.Name},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter == 0 {
		t.Fatal("RequeueAfter is not set")
	}

	var checkJenkinsJobBuildRun v1alpha1.JenkinsJobBuildRun
	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err != nil {
		t.Fatal(err)
	}

	if checkJenkinsJobBuildRun.Status.Status != v1alpha1.JobBuildRunStatusCreated {
		t.Fatalf("wrong job status: %s", checkJenkinsJobBuildRun.Status.Status)
	}
}

func TestReconcile_ReconcileOldBuild(t *testing.T) {
	jbr := v1alpha1.JenkinsJobBuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsJobBuildRun",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsJobBuildRunSpec{
			JobPath: "path/job",
			Retry:   1,
		},
		Status: v1alpha1.JenkinsJobBuildRunStatus{
			BuildNumber: 5,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jbr.ObjectMeta, jbr.Spec.OwnerName).Return(&jClient, nil)

	job := gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			InQueue: false,
		},
	}
	jClient.On("GetJobByName", "path/job").Return(&job, nil)

	jobBuild := gojenkins.Build{
		Raw: &gojenkins.BuildResponse{
			Number: 5,
		},
	}
	jClient.On("GetLastBuild", &job).Return(&jobBuild, nil)
	jClient.On("BuildIsRunning", &jobBuild).Return(false)

	var buildNum int64 = 5
	jClient.On("BuildJob", jbr.Spec.JobPath, jbr.Spec.Params).Return(&buildNum, nil)

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jbr.Namespace, Name: jbr.Name},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter == 0 {
		t.Fatal("RequeueAfter is not set")
	}

	var checkJenkinsJobBuildRun v1alpha1.JenkinsJobBuildRun
	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err != nil {
		t.Fatal(err)
	}

	if checkJenkinsJobBuildRun.Status.Status != v1alpha1.JobBuildRunStatusRetrying {
		t.Fatalf("wrong job status: %s", checkJenkinsJobBuildRun.Status.Status)
	}

	jBuilder.On("MakeNewClient", &checkJenkinsJobBuildRun.ObjectMeta, checkJenkinsJobBuildRun.Spec.OwnerName).Return(&jClient, nil)

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err != nil {
		t.Fatal(err)
	}

	if checkJenkinsJobBuildRun.Status.Status != v1alpha1.JobBuildRunStatusFailed {
		t.Fatalf("wrong job status: %s", checkJenkinsJobBuildRun.Status.Status)
	}
}

func TestReconcile_ReconcileDeleteExpiredBuilds(t *testing.T) {
	deleteJobInterval := "1s"

	jbr := v1alpha1.JenkinsJobBuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsJobBuildRun",
			APIVersion: "apps/v1",
		},
		Spec: v1alpha1.JenkinsJobBuildRunSpec{
			JobPath:                       "path/job",
			Retry:                         1,
			DeleteAfterCompletionInterval: &deleteJobInterval,
		},
		Status: v1alpha1.JenkinsJobBuildRunStatus{
			BuildNumber: 5,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, &jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(&jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", &jbr.ObjectMeta, jbr.Spec.OwnerName).Return(&jClient, nil)

	job := gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			InQueue: false,
		},
	}
	jClient.On("GetJobByName", "path/job").Return(&job, nil)

	jobBuild := gojenkins.Build{
		Raw: &gojenkins.BuildResponse{
			Number: 5,
			Result: gojenkins.STATUS_SUCCESS,
		},
	}
	jClient.On("GetLastBuild", &job).Return(&jobBuild, nil)
	jClient.On("BuildIsRunning", &jobBuild).Return(false)

	var buildNum int64 = 5
	jClient.On("BuildJob", jbr.Spec.JobPath, jbr.Spec.Params).Return(&buildNum, nil)

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jbr.Namespace, Name: jbr.Name},
	}

	res, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequeueAfter == 0 {
		t.Fatal("RequeueAfter is not set")
	}

	var checkJenkinsJobBuildRun v1alpha1.JenkinsJobBuildRun
	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err != nil {
		t.Fatal(err)
	}

	if checkJenkinsJobBuildRun.Status.Status != v1alpha1.JobBuildRunStatusCompleted {
		t.Fatalf("wrong job status: %s", checkJenkinsJobBuildRun.Status.Status)
	}

	time.Sleep(time.Second)

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if err := k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun); err == nil {
		t.Fatal("build is not deleted")
	}
}
