package cdstagejenkinsdeployment

import (
	"context"
	"time"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain"
	cdStageJenkinshelper "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

func NewReconcileCDStageJenkinsDeployment(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCDStageJenkinsDeployment {
	return &ReconcileCDStageJenkinsDeployment{
		client: client,
		scheme: scheme,
		log:    log.WithName("cd-stage-jenkins-deployment"),
	}
}

type ReconcileCDStageJenkinsDeployment struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileCDStageJenkinsDeployment) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.CDStageJenkinsDeployment{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileCDStageJenkinsDeployment) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("reconciling has been started")

	i := &jenkinsApi.CDStageJenkinsDeployment{}
	if err := r.client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defer func() {
		if err := r.updateStatus(ctx, i); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setOwnerReference(i); err != nil {
		err := errors.Wrapf(err, "cannot set owner ref for %v CDStageJenkinsDeployment", i.Name)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	env, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return reconcile.Result{}, err
	}
	platform, err := ps.NewPlatformService(env, r.scheme, r.client)
	if err != nil {
		err := errors.Wrap(err, "couldn't create platform service")
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client, platform).ServeRequest(i); err != nil {
		i.SetFailedStatus(err)
		p := r.setReconcilationPeriod(i)
		return reconcile.Result{RequeueAfter: p}, nil
	}

	log.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageJenkinsDeployment) setReconcilationPeriod(jd *jenkinsApi.CDStageJenkinsDeployment) time.Duration {
	timeout := util.GetTimeout(jd.Status.FailureCount, 500*time.Millisecond)
	r.log.Info("wait for next reconcilation", "next reconcilation in", timeout)
	jd.Status.FailureCount += 1
	return timeout
}

func (r *ReconcileCDStageJenkinsDeployment) updateStatus(ctx context.Context, jenkinsDeployment *jenkinsApi.CDStageJenkinsDeployment) error {
	if err := r.client.Status().Update(ctx, jenkinsDeployment); err != nil {
		if err := r.client.Update(ctx, jenkinsDeployment); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageJenkinsDeployment) setOwnerReference(jenkinsDeployment *jenkinsApi.CDStageJenkinsDeployment) error {
	s, err := cdStageJenkinshelper.GetCDStageDeploy(r.client, jenkinsDeployment.Labels[consts.CdStageDeployKey], jenkinsDeployment.Namespace)
	if err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(s, jenkinsDeployment, r.scheme); err != nil {
		return err
	}
	return r.client.Update(context.TODO(), jenkinsDeployment)
}
