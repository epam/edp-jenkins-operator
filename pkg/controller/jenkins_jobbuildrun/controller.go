package jenkins_jobbuildrun

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	retryInterval = 10 * time.Second
)

type Reconcile struct {
	client               client.Client
	log                  logr.Logger
	jenkinsClientFactory jenkins.ClientFactory
}

func NewReconciler(k8sCl client.Client, logf logr.Logger, ps platform.PlatformService) *Reconcile {
	return &Reconcile{
		client:               k8sCl,
		log:                  logf.WithName("controller_jenkins_jobbuildrun"),
		jenkinsClientFactory: jenkins.MakeClientBuilder(ps, k8sCl),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: specUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsJobBuildRun{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func specUpdated(e event.UpdateEvent) bool {
	oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsJobBuildRun)
	if !ok {
		return false
	}

	newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsJobBuildRun)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oldObject.Spec, newObject.Spec) ||
		(oldObject.GetDeletionTimestamp().IsZero() && !newObject.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var result reconcile.Result

	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsJobBuildRun has been started")

	var instance jenkinsApi.JenkinsJobBuildRun
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.Info("instance not found")

			return result, nil
		}

		return result, fmt.Errorf("failed to get JenkinsJobBuildRun instance: %w", err)
	}

	if instance.Status.Status == jenkinsApi.JobBuildRunStatusCompleted {
		reqLogger.V(2).Info("Reconciling JenkinsJobBuildRun has been finished, job already completed")

		if err := r.deleteExpiredBuilds(&instance); err != nil {
			return result, fmt.Errorf("failed to delete expired builds: %w", err)
		}

		return result, nil
	}

	jc, err := r.jenkinsClientFactory.MakeNewClient(&instance.ObjectMeta, instance.Spec.OwnerName)
	if err != nil {
		return result,
			fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	requeue, err := tryToReconcile(&instance, jc)
	if err != nil {
		r.log.Error(err, "error during reconciliation", "instance", instance)

		result.RequeueAfter = helper.DefaultRequeueTime * time.Second

		return result, nil
	}

	result.RequeueAfter = requeue
	instance.Status.LastUpdated = metav1.NewTime(time.Now())

	if err := r.client.Status().Update(context.Background(), &instance); err != nil {
		r.log.Error(err, "unable to update status", "instance", instance)
	}

	reqLogger.V(2).Info("Reconciling JenkinsJobBuildRun has been finished")

	return result, nil
}

func tryToReconcile(instance *jenkinsApi.JenkinsJobBuildRun, jc jenkins.ClientInterface) (time.Duration, error) {
	job, err := jc.GetJobByName(instance.Spec.JobPath) // check if job exists
	if err != nil {
		if helper.JenkinsIsNotFoundErr(err) {
			// job is not found, returning error and setting not found status for CR
			instance.Status.Status = jenkinsApi.JobBuildRunStatusNotFound

			return 0, nil
		}
		// unknown error
		return 0, fmt.Errorf("failed to get job by name: %s: %w", instance.Spec.JobPath, err)
	}

	// job exists, and it's already in queue, stop here and check later after specified interval
	if job.Raw.InQueue {
		return retryInterval, nil
	}

	// check latest job build
	interval, err := checkLastBuild(job, instance, jc)
	if err != nil {
		return 0, fmt.Errorf("failed to check latest build: %w", err)
	}

	return interval, nil
}

func checkLastBuild(job *gojenkins.Job, instance *jenkinsApi.JenkinsJobBuildRun,
	jc jenkins.ClientInterface,
) (time.Duration, error) {
	build, err := jc.GetLastBuild(job)
	if err != nil {
		// job does not have any builds so we can trigger new one
		if helper.JenkinsIsNotFoundErr(err) {
			return retryInterval, triggerNewBuild(instance, jc, jenkinsApi.JobBuildRunStatusCreated)
		}

		// unknown error
		return 0, fmt.Errorf("failed to get last build: %w", err)
	}

	// check if latest build already running
	if jc.BuildIsRunning(build) {
		return retryInterval, nil // latest build already running, stop here and check later after specified interval
	}

	// if job has latest build we must check if it was created by this controller
	if build.GetBuildNumber() == instance.Status.BuildNumber { // build created by this controller
		if build.GetResult() == gojenkins.STATUS_SUCCESS { // build finished with success, so we can set Completed status to CR and exit
			instance.Status.Status = jenkinsApi.JobBuildRunStatusCompleted

			return instance.GetDeleteAfterCompletionInterval(), nil
		}

		// build was not finished with success, so we must check how many times we already started it
		if instance.Spec.Retry > instance.Status.Launches { // launches is less than amount of specified retries
			return retryInterval, triggerNewBuild(instance, jc, jenkinsApi.JobBuildRunStatusRetrying)
		}

		// we reach amount of specified retries so job is failed, exit
		instance.Status.Status = jenkinsApi.JobBuildRunStatusFailed

		return 0, nil
	}

	// latest job was not created by this controller so we can trigger a new one
	return retryInterval, triggerNewBuild(instance, jc, jenkinsApi.JobBuildRunStatusCreated)
}

func triggerNewBuild(
	instance *jenkinsApi.JenkinsJobBuildRun,
	jc jenkins.ClientInterface,
	status string,
) error {
	buildNumber, err := jc.BuildJob(instance.Spec.JobPath, instance.Spec.Params)
	if err != nil {
		return fmt.Errorf("failed to build job: %w", err)
	}

	instance.Status.Status = status
	instance.Status.Launches++
	instance.Status.BuildNumber = *buildNumber

	return nil
}

func (r *Reconcile) deleteExpiredBuilds(instance *jenkinsApi.JenkinsJobBuildRun) error {
	cond := time.Now().After(
		instance.Status.LastUpdated.Add(
			instance.GetDeleteAfterCompletionInterval(),
		),
	)

	if cond {
		if err := r.client.Delete(context.Background(), instance); err != nil {
			return fmt.Errorf("failed to delete expired build: %w", err)
		}
	}

	return nil
}
