package jenkinsserviceaccount

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	logNamespaceKey = "Request.Namespace"
	logNameKey      = "Request.Name"
	requeueAfter    = 60 * time.Second
)

func NewReconcileJenkinsServiceAccount(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkinsServiceAccount {
	return &ReconcileJenkinsServiceAccount{
		client:   k8sClient,
		scheme:   scheme,
		platform: ps,
		log:      log.WithName("jenkins-service-account"),
	}
}

type ReconcileJenkinsServiceAccount struct {
	client   client.Client
	scheme   *runtime.Scheme
	platform platform.PlatformService
	log      logr.Logger
}

func (r *ReconcileJenkinsServiceAccount) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsServiceAccount)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsServiceAccount)
			if !ok {
				return false
			}

			return oldObject.Status == newObject.Status
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsServiceAccount{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed JenkinsServiceAccount controller: %w", err)
	}

	return nil
}

func (r *ReconcileJenkinsServiceAccount) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues(logNamespaceKey, request.Namespace, logNameKey, request.Name)
	log.Info("Reconciling JenkinsServiceAccounts")

	instance := &jenkinsApi.JenkinsServiceAccount{}

	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get JenkinsServiceAccount: %w", err)
	}

	jenkinsInstance, err := r.getOrCreateInstanceOwner(ctx, instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to get owner for %v: %w", instance.Name, err)
	}

	if jenkinsInstance == nil {
		log.Info("Couldn't find Jenkins Service Account owner instance")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	jc, err := jenkinsClient.InitJenkinsClient(jenkinsInstance, r.platform)
	if err != nil {
		log.Info("Failed to init Jenkins REST client")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to init Jenkins REST client: %w", err)
	}

	if jc == nil {
		log.V(1).Info("Jenkins returns nil client")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if err := jc.CreateUser(instance); err != nil {
		log.Info("Failed to create user in Jenkins")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to create user: %w", err)
	}

	if err := r.updateCreatedStatus(ctx, instance, true); err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	if err := r.updateAvailableStatus(ctx, instance, true); err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func (r *ReconcileJenkinsServiceAccount) getJenkinsInstance(ctx context.Context, namespace string) (*jenkinsApi.Jenkins, error) {
	list := &jenkinsApi.JenkinsList{}

	if err := r.client.List(ctx, list, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, fmt.Errorf("failed to get Jenkins instance in namespace %v: %w", namespace, err)
	}

	if len(list.Items) == 0 {
		return nil, nil
	}

	return &list.Items[0], nil
}

func (*ReconcileJenkinsServiceAccount) setOwnerReference(owner *jenkinsApi.Jenkins, jenkinsScript *jenkinsApi.JenkinsServiceAccount) *jenkinsApi.JenkinsServiceAccount {
	jenkinsScriptOwners := jenkinsScript.GetOwnerReferences()
	newOwnerRef := metav1.OwnerReference{
		APIVersion:         owner.APIVersion,
		Kind:               owner.Kind,
		Name:               owner.Name,
		UID:                owner.UID,
		BlockOwnerDeletion: helper.NewTrue(),
		Controller:         helper.NewTrue(),
	}

	jenkinsScriptOwners = append(jenkinsScriptOwners, newOwnerRef)
	jenkinsScript.SetOwnerReferences(jenkinsScriptOwners)

	return jenkinsScript
}

func (r *ReconcileJenkinsServiceAccount) updateAvailableStatus(ctx context.Context, instance *jenkinsApi.JenkinsServiceAccount, value bool) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).WithName("status_update")

	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

		if err := r.client.Status().Update(ctx, instance); err != nil {
			if err := r.client.Update(ctx, instance); err != nil {
				return fmt.Errorf("failed to update availability status to %v: %w", value, err)
			}
		}

		log.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}

	return nil
}

func (r *ReconcileJenkinsServiceAccount) updateCreatedStatus(ctx context.Context, instance *jenkinsApi.JenkinsServiceAccount, value bool) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).WithName("status_update")

	if instance.Status.Created != value {
		instance.Status.Created = value
		instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

		if err := r.client.Status().Update(ctx, instance); err != nil {
			if err := r.client.Update(ctx, instance); err != nil {
				return fmt.Errorf("failed to update created status to %v: %w", value, err)
			}
		}

		log.Info(fmt.Sprintf("Created status has been updated to '%v'", value))
	}

	return nil
}

func (r *ReconcileJenkinsServiceAccount) getOrCreateInstanceOwner(ctx context.Context, jenkinsServiceAccount *jenkinsApi.JenkinsServiceAccount) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues(logNamespaceKey, jenkinsServiceAccount.Namespace, logNameKey, jenkinsServiceAccount.Name)

	owner := r.getOwnerByCr(jenkinsServiceAccount)
	if owner != nil {
		return r.getInstanceByName(ctx, jenkinsServiceAccount.Namespace, owner.Name)
	}

	if jenkinsServiceAccount.Spec.OwnerName != "" {
		return r.getInstanceByOwnerFromSpec(ctx, jenkinsServiceAccount)
	}

	jenkinsInstance, err := r.getJenkinsInstance(ctx, jenkinsServiceAccount.Namespace)
	if err != nil {
		return nil, err
	}

	if jenkinsInstance == nil {
		return nil, nil
	}

	jenkinsServiceAccount = r.setOwnerReference(jenkinsInstance, jenkinsServiceAccount)

	log.Info(fmt.Sprintf("jenkinsServiceAccount.GetOwnerReferences() - %v", jenkinsServiceAccount.GetOwnerReferences()))

	if err := r.client.Update(ctx, jenkinsServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to set owner reference for %v: %w", jenkinsServiceAccount.Name, err)
	}

	return jenkinsInstance, nil
}

func (*ReconcileJenkinsServiceAccount) getOwnerByCr(jenkinsScript *jenkinsApi.JenkinsServiceAccount) *metav1.OwnerReference {
	owners := jenkinsScript.GetOwnerReferences()

	for _, owner := range owners {
		if owner.Kind == "Jenkins" {
			return &owner
		}
	}

	return nil
}

func (r *ReconcileJenkinsServiceAccount) getInstanceByOwnerFromSpec(ctx context.Context, jenkinsUser *jenkinsApi.JenkinsServiceAccount) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues(logNamespaceKey, jenkinsUser.Namespace, logNameKey, jenkinsUser.Name)
	nsn := types.NamespacedName{
		Namespace: jenkinsUser.Namespace,
		Name:      jenkinsUser.Spec.OwnerName,
	}
	jenkinsInstance := &jenkinsApi.Jenkins{}

	if err := r.client.Get(ctx, nsn, jenkinsInstance); err != nil {
		log.Info(fmt.Sprintf("Failed to get owner CR %v", jenkinsUser.Spec.OwnerName))

		return nil, nil
	}

	jenkinsUser = r.setOwnerReference(jenkinsInstance, jenkinsUser)

	if err := r.client.Update(ctx, jenkinsUser); err != nil {
		return nil, fmt.Errorf("failed to set owner name from spec for %v: %w", jenkinsUser.Name, err)
	}

	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsServiceAccount) getInstanceByName(ctx context.Context, namespace, name string) (*jenkinsApi.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	instance := &jenkinsApi.Jenkins{}

	if err := r.client.Get(ctx, nsn, instance); err != nil {
		return nil, fmt.Errorf("failed to get instance by owner %v: %w", name, err)
	}

	return instance, nil
}
