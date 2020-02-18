package jenkins

import (
	"context"
	"fmt"
	pipev1alpha1 "github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	v2v1alpha1 "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsClient "github.com/epmd-edp/jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/helper"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/controller/jenkins_job/chain"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/service/platform"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/consts"
	"github.com/epmd-edp/jenkins-operator/v2/pkg/util/finalizer"
	plutil "github.com/epmd-edp/jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_jenkins_job")
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
	return &ReconcileJenkinsJob{
		client: client,
		scheme: scheme,
		ps:     ps,
	}
}

func addKnownTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&pipev1alpha1.Stage{},
		&pipev1alpha1.StageList{},
		&pipev1alpha1.CDPipeline{},
		&pipev1alpha1.CDPipelineList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("jenkins-job-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*v2v1alpha1.JenkinsJob)
			no := e.ObjectNew.(*v2v1alpha1.JenkinsJob)
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
	err = c.Watch(&source.Kind{Type: &v2v1alpha1.JenkinsJob{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileJenkins implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileJenkinsJob{}

const jenkinsJobFinalizerName = "jenkinsjob.finalizer.name"

// ReconcileJenkinsJob reconciles a Jenkins object
type ReconcileJenkinsJob struct {
	client client.Client
	scheme *runtime.Scheme
	ps     platform.PlatformService
}

func (r *ReconcileJenkinsJob) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rlog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	rlog.V(2).Info("reconciling JenkinsJob has been started")

	i := &v2v1alpha1.JenkinsJob{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if result, err := r.tryToDeleteJob(i); result != nil || err != nil {
		return *result, err
	}

	if err := r.setOwners(i); err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while setting owner reference")
	}

	c, err := r.canJenkinsJobBeHandled(i)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while checking availability of creating jenkins job")
	}
	if !c {
		rlog.V(2).Info("jenkins folder for stages is not ready yet")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	ch, err := chain.CreateDefChain(r.scheme, &r.client)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has occurred while selecting chain")
	}

	if err := ch.ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}

	rlog.V(2).Info("reconciling JenkinsJob has been finished")
	return reconcile.Result{}, nil
}

func (r ReconcileJenkinsJob) initGoJenkinsClient(jj v2v1alpha1.JenkinsJob) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, jj.Name, jj.Namespace, jj.Spec.OwnerName, jj.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jj.Name)
	}
	log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, r.ps)
}

func (r ReconcileJenkinsJob) tryToDeleteJob(jj *v2v1alpha1.JenkinsJob) (*reconcile.Result, error) {
	if jj.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName) {
			jj.ObjectMeta.Finalizers = append(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)
			if err := r.client.Update(context.TODO(), jj); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return nil, nil
	}

	if err := r.deleteJob(jj); err != nil {
		return &reconcile.Result{}, err
	}

	if err := r.deleteProject(jj); err != nil {
		return &reconcile.Result{}, err
	}

	jj.ObjectMeta.Finalizers = finalizer.RemoveString(jj.ObjectMeta.Finalizers, jenkinsJobFinalizerName)
	if err := r.client.Update(context.TODO(), jj); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r ReconcileJenkinsJob) deleteJob(jj *v2v1alpha1.JenkinsJob) error {
	jc, err := r.initGoJenkinsClient(*jj)
	if err != nil {
		return errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	j := r.getJobName(jj)
	if _, err := jc.GoJenkins.DeleteJob(j); err != nil {
		if err.Error() == "404" {
			log.V(2).Info("job/folder doesn't exist. skip deleting", "name", j)
			return nil
		}
		return err
	}
	return nil
}

func (r ReconcileJenkinsJob) getJobName(jj *v2v1alpha1.JenkinsJob) string {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		return fmt.Sprintf("%v-cd-pipeline/job/%v", *jj.Spec.JenkinsFolder, jj.Spec.Job.Name)
	}
	return jj.Spec.Job.Name
}

func (r ReconcileJenkinsJob) deleteProject(jj *v2v1alpha1.JenkinsJob) error {
	d, err := r.ps.GetConfigMapData(jj.Namespace, "edp-config")
	if err != nil {
		return err
	}

	s, err := plutil.GetStageInstanceOwner(r.client, *jj)
	if err != nil {
		return err
	}

	pn := fmt.Sprintf("%v-%v", d["edp_name"], s.Name)
	if err := r.ps.DeleteProject(pn); err != nil {
		if k8serrors.IsNotFound(err) {
			log.V(2).Info("project doesn't exist. skip deleting", "name", pn)
			return nil
		}
		return errors.Wrapf(err, "couldn't delete project %v", pn)
	}
	return nil
}

func (r *ReconcileJenkinsJob) setOwners(jj *v2v1alpha1.JenkinsJob) error {
	if err := r.tryToSetStageOwnerRef(jj); err != nil {
		return err
	}
	if err := r.client.Update(context.TODO(), jj); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", jj.Name)
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetJenkinsOwnerRef(jj *v2v1alpha1.JenkinsJob) error {
	if ow := plutil.GetOwnerReference(consts.JenkinsKind, jj.GetOwnerReferences()); ow != nil {
		log.V(2).Info("jenkins owner ref already exists", "jenkins job", jj.Name)
		return nil
	}

	j, err := plutil.GetFirstJenkinsInstance(r.client, jj.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(j, jj, r.scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetStageOwnerRef(jj *v2v1alpha1.JenkinsJob) error {
	if ow := plutil.GetOwnerReference(consts.StageKind, jj.GetOwnerReferences()); ow != nil {
		log.V(2).Info("stage ref already exists", "jenkins job", jj.Name)
		return nil
	}

	s, err := plutil.GetStageInstance(r.client, *jj.Spec.StageName, jj.Namespace)
	if err != nil {
		return err
	}

	if err := controllerutil.SetControllerReference(s, jj, r.scheme); err != nil {
		return errors.Wrap(err, "couldn't set stage owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) tryToSetJenkinsFolderOwnerRef(jj *v2v1alpha1.JenkinsJob) error {
	if jj.Spec.JenkinsFolder == nil || *jj.Spec.JenkinsFolder == "" {
		log.V(2).Info("skip setting jenkins folder reference", "jenkins job", jj.Name)
		return nil
	}

	if ow := plutil.GetOwnerReference(consts.JenkinsFolderKind, jj.GetOwnerReferences()); ow != nil {
		log.V(2).Info("jenkins folder ref already exists", "jenkins job", jj.Name)
		return nil
	}

	jf, err := plutil.GetJenkinsFolderInstance(r.client, *jj.Spec.JenkinsFolder, jj.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(jf, jj, r.scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins folder owner ref")
	}
	return nil
}

func (r *ReconcileJenkinsJob) canJenkinsJobBeHandled(jj *v2v1alpha1.JenkinsJob) (bool, error) {
	if jj.Spec.JenkinsFolder != nil && *jj.Spec.JenkinsFolder != "" {
		jfn := fmt.Sprintf("%v-%v", *jj.Spec.JenkinsFolder, "cd-pipeline")
		jf, err := plutil.GetJenkinsFolderInstance(r.client, jfn, jj.Namespace)
		if err != nil {
			return false, err
		}
		log.V(2).Info("create job in Jenkins folder", "name", jfn, "status folder", jf.Status.Available)
		return jf.Status.Available, nil
	}
	log.V(2).Info("create job in Jenkins root folder", "name", jj.Name)
	return true, nil
}
