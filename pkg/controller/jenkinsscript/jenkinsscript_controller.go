package jenkinsscript

import (
	"context"
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"

	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	errorsf "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_jenkinsscript")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new JenkinsScript Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	client := mgr.GetClient()

	platformType := helper.GetPlatformTypeEnv()
	platformService, _ := platform.NewPlatformService(platformType, scheme, &client)

	return &ReconcileJenkinsScript{
		client:   client,
		scheme:   scheme,
		platform: platformService,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkinsscript-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject := e.ObjectOld.(*v2v1alpha1.JenkinsScript)
			newObject := e.ObjectNew.(*v2v1alpha1.JenkinsScript)
			if oldObject.Status != newObject.Status {
				return false
			}
			return true
		},
	}

	// Watch for changes to primary resource JenkinsScript
	err = c.Watch(&source.Kind{Type: &v2v1alpha1.JenkinsScript{}}, &handler.EnqueueRequestForObject{}, p)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileJenkinsScript implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkinsScript{}

// ReconcileJenkinsScript reconciles a JenkinsScript object
type ReconcileJenkinsScript struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	platform platform.PlatformService
}

// Reconcile reads that state of the cluster for a JenkinsScript object and makes changes based on the state read
// and what is in the JenkinsScript.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJenkinsScript) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling JenkinsScript")

	// Fetch the JenkinsScript instance
	instance := &v2v1alpha1.JenkinsScript{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	jenkinsInstance, err := r.getOrCreateInstanceOwner(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errorsf.Wrapf(err, "Failed to get owner for %v", instance.Name)
	}
	if jenkinsInstance == nil {
		reqLogger.Info("Couldn't find Jenkins Script owner instance")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Executed == true {
		return reconcile.Result{}, nil
	}

	jc, err := jenkinsClient.InitJenkinsClient(jenkinsInstance, r.platform)
	if err != nil {
		reqLogger.Info("Failed to init Jenkins REST client")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}
	if jc == nil {
		reqLogger.V(1).Info("Jenkins returns nil client")
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

	reqLogger.V(1).Info("Script has been executed successfully")

	err = r.updateAvailableStatus(instance, true)
	if err != nil {
		reqLogger.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	err = r.updateExecutedStatus(instance, true)
	if err != nil {
		reqLogger.Info("Failed to update executed status")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	reqLogger.Info("Reconciling has been finished")
	return reconcile.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *ReconcileJenkinsScript) getInstanceByName(namespace string, name string) (*v2v1alpha1.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	instance := &v2v1alpha1.Jenkins{}
	err := r.client.Get(context.TODO(), nsn, instance)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to get instance by owner %v", name)
	}
	return instance, nil
}

func (r *ReconcileJenkinsScript) getInstanceByOwnerFromSpec(jenkinsScript *v2v1alpha1.JenkinsScript) (*v2v1alpha1.Jenkins, error) {
	reqLogger := log.WithValues("Request.Namespace", jenkinsScript.Namespace, "Request.Name", jenkinsScript.Name)
	nsn := types.NamespacedName{
		Namespace: jenkinsScript.Namespace,
		Name:      *jenkinsScript.Spec.OwnerName,
	}
	jenkinsInstance := &v2v1alpha1.Jenkins{}
	err := r.client.Get(context.TODO(), nsn, jenkinsInstance)
	if err != nil {
		reqLogger.Info(fmt.Sprintf("Failed to get owner CR %v", *jenkinsScript.Spec.OwnerName))
		return nil, nil
	}
	jenkinsScript = r.setOwnerReference(jenkinsInstance, jenkinsScript)
	err = r.client.Update(context.TODO(), jenkinsScript)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to set owner name from spec for %v", jenkinsScript.Name)
	}
	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsScript) getOwnerByCr(jenkinsScript *v2v1alpha1.JenkinsScript) *metav1.OwnerReference {
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

func (r *ReconcileJenkinsScript) getOrCreateInstanceOwner(jenkinsScript *v2v1alpha1.JenkinsScript) (*v2v1alpha1.Jenkins, error) {
	reqLogger := log.WithValues("Request.Namespace", jenkinsScript.Namespace, "Request.Name", jenkinsScript.Name)
	owner := r.getOwnerByCr(jenkinsScript)
	if owner != nil {
		return r.getInstanceByName(jenkinsScript.Namespace, owner.Name)
	}

	if jenkinsScript.Spec.OwnerName != nil {
		return r.getInstanceByOwnerFromSpec(jenkinsScript)
	}

	jenkinsInstance, err := r.getJenkinsInstance(jenkinsScript.Namespace)
	if err != nil {
		return nil, err
	}
	if jenkinsInstance == nil {
		return nil, nil
	}
	jenkinsScript = r.setOwnerReference(jenkinsInstance, jenkinsScript)
	reqLogger.Info(fmt.Sprintf("jenkinsScript.GetOwnerReferences() - %v", jenkinsScript.GetOwnerReferences()))
	err = r.client.Update(context.TODO(), jenkinsScript)
	if err != nil {
		return nil, errorsf.Wrapf(err, "Failed to set owner reference for %v", jenkinsScript.Name)
	}
	return jenkinsInstance, nil
}

func (r *ReconcileJenkinsScript) getJenkinsInstance(namespace string) (*v2v1alpha1.Jenkins, error) {
	list := &v2v1alpha1.JenkinsList{}
	err := r.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, list)
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
	jenkinsInstance := &v2v1alpha1.Jenkins{}
	err = r.client.Get(context.TODO(), nsn, jenkinsInstance)
	return jenkinsInstance, err
}

func (r *ReconcileJenkinsScript) setOwnerReference(owner *v2v1alpha1.Jenkins, jenkinsScript *v2v1alpha1.JenkinsScript) *v2v1alpha1.JenkinsScript {
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

func (r ReconcileJenkinsScript) updateAvailableStatus(instance *v2v1alpha1.JenkinsScript, value bool) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return errorsf.Wrapf(err, "Couldn't update availability status to %v", value)
			}
		}
		reqLogger.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	}
	return nil
}

func (r ReconcileJenkinsScript) updateExecutedStatus(instance *v2v1alpha1.JenkinsScript, value bool) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Executed != value {
		instance.Status.Executed = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return errorsf.Wrapf(err, "Couldn't update executed status to %v", value)
			}
		}
		reqLogger.Info(fmt.Sprintf("Executed status has been updated to '%v'", value))
	}
	return nil
}
