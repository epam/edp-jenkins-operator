package cdstagejenkinsdeployment

import (
	"context"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	chain "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/factory"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	cdStageDeployKey                = "cdStageDeployName"
	foregroundDeletionFinalizerName = "foregroundDeletion"
)

type ReconcileCDStageJenkinsDeployment struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

func (r *ReconcileCDStageJenkinsDeployment) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CDStageJenkinsDeployment{}).
		Complete(r)
}

func (r *ReconcileCDStageJenkinsDeployment) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("reconciling has been started")

	i := &v1alpha1.CDStageJenkinsDeployment{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defer func() {
		if err := r.updateStatus(ctx, i); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizer(ctx, i); err != nil {
		err := errors.Wrapf(err, "cannot set %v finalizer", foregroundDeletionFinalizerName)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := r.setOwnerReference(ctx, i); err != nil {
		err := errors.Wrapf(err, "cannot set owner ref for %v CDStageJenkinsDeployment", i.Name)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	platform, err := ps.NewPlatformService(helper.GetPlatformTypeEnv(), r.Scheme, &r.Client)
	if err != nil {
		err := errors.Wrap(err, "couldn't create platform service")
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.Client, platform).ServeRequest(i); err != nil {
		err := errors.Wrapf(err, "an error has occurred during triggering deploy jenkins job")
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}
	i.SetSuccessStatus()

	log.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageJenkinsDeployment) setFinalizer(ctx context.Context, jd *v1alpha1.CDStageJenkinsDeployment) error {
	if !jd.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if !finalizer.ContainsString(jd.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		jd.ObjectMeta.Finalizers = append(jd.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
	}
	return r.Client.Update(ctx, jd)
}

func (r *ReconcileCDStageJenkinsDeployment) updateStatus(ctx context.Context, jenkinsDeployment *v1alpha1.CDStageJenkinsDeployment) error {
	if err := r.Client.Status().Update(ctx, jenkinsDeployment); err != nil {
		if err := r.Client.Update(ctx, jenkinsDeployment); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageJenkinsDeployment) setOwnerReference(ctx context.Context, jenkinsDeployment *jenkinsApi.CDStageJenkinsDeployment) error {
	s, err := r.getCDStageDeploy(ctx, jenkinsDeployment.Labels[cdStageDeployKey], jenkinsDeployment.Namespace)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(s, jenkinsDeployment, r.Scheme)
}

func (r *ReconcileCDStageJenkinsDeployment) getCDStageDeploy(ctx context.Context, name, ns string) (*codebaseApi.CDStageDeploy, error) {
	r.Log.Info("getting cd stage deploy", "name", name)
	i := &codebaseApi.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	if err := r.Client.Get(ctx, nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
