package jenkinsagent

import (
	"context"
	"reflect"
	"time"

	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "jenkinsagent.jenkins.finalizer.name"

type Reconcile struct {
	client client.Client
	log    logr.Logger
}

func NewReconciler(k8sCl client.Client, logf logr.Logger) *Reconcile {
	return &Reconcile{
		client: k8sCl,
		log:    logf.WithName("controller_jenkins_authorizationrole"),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: specUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v2v1alpha1.JenkinsAgent{}, builder.WithPredicates(p)).
		Complete(r)
}

func specUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v2v1alpha1.JenkinsAgent)
	no := e.ObjectNew.(*v2v1alpha1.JenkinsAgent)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsAgent has been started")

	var instance v2v1alpha1.JenkinsAgent
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.V(2).Info("JenkinsAgent is not found")
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get JenkinsAgent instance")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			r.log.Error(err, "unable to update status", "instance", instance)
		}
	}()

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		r.log.Error(err, "error during reconcilation", "instance", instance)
		instance.Status.Value = err.Error()
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}
	instance.Status.Value = helper.StatusSuccess

	reqLogger.V(2).Info("Reconciling JenkinsAgent has been finished")
	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v2v1alpha1.JenkinsAgent) error {
	var slavesCm v1.ConfigMap
	if err := r.client.Get(ctx,
		types.NamespacedName{Namespace: instance.Namespace, Name: jenkins.SlavesTemplateName}, &slavesCm); err != nil {
		return errors.Wrap(err, "unable to get slaves config map")
	}

	slavesCm.Data[instance.Spec.SalvesKey()] = instance.Spec.Template

	if err := r.client.Update(ctx, &slavesCm); err != nil {
		return errors.Wrap(err, "unable to update slaves config map")
	}

	updateNeeded, err := helper.TryToDelete(instance, finalizerName, makeDeletionFunc(ctx, r.client, instance))
	if err != nil {
		return errors.Wrap(err, "unable to delete jenkins agent")
	}

	if updateNeeded {
		if err := r.client.Update(ctx, instance); err != nil {
			return errors.Wrap(err, "unable to update instance")
		}
	}

	return nil
}

func makeDeletionFunc(ctx context.Context, k8sClient client.Client, instance *v2v1alpha1.JenkinsAgent) func() error {
	return func() error {
		var slavesCm v1.ConfigMap
		if err := k8sClient.Get(ctx,
			types.NamespacedName{Namespace: instance.Namespace, Name: jenkins.SlavesTemplateName}, &slavesCm); err != nil {
			return errors.Wrap(err, "unable to get slaves config map")
		}

		delete(slavesCm.Data, instance.Spec.SalvesKey())

		if err := k8sClient.Update(ctx, &slavesCm); err != nil {
			return errors.Wrap(err, "unable to update slaves config map")
		}

		return nil
	}
}
