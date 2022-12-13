package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

const (
	jenkinsJobFinalizerName = "jenkinsjob.finalizer.name"
	logNamespaceKey         = "Request.Namespace"
	logNameKey              = "name"
	logJenkinsJobKey        = "jenkins job"
	requeueAfter            = 10 * time.Second
)

func NewReconcileJenkinsJob(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkinsJob {
	return &ReconcileJenkinsJob{
		client:   k8sClient,
		scheme:   scheme,
		platform: ps,
		log:      log.WithName("jenkins-job"),
	}
}

type ReconcileJenkinsJob struct {
	client   client.Client
	scheme   *runtime.Scheme
	platform platform.PlatformService
	log      logr.Logger
}

func (r *ReconcileJenkinsJob) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsJob)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsJob)
			if !ok {
				return false
			}

			if !reflect.DeepEqual(oldObject.Spec, newObject.Spec) {
				return true
			}

			if newObject.DeletionTimestamp != nil {
				return true
			}

			return false
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsJob{}, builder.WithPredicates(p)).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func (r *ReconcileJenkinsJob) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues(logNamespaceKey, request.Namespace, "Request.Name", request.Name)
	log.Info("reconciling JenkinsJob had started")

	jenkinsJob := &jenkinsApi.JenkinsJob{}
	if err := r.client.Get(ctx, request.NamespacedName, jenkinsJob); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to Get JenkinsJob: %w", err)
	}

	if result, err := r.tryToDeleteJob(ctx, jenkinsJob); result != nil || err != nil {
		return *result, err
	}

	if err := r.setOwners(ctx, jenkinsJob); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to set owner reference: %w", err)
	}

	jobCanBeHandled, err := r.canJenkinsJobBeHandled(jenkinsJob)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check whether the jenkins job can be created: %w", err)
	}

	if !jobCanBeHandled {
		log.V(2).Info("jenkins folder for stages is not ready yet")
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	result, err := r.handleJob(jenkinsJob)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to handle JenkinsJob: %w", err)
	}

	if result.Requeue {
		r.log.Info("the next job provision will be triggered in few minutes", "minutes", result.RequeueAfter)

		return result, nil
	}

	log.Info("reconciling JenkinsJob has been finished")

	return result, nil
}

func (r *ReconcileJenkinsJob) handleJob(job *jenkinsApi.JenkinsJob) (reconcile.Result, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, job.Name, job.Namespace, job.Spec.OwnerName, job.GetOwnerReferences())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get jenkins owner for jenkins job %v: %w", job.Name, err)
	}

	jc, err := jenkinsClient.InitGoJenkinsClient(j, r.platform)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to init jenkins client %v: %w", j, err)
	}

	jobExists, err := jenkinsJobExists(jc, job.Spec.Job.Name)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to retrieve jenkins job %v; %w", job.Spec.Job.Name, err)
	}

	ch, err := r.getChain(jobExists)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to select chain: %w", err)
	}

	if err := chain.NewChain(ch).ServeRequest(job); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ServeRequest: %w", err)
	}

	if jobExists && job.IsAutoTriggerEnabled() {
		period := time.Duration(*job.Spec.Job.AutoTriggerPeriod) * time.Minute

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: period,
		}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsJob) getChain(jobExist bool) (chain.Chain, error) {
	if jobExist {
		ch, err := chain.InitTriggerJobProvisionChain(r.scheme, r.client)
		if err != nil {
			return ch, fmt.Errorf("failed to InitTriggerJobProvisionChain: %w", err)
		}

		return ch, nil
	}

	ch, err := chain.InitDefChain(r.scheme, r.client)
	if err != nil {
		return ch, fmt.Errorf("failed to InitDefChain: %w", err)
	}

	return ch, nil
}

func jenkinsJobExists(jc *jenkinsClient.JenkinsClient, jp string) (bool, error) {
	_, err := jc.GoJenkins.GetJob(jp)
	if err != nil {
		if helper.JenkinsIsNotFoundErr(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to GetJob: %w", err)
	}

	return true, nil
}

func (r *ReconcileJenkinsJob) initGoJenkinsClient(jj *jenkinsApi.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner for jenkins folder %v: %w", jj.Name, err)
	}

	r.log.Info("Jenkins instance has been received", logNameKey, j.Name)

	jClient, err := jenkinsClient.InitGoJenkinsClient(j, r.platform)
	if err != nil {
		return nil, fmt.Errorf("failed to InitGoJenkinsClient: %w", err)
	}

	return jClient, nil
}

