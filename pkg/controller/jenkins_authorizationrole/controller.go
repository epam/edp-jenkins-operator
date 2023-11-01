package jenkins_authorizationrole

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

const finalizerName = "jenkinsauthrole.jenkins.finalizer.name"

type Reconcile struct {
	client               client.Client
	log                  logr.Logger
	jenkinsClientFactory jenkins.ClientFactory
}

func NewReconciler(k8sCl client.Client, logf logr.Logger, ps platform.PlatformService) *Reconcile {
	return &Reconcile{
		client:               k8sCl,
		log:                  logf.WithName("controller_jenkins_authorizationrole"),
		jenkinsClientFactory: jenkins.MakeClientBuilder(ps, k8sCl),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: specUpdated,
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsAuthorizationRole{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func specUpdated(e event.UpdateEvent) bool {
	oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsAuthorizationRole)
	if !ok {
		return false
	}

	newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsAuthorizationRole)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oldObject.Spec, newObject.Spec) ||
		(oldObject.GetDeletionTimestamp().IsZero() && !newObject.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsAuthorizationRole has been started")

	instance := new(jenkinsApi.JenkinsAuthorizationRole)

	if err := r.client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get JenkinsAuthorizationRole instance: %w", err)
	}

	jc, err := r.jenkinsClientFactory.MakeNewClient(&instance.ObjectMeta, instance.Spec.OwnerName)
	if err != nil {
		return reconcile.Result{},
			fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	if err := r.tryToReconcile(ctx, instance, jc); err != nil {
		r.log.Error(err, "error during reconciliation", "instance", instance)
		r.updateInstanceStatus(ctx, instance, err.Error())

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	r.updateInstanceStatus(ctx, instance, helper.StatusSuccess)

	reqLogger.Info("Reconciling JenkinsAuthorizationRole has been finished")

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryToReconcile(
	ctx context.Context,
	instance *jenkinsApi.JenkinsAuthorizationRole,
	jc jenkins.ClientInterface,
) error {
	if err := jc.AddRole(instance.Spec.RoleType, instance.Spec.Name, instance.Spec.Pattern, instance.Spec.Permissions); err != nil {
		return fmt.Errorf("failed to add role: %w", err)
	}

	updateNeeded, err := helper.TryToDelete(instance, finalizerName, makeDeletionFunc(instance, jc))
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if updateNeeded {
		if err := r.client.Update(ctx, instance); err != nil {
			return fmt.Errorf("failed to update instance: %w", err)
		}
	}

	return nil
}

func makeDeletionFunc(instance *jenkinsApi.JenkinsAuthorizationRole,
	jc jenkins.ClientInterface,
) func() error {
	return func() error {
		if err := jc.RemoveRoles(instance.Spec.RoleType, []string{instance.Spec.Name}); err != nil {
			return fmt.Errorf("failed to delete role: %w", err)
		}

		return nil
	}
}

func (r *Reconcile) updateInstanceStatus(
	ctx context.Context,
	instance *jenkinsApi.JenkinsAuthorizationRole,
	statusValue string,
) {
	instance.Status.Value = statusValue

	if err := r.client.Status().Update(ctx, instance); err != nil {
		r.log.Error(err, "unable to update status", "instance", instance)
	}
}
