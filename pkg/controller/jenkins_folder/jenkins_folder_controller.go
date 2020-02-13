package jenkins

import (
	"context"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder/chain"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/finalizer"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
	"time"

	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"

	"github.com/epmd-edp/codebase-operator/v2/pkg/apis/edp/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	jf_handler "github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
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
		client:   client,
		scheme:   scheme,
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
			if no.DeletionTimestamp != nil {
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

const JenkinsFolderJenkinsFinalizerName = "jenkinsfolder.jenkins.finalizer.name"

// ReconcileJenkinsFolder reconciles a Jenkins object
type ReconcileJenkinsFolder struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
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

	jc, err := r.initGoJenkinsClient(*i)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	result, err := r.tryToDeleteJenkinsFolder(*jc, i)
	if err != nil || result != nil {
		return *result, err
	}

	h, err := r.createChain(i.Spec.JobName != nil)
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := h.ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.V(2).Info("Reconciling JenkinsFolder has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsFolder) createChain(flag bool) (jf_handler.JenkinsFolderHandler, error) {
	if flag {
		return chain.CreateTriggerBuildProvisionChain(r.scheme, &r.client)
	}
	return chain.CreateCDPipelineFolderChain(r.scheme, &r.client)
}

func (r ReconcileJenkinsFolder) initGoJenkinsClient(jf v2v1alpha1.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, r.Platform)
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

func (r ReconcileJenkinsFolder) tryToDeleteJenkinsFolder(jc jenkinsClient.JenkinsClient, jf *v2v1alpha1.JenkinsFolder) (*reconcile.Result, error) {
	if jf.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jf.ObjectMeta.Finalizers, JenkinsFolderJenkinsFinalizerName) {
			jf.ObjectMeta.Finalizers = append(jf.ObjectMeta.Finalizers, JenkinsFolderJenkinsFinalizerName)
			if err := r.client.Update(context.TODO(), jf); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return nil, nil
	}

	fn := r.getJenkinsFolderName(jf)
	if _, err := jc.GoJenkins.DeleteJob(fn); err != nil {
		return &reconcile.Result{}, err
	}

	jf.ObjectMeta.Finalizers = finalizer.RemoveString(jf.ObjectMeta.Finalizers, JenkinsFolderJenkinsFinalizerName)
	if err := r.client.Update(context.TODO(), jf); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r ReconcileJenkinsFolder) getJenkinsFolderName(jf *v2v1alpha1.JenkinsFolder) string {
	if jf.Spec.JobName == nil {
		return jf.Name
	}
	return strings.Replace(jf.Name, "-codebase", "", -1)
}

func (r *ReconcileJenkinsFolder) tryToSetJenkinsOwnerRef(jf *v2v1alpha1.JenkinsFolder) error {
	ow := plutil.GetOwnerReference(consts.JenkinsKind, jf.GetOwnerReferences())
	if ow != nil {
		log.V(2).Info("jenkins owner ref already exists", "jenkins folder", jf.Name)
		return nil
	}

	j, err := plutil.GetFirstJenkinsInstance(r.client, jf.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(j, jf, r.scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins owner ref")
	}

	if err := r.client.Update(context.TODO(), jf); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", jf.Name)
	}
	return nil
}
