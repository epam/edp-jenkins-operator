package jenkinsscript

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

func NewReconcileJenkinsScript(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkinsScript {
	return &ReconcileJenkinsScript{
		client:   k8sClient,
		scheme:   scheme,
		platform: ps,
		log:      log.WithName("jenkins-script"),
	}
}

type ReconcileJenkinsScript struct {
	client   client.Client
	scheme   *runtime.Scheme
	platform platform.PlatformService
	log      logr.Logger
}

func (r *ReconcileJenkinsScript) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsScript)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsScript)
			if !ok {
				return false
			}

			return oldObject.Status == newObject.Status
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsScript{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

//nolint:funlen,cyclop // TODO: remove nolint.
func (r *ReconcileJenkinsScript) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues(logNamespaceKey, request.Namespace, logNameKey, request.Name)
	log.Info("Reconciling JenkinsScript")

	instance := &jenkinsApi.JenkinsScript{}

	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get client: %w", err)
	}

	jenkinsInstance, err := r.getOrCreateInstanceOwner(ctx, instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, fmt.Errorf("failed to get owner for %v: %w", instance.Name, err)
	}

	if jenkinsInstance == nil {
		log.Info("Couldn't find Jenkins Script owner instance")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Executed {
		log.Info("Script already finished")

		return reconcile.Result{}, nil
	}

	log.Info("Applying the script")

	jc, err := jenkinsClient.InitJenkinsClient(jenkinsInstance, r.platform)
	if err != nil {
		log.Info("Failed to init Jenkins REST client")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, fmt.Errorf("failed to init jenkins client for %v: %w", instance.Name, err)
	}

	if jc == nil {
		log.V(1).Info("Jenkins returns nil client")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	cm, err := r.platform.GetConfigMapData(instance.Namespace, instance.Spec.SourceCmName)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, fmt.Errorf("failed to get config map for %v: %w", instance.Name, err)
	}

	if err := jc.RunScript(cm["context"]); err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, fmt.Errorf("failed to RunScript: %w", err)
	}

	log.V(1).Info("Script has been executed successfully")

	if err := r.updateAvailableStatus(ctx, instance, true); err != nil {
		log.Info("Failed to update availability status")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	if err := r.updateExecutedStatus(ctx, instance, true); err != nil {
		log.Info("Failed to update executed status")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func (r *ReconcileJenkinsScript) getInstanceByName(ctx context.Context, namespace, name string) (*jenkinsApi.Jenkins, error) {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	instance := &jenkinsApi.Jenkins{}

	if err := r.client.Get(ctx, namespacedName, instance); err != nil {
		return nil, fmt.Errorf("failed to get instance by owner %v: %w", name, err)
	}

	return instance, nil
}

func (r *ReconcileJenkinsScript) getInstanceByOwnerFromSpec(ctx context.Context, jenkinsScript *jenkinsApi.JenkinsScript) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues(logNamespaceKey, jenkinsScript.Namespace, logNameKey, jenkinsScript.Name)
	nsn := types.NamespacedName{
		Namespace: jenkinsScript.Namespace,
		Name:      *jenkinsScript.Spec.OwnerName,
	}
	jenkinsInstance := &jenkinsApi.Jenkins{}

	if err := r.client.Get(ctx, nsn, jenkinsInstance); err != nil {
		log.Info(fmt.Sprintf("Failed to get owner CR %v", *jenkinsScript.Spec.OwnerName))

		return nil, nil
	}

	jenkinsScript = r.setOwnerReference(jenkinsInstance, jenkinsScript)

	if err := r.client.Update(ctx, jenkinsScript); err != nil {
		return nil, fmt.Errorf("failed to set owner name from spec for %v: %w", jenkinsScript.Name, err)
	}

	return jenkinsInstance, nil
}

func (*ReconcileJenkinsScript) getOwnerByCr(jenkinsScript *jenkinsApi.JenkinsScript) *metav1.OwnerReference {
	owners := jenkinsScript.GetOwnerReferences()

	for _, owner := range owners {
		if owner.Kind == "Jenkins" {
			return &owner
		}
	}

	return nil
}

func (r *ReconcileJenkinsScript) getOrCreateInstanceOwner(ctx context.Context, jenkinsScript *jenkinsApi.JenkinsScript) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues(logNamespaceKey, jenkinsScript.Namespace, logNameKey, jenkinsScript.Name)

	owner := r.getOwnerByCr(jenkinsScript)
	if owner != nil {
		return r.getInstanceByName(ctx, jenkinsScript.Namespace, owner.Name)
	}

	if jenkinsScript.Spec.OwnerName != nil {
		return r.getInstanceByOwnerFromSpec(ctx, jenkinsScript)
	}

	jenkinsInstance, err := r.getJenkinsInstance(ctx, jenkinsScript.Namespace)
	if err != nil {
		return nil, err
	}

	if jenkinsInstance == nil {
		return nil, nil
	}

	jenkinsScript = r.setOwnerReference(jenkinsInstance, jenkinsScript)
	log.Info(fmt.Sprintf("jenkinsScript.GetOwnerReferences() - %v", jenkinsScript.GetOwnerReferences()))

	if err = r.client.Update(ctx, jenkinsScript); err != nil {
		return nil, fmt.Errorf("failed to set owner reference for %v: %w", jenkinsScript.Name, err)
	}

	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsScript) getJenkinsInstance(ctx context.Context, namespace string) (*jenkinsApi.Jenkins, error) {
	list := &jenkinsApi.JenkinsList{}

	if err := r.client.List(ctx, list, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, fmt.Errorf("failed to get Jenkins instance in namespace %v: %w", namespace, err)
	}

	if len(list.Items) == 0 {
		return nil, nil
	}

	return &list.Items[0], nil
}

func (*ReconcileJenkinsScript) setOwnerReference(owner *jenkinsApi.Jenkins, jenkinsScript *jenkinsApi.JenkinsScript) *jenkinsApi.JenkinsScript {
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

func (r *ReconcileJenkinsScript) updateAvailableStatus(ctx context.Context, instance *jenkinsApi.JenkinsScript, value bool) error {
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

func (r *ReconcileJenkinsScript) updateExecutedStatus(ctx context.Context, instance *jenkinsApi.JenkinsScript, value bool) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).WithName("status_update")

	if instance.Status.Executed != value {
		instance.Status.Executed = value
		instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

		if err := r.client.Status().Update(ctx, instance); err != nil {
			if err := r.client.Update(ctx, instance); err != nil {
				return fmt.Errorf("failed to update executed status to %v: %w", value, err)
			}
		}

		log.Info(fmt.Sprintf("Executed status has been updated to '%v'", value))
	}

	return nil
}
