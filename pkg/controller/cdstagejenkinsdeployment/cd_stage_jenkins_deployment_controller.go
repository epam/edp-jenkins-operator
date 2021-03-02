package cdstagejenkinsdeployment

import (
	"context"
	v1alpha1Codebase "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	chain "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain/factory"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	return &ReconcileCDStageJenkinsDeployment{
		client: mgr.GetClient(),
		scheme: scheme,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	schemeGroupVersionV2 := schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}
	scheme.AddKnownTypes(schemeGroupVersionV2,
		&v1alpha1Codebase.CDStageDeploy{},
		&v1alpha1Codebase.CDStageDeployList{},
	)
	metav1.AddToGroupVersion(scheme, schemeGroupVersionV2)
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

const (
	cdStageDeployKey                = "cdStageDeployName"
	foregroundDeletionFinalizerName = "foregroundDeletion"
)

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

	defer func() {
		if err := r.updateStatus(i); err != nil {
			vLog.Error(err, "error during status updating")
		}
	}()

	if err := r.setFinalizer(i); err != nil {
		err := errors.Wrapf(err, "cannot set %v finalizer", foregroundDeletionFinalizerName)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := r.setOwnerReference(i); err != nil {
		err := errors.Wrapf(err, "cannot set owner ref for %v CDStageJenkinsDeployment", i.Name)
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	platform, err := ps.NewPlatformService(helper.GetPlatformTypeEnv(), r.scheme, &r.client)
	if err != nil {
		err := errors.Wrap(err, "couldn't create platform service")
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}

	if err := chain.CreateDefChain(r.client, platform).ServeRequest(i); err != nil {
		err := errors.Wrapf(err, "an error has occurred during triggering deploy jenkins job")
		i.SetFailedStatus(err)
		return reconcile.Result{}, err
	}
	i.SetSuccessStatus()

	vLog.Info("Reconciling has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageJenkinsDeployment) setFinalizer(jd *v1alpha1.CDStageJenkinsDeployment) error {
	if !jd.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if !finalizer.ContainsString(jd.ObjectMeta.Finalizers, foregroundDeletionFinalizerName) {
		jd.ObjectMeta.Finalizers = append(jd.ObjectMeta.Finalizers, foregroundDeletionFinalizerName)
	}
	return r.client.Update(context.TODO(), jd)
}

func (r *ReconcileCDStageJenkinsDeployment) updateStatus(jenkinsDeployment *v1alpha1.CDStageJenkinsDeployment) error {
	if err := r.client.Status().Update(context.TODO(), jenkinsDeployment); err != nil {
		if err := r.client.Update(context.TODO(), jenkinsDeployment); err != nil {
			return err
		}
	}
	return nil
}

func (r *ReconcileCDStageJenkinsDeployment) setOwnerReference(jenkinsDeployment *v1alpha1.CDStageJenkinsDeployment) error {
	s, err := r.getCDStageDeploy(jenkinsDeployment.Labels[cdStageDeployKey], jenkinsDeployment.Namespace)
	if err != nil {
		return err
	}
	return controllerutil.SetControllerReference(s, jenkinsDeployment, r.scheme)
}

func (r *ReconcileCDStageJenkinsDeployment) getCDStageDeploy(name, ns string) (*v1alpha1Codebase.CDStageDeploy, error) {
	log.Info("getting cd stage deploy", "name", name)
	i := &v1alpha1Codebase.CDStageDeploy{}
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	if err := r.client.Get(context.TODO(), nn, i); err != nil {
		return nil, err
	}
	return i, nil
}
