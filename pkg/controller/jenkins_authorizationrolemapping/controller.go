package jenkins_authorizationrolemapping

import (
	"context"
	"reflect"
	"time"

	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "jenkinsauthrolemapping.jenkins.finalizer.name"

type Reconcile struct {
	client               client.Client
	log                  logr.Logger
	jenkinsClientFactory jenkins.ClientFactory
}

func NewReconciler(k8sCl client.Client, logf logr.Logger,
	ps platform.PlatformService) *Reconcile {

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

	return ctrl.NewControllerManagedBy(mgr).
		For(&v2v1alpha1.JenkinsAuthorizationRoleMapping{}, builder.WithPredicates(p)).
		Complete(r)
}

func specUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v2v1alpha1.JenkinsAuthorizationRoleMapping)
	no := e.ObjectNew.(*v2v1alpha1.JenkinsAuthorizationRoleMapping)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsAuthorizationRoleMapping has been started")

	var instance v2v1alpha1.JenkinsAuthorizationRoleMapping
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get JenkinsAuthorizationRoleMapping instance")
	}

	jc, err := r.jenkinsClientFactory.MakeNewClient(&instance.ObjectMeta, instance.Spec.OwnerName)
	if err != nil {
		return reconcile.Result{},
			errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	defer func() {
		if err := r.client.Status().Update(context.Background(), &instance); err != nil {
			r.log.Error(err, "unable to update status", "instance", instance)
		}
	}()

	if err := r.tryToReconcile(ctx, &instance, jc); err != nil {
		r.log.Error(err, "error during reconcilation", "instance", instance)
		instance.Status.Value = err.Error()
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}
	instance.Status.Value = helper.StatusSuccess

	reqLogger.V(2).Info("Reconciling JenkinsAuthorizationRole has been finished")
	return
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v2v1alpha1.JenkinsAuthorizationRoleMapping,
	jc jenkins.ClientInterface) error {

	for _, rl := range instance.Spec.Roles {
		if err := jc.AssignRole(instance.Spec.RoleType, rl, instance.Spec.Group); err != nil {
			return errors.Wrap(err, "unable to assign role")
		}
	}

	updateNeeded, err := helper.TryToDelete(instance, finalizerName, makeDeletionFunc(instance, jc))
	if err != nil {
		return errors.Wrap(err, "unable to delete instance")
	}

	if updateNeeded {
		if err := r.client.Update(ctx, instance); err != nil {
			return errors.Wrap(err, "unable to update instance")
		}
	}

	return nil
}

func makeDeletionFunc(instance *v2v1alpha1.JenkinsAuthorizationRoleMapping,
	jc jenkins.ClientInterface) func() error {
	return func() error {
		for _, rl := range instance.Spec.Roles {
			if err := jc.UnAssignRole(instance.Spec.RoleType, rl, instance.Spec.Group); err != nil {
				return errors.Wrap(err, "unable to unassign role")
			}
		}

		return nil
	}
}
