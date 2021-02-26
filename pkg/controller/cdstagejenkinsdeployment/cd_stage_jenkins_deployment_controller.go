package cdstagejenkinsdeployment

import (
	"context"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	chain "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/factory"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("v")

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCDStageJenkinsDeployment{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("cd-stage-jenkins-deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &v1alpha1.CDStageJenkinsDeployment{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCDStageJenkinsDeployment{}

type ReconcileCDStageJenkinsDeployment struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileCDStageJenkinsDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	vLog := log.WithValues("type", "CDStageJenkinsDeployment", "Request.Namespace", request.Namespace, "Request.Name", request.Name)
	vLog.Info("reconciling has been started")

	i := &v1alpha1.CDStageJenkinsDeployment{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if err := r.setOwnerRef(i); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "cannot set owner ref for %v CDStageDeployJenkins", i.Name)
	}

	platform, err := ps.NewPlatformService(helper.GetPlatformTypeEnv(), r.scheme, &r.client)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "couldn't create platform service")
	}

	if err := chain.CreateDefChain(r.client, platform).ServeRequest(i); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has occurred during triggering deploy jenkins job")
	}

	vLog.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageJenkinsDeployment) setOwnerRef(jenkinsDeployment *v1alpha1.CDStageJenkinsDeployment) error {
	j, err := platform.GetFirstJenkinsInstance(r.client, jenkinsDeployment.Namespace)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(j, jenkinsDeployment, r.scheme)
}
