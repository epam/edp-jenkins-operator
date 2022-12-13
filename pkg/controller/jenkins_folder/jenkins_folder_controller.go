package jenkins

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	jenkinsClient "github.com/epam/edp-jenkins-operator/v2/pkg/client/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain"
	jfHandler "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder/chain/handler"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/finalizer"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

const jenkinsFolderJenkinsFinalizerName = "jenkinsfolder.jenkins.finalizer.name"

func NewReconcileJenkinsFolder(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
	ps platform.PlatformService,
) *ReconcileJenkinsFolder {
	return &ReconcileJenkinsFolder{
		client:   k8sClient,
		scheme:   scheme,
		platform: ps,
		log:      log.WithName("jenkins-folder"),
	}
}

type ReconcileJenkinsFolder struct {
	client   client.Client
	scheme   *runtime.Scheme
	platform platform.PlatformService
	log      logr.Logger
}

func (r *ReconcileJenkinsFolder) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsFolder)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsFolder)
			if !ok {
				return false
			}

			if !reflect.DeepEqual(oldObject.Spec, newObject.Spec) {
				return true
			}

			if newObject.DeletionTimestamp != nil {
				return true
			}

			return false
		},
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsFolder{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func (r *ReconcileJenkinsFolder) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.V(2).Info("Reconciling JenkinsFolder has been started")

	jenkinsFolder := &jenkinsApi.JenkinsFolder{}

	if err := r.client.Get(ctx, request.NamespacedName, jenkinsFolder); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to Get JenkinsFolder: %w", err)
	}

	jc, err := r.initGoJenkinsClient(jenkinsFolder)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create gojenkins client: %w", err)
	}

	result, err := r.tryToDeleteJenkinsFolder(ctx, *jc, jenkinsFolder)
	if err != nil || result != nil {
		return *result, err
	}

	h, err := r.createChain(jenkinsFolder.Spec.Job != nil)
	if err != nil {
		return reconcile.Result{}, err
	}

	if err = h.ServeRequest(jenkinsFolder); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ServeRequest: %w", err)
	}

	log.V(2).Info("Reconciling JenkinsFolder has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsFolder) createChain(flag bool) (jfHandler.JenkinsFolderHandler, error) {
	if flag {
		folderHandler, err := chain.CreateTriggerBuildProvisionChain(r.scheme, r.client)
		if err != nil {
			return nil, fmt.Errorf("failed to CreateTriggerBuildProvisionChain: %w", err)
		}

		return folderHandler, nil
	}

	folderHandler, err := chain.CreateCDPipelineFolderChain(r.scheme, r.client)
	if err != nil {
		return nil, fmt.Errorf("failed to CreateCDPipelineFolderChain: %w", err)
	}

	return folderHandler, nil
}

func (r *ReconcileJenkinsFolder) initGoJenkinsClient(jf *jenkinsApi.JenkinsFolder) (*jenkinsClient.JenkinsClient, error) {
	j, err := plutil.GetJenkinsInstanceOwner(r.client, jf.Name, jf.Namespace, jf.Spec.OwnerName, jf.GetOwnerReferences())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner jenkins for jenkins folder %v: %w",
			jf.Name, err)
	}

	r.log.Info("Jenkins instance has been received", "name", j.Name)

	jClient, err := jenkinsClient.InitGoJenkinsClient(j, r.platform)
	if err != nil {
		return nil, fmt.Errorf("failed to InitGoJenkinsClient: %w", err)
	}

	return jClient, nil
}

func (r *ReconcileJenkinsFolder) tryToDeleteJenkinsFolder(
	ctx context.Context,
	jc jenkinsClient.JenkinsClient,
	jenkinsFolder *jenkinsApi.JenkinsFolder,
) (*reconcile.Result, error) {
	if jenkinsFolder.GetDeletionTimestamp().IsZero() {
		if !finalizer.ContainsString(jenkinsFolder.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName) {
			jenkinsFolder.ObjectMeta.Finalizers = append(jenkinsFolder.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName)

			if err := r.client.Update(ctx, jenkinsFolder); err != nil {
				return &reconcile.Result{}, fmt.Errorf("failed to update JenkinsFolder: %w", err)
			}
		}

		return nil, nil
	}

	jenkinsFolderName := r.getJenkinsFolderName(jenkinsFolder)

	if _, err := jc.GoJenkins.DeleteJob(jenkinsFolderName); err != nil {
		if helper.JenkinsIsNotFoundErr(err) {
			return &reconcile.Result{}, fmt.Errorf("failed to delete JenkinsFolder: %w", err)
		}

		r.log.V(2).Info("404 code error when Jenkins job was deleted earlier during reconciliation", "jenkins folder", jenkinsFolder.Name)
	}

	jenkinsFolder.ObjectMeta.Finalizers = finalizer.RemoveString(jenkinsFolder.ObjectMeta.Finalizers, jenkinsFolderJenkinsFinalizerName)

	if err := r.client.Update(ctx, jenkinsFolder); err != nil {
		return &reconcile.Result{}, fmt.Errorf("failed to update JenkinsFolder: %w", err)
	}

	return &reconcile.Result{}, nil
}

func (*ReconcileJenkinsFolder) getJenkinsFolderName(jf *jenkinsApi.JenkinsFolder) string {
	if jf.Spec.Job == nil {
		return jf.Name
	}

	return strings.ReplaceAll(jf.Name, "-codebase", "")
}
