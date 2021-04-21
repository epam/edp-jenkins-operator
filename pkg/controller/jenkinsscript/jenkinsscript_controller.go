package jenkinsscript

import (
	"context"
	"fmt"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"

	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	errorsf "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewReconcileJenkinsScript(client client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkinsScript {
	return &ReconcileJenkinsScript{
		client:   client,
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
			oldObject := e.ObjectOld.(*jenkinsApi.JenkinsScript)
			newObject := e.ObjectNew.(*jenkinsApi.JenkinsScript)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsScript{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileJenkinsScript) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling JenkinsScript")

	instance := &jenkinsApi.JenkinsScript{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	jenkinsInstance, err := r.getOrCreateInstanceOwner(ctx, instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Failed to get owner for %v", instance.Name)
	}
	if jenkinsInstance == nil {
		log.Info("Couldn't find Jenkins Script owner instance")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Executed == true {
		return reconcile.Result{}, nil
	}

	jc, err := jenkinsClient.InitJenkinsClient(jenkinsInstance, r.platform)
	if err != nil {
		log.Info("Failed to init Jenkins REST client")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}
	if jc == nil {
		log.V(1).Info("Jenkins returns nil client")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	cm, err := r.platform.GetConfigMapData(instance.Namespace, instance.Spec.SourceCmName)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	err = jc.RunScript(cm["context"])
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	log.V(1).Info("Script has been executed successfully")

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	err = r.updateExecutedStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed to update executed status")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	log.Info("Reconciling has been finished")
	return reconcile.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *ReconcileJenkinsScript) getInstanceByName(ctx context.Context, namespace string, name string) (*jenkinsApi.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	instance := &jenkinsApi.Jenkins{}
	err := r.client.Get(ctx, nsn, instance)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to get instance by owner %v", name)
	}
	return instance, nil
}

func (r *ReconcileJenkinsScript) getInstanceByOwnerFromSpec(ctx context.Context, jenkinsScript *jenkinsApi.JenkinsScript) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues("Request.Namespace", jenkinsScript.Namespace, "Request.Name", jenkinsScript.Name)
	nsn := types.NamespacedName{
		Namespace: jenkinsScript.Namespace,
		Name:      *jenkinsScript.Spec.OwnerName,
	}
	jenkinsInstance := &jenkinsApi.Jenkins{}
	err := r.client.Get(ctx, nsn, jenkinsInstance)
	if err != nil {
		log.Info(fmt.Sprintf("Failed to get owner CR %v", *jenkinsScript.Spec.OwnerName))
		return nil, nil
	}
	jenkinsScript = r.setOwnerReference(jenkinsInstance, jenkinsScript)
	err = r.client.Update(ctx, jenkinsScript)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to set owner name from spec for %v", jenkinsScript.Name)
	}
	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsScript) getOwnerByCr(jenkinsScript *jenkinsApi.JenkinsScript) *metav1.OwnerReference {
	owners := jenkinsScript.GetOwnerReferences()
	if len(owners) == 0 {
		return nil
	}
	for _, owner := range owners {
		if owner.Kind == "Jenkins" {
			return &owner
		}
	}
	return nil
}

func (r *ReconcileJenkinsScript) getOrCreateInstanceOwner(ctx context.Context, jenkinsScript *jenkinsApi.JenkinsScript) (*jenkinsApi.Jenkins, error) {
	log := r.log.WithValues("Request.Namespace", jenkinsScript.Namespace, "Request.Name", jenkinsScript.Name)
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
	err = r.client.Update(ctx, jenkinsScript)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to set owner reference for %v", jenkinsScript.Name)
	}
	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsScript) getJenkinsInstance(ctx context.Context, namespace string) (*jenkinsApi.Jenkins, error) {
	list := &jenkinsApi.JenkinsList{}
	err := r.client.List(ctx, list, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, errorsf.Wrapf(err, "Couldn't get Jenkins instance in namespace %v", namespace)
	}
	if len(list.Items) == 0 {
		return nil, nil
	}
	jenkins := list.Items[0]
	nsn := types.NamespacedName{
		Namespace: jenkins.Namespace,
		Name:      jenkins.Name,
	}
	jenkinsInstance := &jenkinsApi.Jenkins{}
	err = r.client.Get(ctx, nsn, jenkinsInstance)
	return jenkinsInstance, err
}

func (r *ReconcileJenkinsScript) setOwnerReference(owner *jenkinsApi.Jenkins, jenkinsScript *jenkinsApi.JenkinsScript) *jenkinsApi.JenkinsScript {
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

func (r ReconcileJenkinsScript) updateAvailableStatus(ctx context.Context, instance *jenkinsApi.JenkinsScript, value bool) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			err := r.client.Update(ctx, instance)
			if err != nil {
				return errorsf.Wrapf(err, "Couldn't update availability status to %v", value)
			}
		}
		log.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}
	return nil
}

func (r ReconcileJenkinsScript) updateExecutedStatus(ctx context.Context, instance *jenkinsApi.JenkinsScript, value bool) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Executed != value {
		instance.Status.Executed = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			err := r.client.Update(ctx, instance)
			if err != nil {
				return errorsf.Wrapf(err, "Couldn't update executed status to %v", value)
			}
		}
		log.Info(fmt.Sprintf("Executed status has been updated to '%v'", value))
	}
	return nil
}
