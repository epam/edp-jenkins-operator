package jenkins_jobbuildrun

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pmock "github.com/epam/edp-jenkins-operator/v2/mock/platform"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
)

func getTestJenkinsJobBuildRun() *jenkinsApi.JenkinsJobBuildRun {
	return &jenkinsApi.JenkinsJobBuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "ns",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "JenkinsJobBuildRun",
			APIVersion: "apps/v1",
		},
		Spec: jenkinsApi.JenkinsJobBuildRunSpec{
			JobPath: "path/job",
			Retry:   2,
		},
		Status: jenkinsApi.JenkinsJobBuildRunStatus{
			BuildNumber: 5,
		},
	}
}

func TestReconcile_ReconcileJobNotFound(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).Return(&jClient, nil)
	jClient.On("GetJobByName", "path/job").
		Return(nil, errors.New("404"))

	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &helper.LoggerMock{},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: jbr.Namespace,
			Name:      jbr.Name,
		},
	}

	res, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	if res.RequeueAfter != 0 {
		t.Fatal("RequeueAfter is set")
	}

	var checkJenkinsJobBuildRun jenkinsApi.JenkinsJobBuildRun

	require.NoError(t, k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun))

	require.Equalf(
		t,
		jenkinsApi.JobBuildRunStatusNotFound,
		checkJenkinsJobBuildRun.Status.Status,
		"wrong job status: %s",
		checkJenkinsJobBuildRun.Status.Status,
	)
}

func TestReconcile_ReconcileNewBuild(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).Return(&jClient, nil)

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
	require.NoError(t, err)

	require.NotEmptyf(t, res.RequeueAfter, "RequeueAfter is not set")

	var checkJenkinsJobBuildRun jenkinsApi.JenkinsJobBuildRun

	require.NoError(t, k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun))

	require.Equalf(
		t,
		jenkinsApi.JobBuildRunStatusCreated,
		checkJenkinsJobBuildRun.Status.Status,
		"wrong job status: %s",
		checkJenkinsJobBuildRun.Status.Status,
	)
}

func TestReconcile_ReconcileOldBuild(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()
	jbr.Spec.Retry = 1

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).Return(&jClient, nil)

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
	require.NoError(t, err)

	require.NotEmptyf(t, res.RequeueAfter, "RequeueAfter is not set")

	var checkJenkinsJobBuildRun jenkinsApi.JenkinsJobBuildRun

	require.NoError(t, k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun))

	require.Equalf(t,
		jenkinsApi.JobBuildRunStatusRetrying,
		checkJenkinsJobBuildRun.Status.Status,
		"wrong job status: %s",
		checkJenkinsJobBuildRun.Status.Status,
	)

	jBuilder.On("MakeNewClient", &checkJenkinsJobBuildRun.ObjectMeta, checkJenkinsJobBuildRun.Spec.OwnerName).
		Return(&jClient, nil)

	_, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.NoError(t, k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun))

	require.Equalf(t,
		jenkinsApi.JobBuildRunStatusFailed,
		checkJenkinsJobBuildRun.Status.Status,
		"wrong job status: %s",
		checkJenkinsJobBuildRun.Status.Status,
	)
}

func TestReconcile_ReconcileDeleteExpiredBuilds(t *testing.T) {
	deleteJobInterval := "1s"

	jbr := getTestJenkinsJobBuildRun()
	jbr.Spec.Retry = 1
	jbr.Spec.DeleteAfterCompletionInterval = &deleteJobInterval

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).Return(&jClient, nil)

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
	require.NoError(t, err)

	require.NotEmptyf(t, res.RequeueAfter, "RequeueAfter is not set")

	var checkJenkinsJobBuildRun jenkinsApi.JenkinsJobBuildRun

	require.NoError(t, k8sClient.Get(context.Background(), req.NamespacedName, &checkJenkinsJobBuildRun))

	require.Equalf(t,
		jenkinsApi.JobBuildRunStatusCompleted,
		checkJenkinsJobBuildRun.Status.Status,
		"wrong job status: %s",
		checkJenkinsJobBuildRun.Status.Status,
	)

	time.Sleep(time.Second)

	_, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.Error(t, k8sClient.Get(
		context.Background(),
		req.NamespacedName,
		&checkJenkinsJobBuildRun,
	), "build is not deleted")
}

func TestSpecUpdate(t *testing.T) {
	jbr1 := getTestJenkinsJobBuildRun()
	jbr2 := getTestJenkinsJobBuildRun()
	jbr2.Spec.Retry = 3

	require.True(t, specUpdated(event.UpdateEvent{ObjectNew: jbr1, ObjectOld: jbr2}))
}

func TestNewReconciler(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	ps := pmock.PlatformService{}
	lg := helper.LoggerMock{}

	rec := NewReconciler(k8sClient, &lg, &ps)
	require.NotNilf(t, rec == nil, "reconciler is not inited")

	require.Falsef(t, rec.client != k8sClient || rec.log != &lg, "wrong rec params")
}

func TestReconcile_Reconcile_FailureNotFound(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	lg := helper.LoggerMock{}

	r := Reconcile{
		client: k8sClient,
		log:    &lg,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: jbr.Namespace,
			Name:      "baz",
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.Equalf(t, "instance not found", lg.LastInfo(), "not found error is not logged")
}

func TestReconcile_Reconcile_FailureUnableToInitJenkinsClient(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	lg := helper.LoggerMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).
		Return(nil, errors.New("client mock fatal"))

	r := Reconcile{
		client:               k8sClient,
		log:                  &lg,
		jenkinsClientFactory: &jBuilder,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: jbr.Namespace,
			Name:      jbr.Name,
		},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.Error(t, err)

	require.Contains(t, err.Error(), "client mock fatal")
}

func TestReconcile_ReconcileNewBuild_FailureGetLastBuild(t *testing.T) {
	jbr := getTestJenkinsJobBuildRun()

	s := scheme.Scheme
	s.AddKnownTypes(v1.SchemeGroupVersion, jbr)

	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(jbr).Build()
	jClient := jenkins.ClientMock{}
	jBuilder := jenkins.ClientBuilderMock{}
	jBuilder.On("MakeNewClient", jbr.Spec.OwnerName).Return(&jClient, nil)

	job := gojenkins.Job{
		Raw: &gojenkins.JobResponse{
			InQueue: false,
		},
	}
	jClient.On("GetJobByName", "path/job").Return(&job, nil)
	jClient.On("GetLastBuild", &job).Return(nil, errors.New("last build fatal"))

	lg := helper.LoggerMock{}
	r := Reconcile{
		client:               k8sClient,
		jenkinsClientFactory: &jBuilder,
		log:                  &lg,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: jbr.Namespace, Name: jbr.Name},
	}

	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	lastErr := lg.LastError()
	require.Error(t, lastErr)

	require.Contains(t, lastErr.Error(), "last build fatal")
}
