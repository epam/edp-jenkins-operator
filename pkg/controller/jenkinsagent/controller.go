package jenkinsagent

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
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

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsAgent{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func specUpdated(e event.UpdateEvent) bool {
	oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsAgent)
	if !ok {
		return false
	}

	newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsAgent)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oldObject.Spec, newObject.Spec) ||
		(oldObject.GetDeletionTimestamp().IsZero() && !newObject.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsAgent has been started")

	var instance jenkinsApi.JenkinsAgent
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.V(2).Info("JenkinsAgent is not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get JenkinsAgent instance: %w", err)
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

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *jenkinsApi.JenkinsAgent) error {
	var slavesCm v1.ConfigMap

	if err := r.client.Get(
		ctx,
		types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      jenkins.SlavesTemplateName,
		},
		&slavesCm,
	); err != nil {
		return fmt.Errorf("failed to get slaves config map: %w", err)
	}

	slavesCm.Data[instance.Spec.SalvesKey()] = instance.Spec.Template

	if err := r.client.Update(ctx, &slavesCm); err != nil {
		return fmt.Errorf("failed to update slaves config map: %w", err)
	}

	updateNeeded, err := helper.TryToDelete(instance, finalizerName, makeDeletionFunc(ctx, r.client, instance))
	if err != nil {
		return fmt.Errorf("failed to delete jenkins agent: %w", err)
	}

	if !updateNeeded {
		return nil
	}

	if err := r.client.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

func makeDeletionFunc(ctx context.Context, k8sClient client.Client, instance *jenkinsApi.JenkinsAgent) func() error {
	return func() error {
		var slavesCm v1.ConfigMap

		if err := k8sClient.Get(
			ctx,
			types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      jenkins.SlavesTemplateName,
			},
			&slavesCm,
		); err != nil {
			return fmt.Errorf("failed to get slaves config map: %w", err)
		}

		delete(slavesCm.Data, instance.Spec.SalvesKey())

		if err := k8sClient.Update(ctx, &slavesCm); err != nil {
			return fmt.Errorf("failed to update slaves config map: %w", err)
		}

		return nil
	}
}