func (r *ReconcileJenkinsJob) tryToDeleteJob(ctx context.Context, jj *jenkinsApi.JenkinsJob) (*reconcile.Result, error) {
	if jj.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName) {
			jj.ObjectMeta.Finalizers = append(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)

			if err := r.client.Update(ctx, jj); err != nil {
				return &reconcile.Result{}, fmt.Errorf("failed to Update jenkins job: %w", err)
			}
		}

		return nil, nil
	}

	if err := r.deleteJob(jj); err != nil {
		return &reconcile.Result{}, err
	}

	jj.ObjectMeta.Finalizers = finalizer.RemoveString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)

	if err := r.client.Update(ctx, jj); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to Update jenkins job: %w", err)
	}

	return &reconcile.Result{}, nil
}

func (r *ReconcileJenkinsJob) deleteJob(jj *jenkinsApi.JenkinsJob) error {
	jc, err := r.initGoJenkinsClient(jj)
	if err != nil {
		return fmt.Errorf("failed to create Go Jenkins client: %w", err)
	}

	j := r.getJobName(jj)

	_, err = jc.GoJenkins.DeleteJob(j)
	if err != nil {
		if helper.JenkinsIsNotFoundErr(err) {
			r.log.V(2).Info("job/folder doesn't exist. skip deleting", logNameKey, j)

			return nil
		}

		return fmt.Errorf("failed to DeleteJob: %w", err)
	}

	return nil
}

func (r *ReconcileJenkinsJob) getJobName(jj *jenkinsApi.JenkinsJob) string {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		jobName, err := r.getStageJobName(jj)
		if err != nil {
			return "an error had occurred while getting jenkins job name"
		}

		return fmt.Sprintf("%v-cd-pipeline/job/%v", *jj.Spec.JenkinsFolder, jobName)
	}

	return jj.Spec.Job.Name
}

func (r *ReconcileJenkinsJob) setOwners(ctx context.Context, jj *jenkinsApi.JenkinsJob) error {
	if err := r.tryToSetStageOwnerRef(jj); err != nil {
		return err
	}

	if err := r.client.Update(ctx, jj); err != nil {
		return fmt.Errorf("failed to update jenkins job %v: %w", jj.Name, err)
	}

	return nil
}

func (r *ReconcileJenkinsJob) tryToSetStageOwnerRef(jj *jenkinsApi.JenkinsJob) error {
	if ow := plutil.GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		r.log.V(2).Info("stage ref already exists", logJenkinsJobKey, jj.Name)

		return nil
	}

	s, err := plutil.GetStageInstance(r.client, *jj.Spec.StageName, jj.Namespace)
	if err != nil {
		return fmt.Errorf("failed to GetStageInstance: %w", err)
	}

	if err := controllerutil.SetControllerReference(s, jj, r.scheme); err != nil {
		return fmt.Errorf("failed to set stage owner ref: %w", err)
	}

	return nil
}

func (r *ReconcileJenkinsJob) canJenkinsJobBeHandled(jj *jenkinsApi.JenkinsJob) (bool, error) {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		jfn := fmt.Sprintf("%v-%v", *jj.Spec.JenkinsFolder, "cd-pipeline")

		jf, err := plutil.GetJenkinsFolderInstance(r.client, jfn, jj.Namespace)
		if err != nil {
			return false, fmt.Errorf("failed to GetJenkinsFolderInstance: %w", err)
		}

		r.log.V(2).Info("create job in Jenkins folder", logNameKey, jfn, "status folder", jf.Status.Available)

		return jf.Status.Available, nil
	}

	r.log.V(2).Info("create job in Jenkins root folder", logNameKey, jj.Name)

	return true, nil
}

func (*ReconcileJenkinsJob) getStageJobName(jj *jenkinsApi.JenkinsJob) (string, error) {
	jobConfig := make(map[string]string)

	if err := json.Unmarshal([]byte(jj.Spec.Job.Config), &jobConfig); err != nil {
		return "", fmt.Errorf("failed to Unmarshal: %w", err)
	}

	stageName := jobConfig["STAGE_NAME"]

	return stageName, nil
}
