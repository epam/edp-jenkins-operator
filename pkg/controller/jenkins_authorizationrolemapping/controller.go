package jenkins_authorizationrolemapping

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

const finalizerName = "jenkinsauthrolemapping.jenkins.finalizer.name"

type Reconcile struct {
	client               client.Client
	log                  logr.Logger
	jenkinsClientFactory jenkins.ClientFactory
}

func NewReconciler(k8sCl client.Client, logf logr.Logger, ps platform.PlatformService) *Reconcile {
	return &Reconcile{
		client:               k8sCl,
		log:                  logf.WithName("controller_jenkins_authorizationrolemapping"),
		jenkinsClientFactory: jenkins.MakeClientBuilder(ps, k8sCl),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: specUpdated,
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsAuthorizationRoleMapping{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func specUpdated(e event.UpdateEvent) bool {
	oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsAuthorizationRoleMapping)
	if !ok {
		return false
	}

	newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsAuthorizationRoleMapping)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oldObject.Spec, newObject.Spec) ||
		(oldObject.GetDeletionTimestamp().IsZero() && !newObject.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsAuthorizationRoleMapping has been started")

	instance := new(jenkinsApi.JenkinsAuthorizationRoleMapping)

	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get JenkinsAuthorizationRoleMapping instance %q: %w", request.NamespacedName, err)
	}

	defaultRequeueResult := reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}

	jc, err := r.jenkinsClientFactory.MakeNewClient(&instance.ObjectMeta, instance.Spec.OwnerName)
	if err != nil {
		r.log.Error(err, "error during client creation", "instance", instance)

		return defaultRequeueResult, fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	if err := r.tryToReconcile(ctx, instance, jc); err != nil {
		r.log.Error(err, "error during reconciliation", "instance", instance)
		r.updateInstanceStatus(ctx, instance, err.Error())

		return defaultRequeueResult, nil
	}

	r.updateInstanceStatus(ctx, instance, helper.StatusSuccess)
	reqLogger.Info("Reconciling JenkinsAuthorizationRoleMapping has been finished")

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryToReconcile(
	ctx context.Context,
	instance *jenkinsApi.JenkinsAuthorizationRoleMapping,
	jenkinsClient jenkins.ClientInterface,
) error {
	for _, rl := range instance.Spec.Roles {
		if err := jenkinsClient.AssignRole(instance.Spec.RoleType, rl, instance.Spec.Group); err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
	}

	updateNeeded, err := helper.TryToDelete(instance, finalizerName, makeDeletionFunc(instance, jenkinsClient))
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if !updateNeeded {
		return nil
	}

	if err := r.client.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

func makeDeletionFunc(instance *jenkinsApi.JenkinsAuthorizationRoleMapping,
	jc jenkins.ClientInterface,
) func() error {
	return func() error {
		for _, rl := range instance.Spec.Roles {
			if err := jc.UnAssignRole(instance.Spec.RoleType, rl, instance.Spec.Group); err != nil {
				return fmt.Errorf("failed to unassign role: %w", err)
			}
		}

		return nil
	}
}

func (r *Reconcile) updateInstanceStatus(
	ctx context.Context,
	instance *jenkinsApi.JenkinsAuthorizationRoleMapping,
	statusValue string,
) {
	instance.Status.Value = statusValue

	if err := r.client.Status().Update(ctx, instance); err != nil {
		r.log.Error(err, "unable to update status", "instance", instance)
	}
}
