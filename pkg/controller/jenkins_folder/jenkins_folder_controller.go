package jenkins

import (
	"context"
	"fmt"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/job_provision"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	edpv1alpha1 "github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"

	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_jenkins_folder")
var SchemeGroupVersion = schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1alpha1"}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Jenkins Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	scheme := mgr.GetScheme()
	addKnownTypes(scheme)
	client := mgr.GetClient()
	pt := helper.GetPlatformTypeEnv()
	ps, _ := platform.NewPlatformService(pt, scheme, &client)
	return &ReconcileJenkinsFolder{
		client: client,
		scheme: scheme,
		handler: job_provision.JobProvision{
			Client: client,
		},
		Platform: ps,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1alpha1.Codebase{},
		&v1alpha1.CodebaseList{},
		&v1alpha1.GitServer{},
		&v1alpha1.GitServerList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkins-folder-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*v2v1alpha1.JenkinsFolder)
			no := e.ObjectNew.(*v2v1alpha1.JenkinsFolder)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			return false
		},
	}

	// Watch for changes to primary resource Jenkins
	err = c.Watch(&source.Kind{Type: &v2v1alpha1.JenkinsFolder{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileJenkins implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkinsFolder{}

// ReconcileJenkinsFolder reconciles a Jenkins object
type ReconcileJenkinsFolder struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	handler  job_provision.JobProvision
	Platform platform.PlatformService
}

func (r *ReconcileJenkinsFolder) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsFolder has been started")

	i := &v2v1alpha1.JenkinsFolder{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	j, err := r.getJenkinsInstanceOwner(*i)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", i.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)

	c, err := r.getCodebaseInstanceOwner(*i)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while getting owner codebase for jenkins folder %v", i.Name)
	}
	log.Info("codebase instance has been received", "name", c.Name)

	jc, err := jenkinsClient.InitGoJenkinsClient(j, r.Platform)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	if i.Spec.JobName == nil {
		reqLogger.V(2).Info("job name is empty. create empty folder in Jenkins")
		if err := jc.CreateFolder(c.Name); err != nil {
			return reconcile.Result{}, err
		}
		log.Info("folder has been created in Jenkins", "name", c.Name)

		if err := r.setStatus(i, true, consts.StatusFinished); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", i.Name)
		}
		reqLogger.V(2).Info("Reconciling JenkinsFolder has been finished")
		return reconcile.Result{}, nil
	}

	if err := r.handler.TriggerBuildJobProvision(*jc, c, i); err != nil {
		if err := r.setStatus(i, false, consts.StatusFailed); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", i.Name)
		}
		return reconcile.Result{}, err
	}

	if err := r.setStatus(i, true, consts.StatusFinished); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "an error has been occurred while updating %v JobFolder status", i.Name)
	}
	reqLogger.V(2).Info("Reconciling JenkinsFolder has been finished")
	return reconcile.Result{}, nil
}

func (r ReconcileJenkinsFolder) getCodebaseInstanceOwner(jf v2v1alpha1.JenkinsFolder) (*edpv1alpha1.Codebase, error) {
	log.V(2).Info("start getting codebase owner name", "jenkins folder", jf.Name)
	if ow := r.getOwnerReference(consts.CodebaseKind, jf); ow != nil {
		log.V(2).Info("trying to fetch codebase owner from reference", "codebase name", ow.Name)
		return r.getCodebaseInstance(ow.Name, jf.Namespace)
	}
	if jf.Spec.CodebaseName != nil {
		log.V(2).Info("trying to fetch codebase owner from spec", "codebase name", jf.Spec.CodebaseName)
		return r.getCodebaseInstance(*jf.Spec.CodebaseName, jf.Namespace)
	}
	return nil, fmt.Errorf("couldn't find codebase owner for jenkins folder %v", jf.Name)
}

func (r ReconcileJenkinsFolder) getJenkinsInstanceOwner(jf v2v1alpha1.JenkinsFolder) (*v2v1alpha1.Jenkins, error) {
	log.V(2).Info("start getting jenkins owner name", "jenkins folder", jf.Name)
	if ow := r.getOwnerReference(consts.JenkinsKind, jf); ow != nil {
		log.V(2).Info("trying to fetch jenkins owner from reference", "jenkins name", ow.Name)
		return r.getJenkinsInstance(ow.Name, jf.Namespace)
	}
	if jf.Spec.OwnerName != nil {
		log.Info("trying to fetch jenkins owner from spec", "jenkins name", jf.Spec.OwnerName)
		return r.getJenkinsInstance(*jf.Spec.OwnerName, jf.Namespace)
	}
	log.V(2).Info("trying to fetch first jenkins instance", "namespace", jf.Namespace)
	j, err := r.getFirstJenkinsInstance(jf.Namespace)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r ReconcileJenkinsFolder) getOwnerReference(ownerKind string, jf v2v1alpha1.JenkinsFolder) *metav1.OwnerReference {
	log.Info("finding owner for jenkins folder", "kind", ownerKind)
	ors := jf.GetOwnerReferences()
	if len(ors) == 0 {
		return nil
	}
	for _, o := range ors {
		if o.Kind == ownerKind {
			return &o
		}
	}
	return nil
}

func (r ReconcileJenkinsFolder) getCodebaseInstance(name, namespace string) (*edpv1alpha1.Codebase, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &edpv1alpha1.Codebase{}
	if err := r.client.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return instance, nil
}

func (r ReconcileJenkinsFolder) getJenkinsInstance(name, namespace string) (*v2v1alpha1.Jenkins, error) {
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	instance := &v2v1alpha1.Jenkins{}
	if err := r.client.Get(context.TODO(), nsn, instance); err != nil {
		return nil, errors.Wrapf(err, "failed to get instance by owner %v", name)
	}
	return instance, nil
}

func (r ReconcileJenkinsFolder) getFirstJenkinsInstance(namespace string) (*v2v1alpha1.Jenkins, error) {
	list := &v2v1alpha1.JenkinsList{}
	err := r.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, list)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get Jenkins instances in namespace %v", namespace)
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("at least one Jenkins instance should be accessible")
	}
	j := list.Items[0]
	return r.getJenkinsInstance(j.Name, j.Namespace)
}

func (r ReconcileJenkinsFolder) setStatus(jf *v2v1alpha1.JenkinsFolder, available bool, status string) error {
	jf.Status = v2v1alpha1.JenkinsFolderStatus{
		Available:                      available,
		LastTimeUpdated:                time.Time{},
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return r.updateStatus(jf)
}

func (r ReconcileJenkinsFolder) updateStatus(jf *v2v1alpha1.JenkinsFolder) error {
	if err := r.client.Status().Update(context.TODO(), jf); err != nil {
		if err := r.client.Update(context.TODO(), jf); err != nil {
			return err
		}
	}
	return nil
}
