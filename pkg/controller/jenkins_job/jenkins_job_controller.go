package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job/chain"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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
)

const jenkinsJobFinalizerName = "jenkinsjob.finalizer.name"

func NewReconcileJenkinsJob(client client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkinsJob {
	return &ReconcileJenkinsJob{
		client:   client,
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
			oo := e.ObjectOld.(*jenkinsApi.JenkinsJob)
			no := e.ObjectNew.(*jenkinsApi.JenkinsJob)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsJob{}, builder.WithPredicates(p)).WithOptions(controller.Options{
		MaxConcurrentReconciles: maxConcurrentReconciles,
	}).
		Complete(r)
}

func (r *ReconcileJenkinsJob) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("reconciling JenkinsJob has been started")

	i := &jenkinsApi.JenkinsJob{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if result, err := r.tryToDeleteJob(ctx, i); result != nil || err != nil {
		return *result, err
	}

	if err := r.setOwners(ctx, i); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while setting owner reference")
	}

	c, err := r.canJenkinsJobBeHandled(i)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while checking availability of creating jenkins job")
	}
	if !c {
		log.V(2).Info("jenkins folder for stages is not ready yet")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	j, err := plutil.GetJenkinsInstanceOwner(r.client, i.Name, i.Namespace, i.Spec.OwnerName, i.GetOwnerReferences())
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins job %v", i.Name)
	}

	jc, err := jenkinsClient.InitGoJenkinsClient(j, r.platform)
	jobExist, err := isJenkinsJobExist(jc, i.Spec.Job.Name)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has occurred while retrieving jenkins job %v", i.Spec.Job.Name)
	}

	ch, err := r.getChain(jobExist)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := chain.NewChain(ch).ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}

	if jobExist && i.IsAutoTriggerEnabled() {
		period := time.Duration(*i.Spec.Job.AutoTriggerPeriod) * time.Minute
		r.log.Info("the next job provision will be triggered in few minutes", "minutes", period)
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: period,
		}, nil
	}

	log.Info("reconciling JenkinsJob has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsJob) getChain(jobExist bool) (chain.Chain, error) {
	if jobExist {
		return chain.InitTriggerJobProvisionChain(r.scheme, r.client)
	}
	return chain.InitDefChain(r.scheme, r.client)
}

func isJenkinsJobExist(jc *jenkinsClient.JenkinsClient, jp string) (bool, error) {
	_, err := jc.GoJenkins.GetJob(jp)
	if err != nil {
		if err.Error() == "404" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r ReconcileJenkinsJob) initGoJenkinsClient(jj jenkinsApi.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jj.Name)
	}
	r.log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, r.platform)
}

func (r ReconcileJenkinsJob) tryToDeleteJob(ctx context.Context, jj *jenkinsApi.JenkinsJob) (*reconcile.Result, error) {
	if jj.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName) {
			jj.ObjectMeta.Finalizers = append(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)
			if err := r.client.Update(ctx, jj); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return nil, nil
	}

	if err := r.deleteJob(jj); err != nil {
		return &reconcile.Result{}, err
	}

	jj.ObjectMeta.Finalizers = finalizer.RemoveString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)
	if err := r.client.Update(ctx, jj); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r ReconcileJenkinsJob) deleteJob(jj *jenkinsApi.JenkinsJob) error {
	jc, err := r.initGoJenkinsClient(*jj)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	j := r.getJobName(jj)
	if _, err := jc.GoJenkins.DeleteJob(j); err != nil {
		if err.Error() == "404" {
			r.log.V(2).Info("job/folder doesn't exist. skip deleting", "name", j)
			return nil
		}
		return err
	}
	return nil
}

func (r ReconcileJenkinsJob) getJobName(jj *jenkinsApi.JenkinsJob) string {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		var jobName, err = r.getStageJobName(jj)
		if err != nil {
			return "an error has been occurred while getting jenkins job name"
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
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", jj.Name)
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetJenkinsOwnerRef(jj *jenkinsApi.JenkinsJob) error {
	if ow := plutil.GetOwnerReference(consts.JenkinsKind, jj.GetOwnerReferences()); ow != nil {
		r.log.V(2).Info("jenkins owner ref already exists", "jenkins job", jj.Name)
		return nil
	}

	j, err := plutil.GetFirstJenkinsInstance(r.client, jj.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(j, jj, r.scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetStageOwnerRef(jj *jenkinsApi.JenkinsJob) error {
	if ow := plutil.GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		r.log.V(2).Info("stage ref already exists", "jenkins job", jj.Name)
		return nil
	}

	s, err := plutil.GetStageInstance(r.client, *jj.Spec.StageName, jj.Namespace)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(s, jj, r.scheme); err != nil {
		return errors.Wrap(err, "couldn't set stage owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetJenkinsFolderOwnerRef(jj *jenkinsApi.JenkinsJob) error {
	if jj.Spec.JenkinsFolder == nil || *jj.Spec.JenkinsFolder == "" {
		r.log.V(2).Info("skip setting jenkins folder reference", "jenkins job", jj.Name)
		return nil
	}

	if ow := plutil.GetOwnerReference(consts.JenkinsFolderKind, jj.GetOwnerReferences()); ow != nil {
		r.log.V(2).Info("jenkins folder ref already exists", "jenkins job", jj.Name)
		return nil
	}

	jf, err := plutil.GetJenkinsFolderInstance(r.client, *jj.Spec.JenkinsFolder, jj.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(jf, jj, r.scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins folder owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) canJenkinsJobBeHandled(jj *jenkinsApi.JenkinsJob) (bool, error) {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		jfn := fmt.Sprintf("%v-%v", *jj.Spec.JenkinsFolder, "cd-pipeline")
		jf, err := plutil.GetJenkinsFolderInstance(r.client, jfn, jj.Namespace)
		if err != nil {
			return false, err
		}
		r.log.V(2).Info("create job in Jenkins folder", "name", jfn, "status folder", jf.Status.Available)
		return jf.Status.Available, nil
	}
	r.log.V(2).Info("create job in Jenkins root folder", "name", jj.Name)
	return true, nil
}

func (r ReconcileJenkinsJob) getStageJobName(jj *jenkinsApi.JenkinsJob) (string, error) {
	jobConfig := make(map[string]string)
	err := json.Unmarshal([]byte(jj.Spec.Job.Config), &jobConfig)
	if err != nil {
		return "", err
	}
	var stageName = jobConfig["STAGE_NAME"]
	return stageName, nil
}
