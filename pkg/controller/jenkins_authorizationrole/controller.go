package jenkins_authorizationrole

import (
	"context"
	"reflect"
	"time"

	v2v1alpha1 "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
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

const finalizerName = "jenkinsauthrole.jenkins.finalizer.name"

type Reconcile struct {
	client               client.Client
	log                  logr.Logger
	jenkinsClientFactory jenkins.ClientFactory
}

func NewReconciler(k8sCl client.Client, logf logr.Logger,
	ps platform.PlatformService) *Reconcile {

	return &Reconcile{
		client:               k8sCl,
		log:                  logf.WithName("controller_jenkins_authorizationrole"),
		jenkinsClientFactory: jenkins.MakeClientBuilder(ps, k8sCl),
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*v2v1alpha1.JenkinsAuthorizationRole)
			no := e.ObjectNew.(*v2v1alpha1.JenkinsAuthorizationRole)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}

			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v2v1alpha1.JenkinsAuthorizationRole{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resError error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsAuthorizationRole has been started")

	var instance v2v1alpha1.JenkinsAuthorizationRole
	if err := r.client.Get(context.TODO(), request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return
		}

		return reconcile.Result{}, errors.Wrap(err, "unable to get JenkinsAuthorizationRole instance")
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

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *v2v1alpha1.JenkinsAuthorizationRole,
	jc jenkins.ClientInterface) error {
	if err := jc.AddRole(instance.Spec.RoleType, instance.Spec.Name, instance.Spec.Pattern, instance.Spec.Permissions); err != nil {
		return errors.Wrap(err, "unable to add role")
	}

	if err := r.tryToDelete(ctx, jc, instance); err != nil {
		return errors.Wrap(err, "unable to delete instance")
	}

	return nil
}

func (r *Reconcile) tryToDelete(ctx context.Context, jc jenkins.ClientInterface,
	instance *v2v1alpha1.JenkinsAuthorizationRole) error {
	if instance.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(instance.ObjectMeta.Finalizers, finalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
			if err := r.client.Update(ctx, instance); err != nil {
				return errors.Wrap(err, "unable to update instance finalizer")
			}
		}

		return nil
	}

	if err := jc.RemoveRoles(instance.Spec.RoleType, []string{instance.Spec.Name}); err != nil {
		return errors.Wrap(err, "unable to delete role")
	}

	instance.ObjectMeta.Finalizers = finalizer.RemoveString(instance.ObjectMeta.Finalizers, finalizerName)
	if err := r.client.Update(ctx, instance); err != nil {
		return errors.Wrap(err, "unable to remove finalizer from instance")
	}

	return nil
}
