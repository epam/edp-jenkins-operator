package jenkins

import (
	"context"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"strings"
	"time"

	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"

	"reflect"

	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	jf_handler "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const jenkinsFolderJenkinsFinalizerName = "jenkinsfolder.jenkins.finalizer.name"

type ReconcileJenkinsFolder struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Platform platform.PlatformService
	Log      logr.Logger
}

func (r *ReconcileJenkinsFolder) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oo := e.ObjectOld.(*jenkinsApi.JenkinsFolder)
			no := e.ObjectNew.(*jenkinsApi.JenkinsFolder)
			if !reflect.DeepEqual(oo.Spec, no.Spec) {
				return true
			}
			if no.DeletionTimestamp != nil {
				return true
			}
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsFolder{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileJenkinsFolder) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling JenkinsFolder has been started")

	i := &jenkinsApi.JenkinsFolder{}
	if err := r.Client.Get(ctx, request.NamespacedName, i); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	jc, err := r.initGoJenkinsClient(*i)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "an error has been occurred while creating gojenkins client")
	}

	result, err := r.tryToDeleteJenkinsFolder(ctx, *jc, i)
	if err != nil || result != nil {
		return *result, err
	}

	h, err := r.createChain(i.Spec.Job != nil)
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := h.ServeRequest(i); err != nil {
		return reconcile.Result{}, err
	}
	log.V(2).Info("Reconciling JenkinsFolder has been finished")
	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsFolder) createChain(flag bool) (jf_handler.JenkinsFolderHandler, error) {
	if flag {
		return chain.CreateTriggerBuildProvisionChain(r.Scheme, &r.Client)
	}
	return chain.CreateCDPipelineFolderChain(r.Scheme, &r.Client)
}

func (r ReconcileJenkinsFolder) initGoJenkinsClient(jf jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.Client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, errors.Wrapf(err, "an error has been occurred while getting owner jenkins for jenkins folder %v", jf.Name)
	}
	r.Log.Info("Jenkins instance has been received", "name", j.Name)
	return jenkinsClient.InitGoJenkinsClient(j, r.Platform)
}

func (r ReconcileJenkinsFolder) setStatus(ctx context.Context, jf *jenkinsApi.JenkinsFolder, available bool, status string) error {
	jf.Status = jenkinsApi.JenkinsFolderStatus{
		Available:                      available,
		LastTimeUpdated:                time.Time{},
		Status:                         status,
		JenkinsJobProvisionBuildNumber: jf.Status.JenkinsJobProvisionBuildNumber,
	}
	return r.updateStatus(ctx, jf)
}

func (r ReconcileJenkinsFolder) updateStatus(ctx context.Context, jf *jenkinsApi.JenkinsFolder) error {
	if err := r.Client.Status().Update(ctx, jf); err != nil {
		if err := r.Client.Update(ctx, jf); err != nil {
			return err
		}
	}
	return nil
}

func (r ReconcileJenkinsFolder) tryToDeleteJenkinsFolder(ctx context.Context, jc jenkinsClient.JenkinsClient, jf *jenkinsApi.JenkinsFolder) (*reconcile.Result, error) {
	if jf.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jf.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName) {
			jf.ObjectMeta.Finalizers = append(jf.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName)
			if err := r.Client.Update(ctx, jf); err != nil {
				return &reconcile.Result{}, err
			}
		}
		return nil, nil
	}

	fn := r.getJenkinsFolderName(jf)
	if _, err := jc.GoJenkins.DeleteJob(fn); err != nil {
		if err.Error() != "404" {
			return &reconcile.Result{}, err
		}
		r.Log.V(2).Info("404 code error when Jenkins job was deleted earlier during reconciliation", "jenkins folder", jf.Name)
	}

	jf.ObjectMeta.Finalizers = finalizer.RemoveString(jf.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName)
	if err := r.Client.Update(ctx, jf); err != nil {
		return &reconcile.Result{}, err
	}
	return &reconcile.Result{}, nil
}

func (r ReconcileJenkinsFolder) getJenkinsFolderName(jf *jenkinsApi.JenkinsFolder) string {
	if jf.Spec.Job == nil {
		return jf.Name
	}
	return strings.Replace(jf.Name, "-codebase", "", -1)
}

func (r *ReconcileJenkinsFolder) tryToSetJenkinsOwnerRef(ctx context.Context, jf *jenkinsApi.JenkinsFolder) error {
	ow := plutil.GetOwnerReference(consts.JenkinsKind, jf.GetOwnerReferences())
	if ow != nil {
		r.Log.V(2).Info("jenkins owner ref already exists", "jenkins folder", jf.Name)
		return nil
	}

	j, err := plutil.GetFirstJenkinsInstance(r.Client, jf.Namespace)
	if err != nil {
		return err
	}

	if err := plutil.SetControllerReference(j, jf, r.Scheme, false); err != nil {
		return errors.Wrap(err, "couldn't set jenkins owner ref")
	}

	if err := r.Client.Update(ctx, jf); err != nil {
		return errors.Wrapf(err, "an error has been occurred while updating jenkins job %v", jf.Name)
	}
	return nil
}
