package sharedLibrary

import (
	"context"
	"path"
	"reflect"
	"time"

	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins"
	jenkinsService "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
	plutil "github.com/epam/edp-jenkins-operator/v2/pkg/util/platform"
)

const (
	scriptConfigMapName = "jenkins-shared-libraries-controller"
	finalizer           = "shared_library.jenkins.finalizer.name"
)

type Reconcile struct {
	client          client.Client
	log             logr.Logger
	platformService platform.PlatformService
	templatesPath   string
}

func NewReconcile(k8sCl client.Client, logf logr.Logger, ps platform.PlatformService, templatesPath string) *Reconcile {
	return &Reconcile{
		client:          k8sCl,
		log:             logf.WithName("controller_shared_library"),
		platformService: ps,
		templatesPath:   templatesPath,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: specUpdated,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.JenkinsSharedLibrary{}, builder.WithPredicates(p)).
		Complete(r)
}

func specUpdated(e event.UpdateEvent) bool {
	oo := e.ObjectOld.(*v1alpha1.JenkinsSharedLibrary)
	no := e.ObjectNew.(*v1alpha1.JenkinsSharedLibrary)

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

func (r Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsSharedLibrary has been started")

	var (
		instance v1alpha1.JenkinsSharedLibrary
		result   reconcile.Result
	)
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.Info("instance not found")
			return result, nil
		}

		return result, errors.Wrap(err, "unable to get JenkinsSharedLibrary instance")
	}

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		r.log.Error(err, "error during reconciliation", "instance", instance)
		instance.Status.Value = err.Error()
		result = reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}
	} else {
		instance.Status.Value = helper.StatusSuccess
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		return result, errors.Wrap(err, "unable to update status")
	}

	reqLogger.V(2).Info("Reconciling JenkinsSharedLibrary has been finished")
	return result, nil
}

func (r Reconcile) tryToReconcile(ctx context.Context, instance *v1alpha1.JenkinsSharedLibrary) error {
	if instance.Status.Value == helper.StatusSuccess && instance.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	rootJenkins, err := plutil.GetJenkinsInstanceOwner(r.client, instance.Name, instance.Namespace,
		instance.Spec.OwnerName, instance.GetOwnerReferences())
	if err != nil {
		return errors.Wrapf(err,
			"an error has been occurred while getting owner jenkins for jenkins folder %s", instance.Name)
	}

	if rootJenkins.Status.Status != jenkins.StatusReady {
		return errors.New("root jenkins is not ready")
	}

	sharedLibraries, err := r.prepareSharedLibraries(ctx, instance, rootJenkins.Spec.SharedLibraries)
	if err != nil {
		return errors.Wrap(err, "unable to prepare shared libraries")
	}

	if err := r.createLibrariesScript(rootJenkins, sharedLibraries); err != nil {
		return errors.Wrap(err, "unable to create libraries script")
	}

	needToUpdate, err := helper.TryToDelete(instance, finalizer, func() error {
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "unable to delete resource")
	}

	if needToUpdate {
		if err := r.client.Update(ctx, instance); err != nil {
			return errors.Wrap(err, "unable to update instance")
		}
	}

	return nil
}

func (r Reconcile) prepareSharedLibraries(ctx context.Context,
	instance *v1alpha1.JenkinsSharedLibrary,
	rootLibraries []v1alpha1.JenkinsSharedLibraries) ([]v1alpha1.JenkinsSharedLibraries, error) {

	var libList v1alpha1.JenkinsSharedLibraryList

	if err := r.client.List(ctx, &libList, &client.ListOptions{Namespace: instance.Namespace}); err != nil {
		return nil, errors.Wrap(err, "unable to list jenkins shared libraries")
	}

	for _, lib := range libList.Items {
		libOwnerName, instanceOwnerName := "", ""
		if lib.Spec.OwnerName != nil {
			libOwnerName = *lib.Spec.OwnerName
		}
		if instance.Spec.OwnerName != nil {
			instanceOwnerName = *instance.Spec.OwnerName
		}

		if libOwnerName == instanceOwnerName && lib.Name != instance.Name {
			rootLibraries = append(rootLibraries, v1alpha1.JenkinsSharedLibraries{
				Name:         lib.Spec.Name,
				CredentialID: &lib.Spec.CredentialID,
				Tag:          lib.Spec.Tag,
				URL:          lib.Spec.URL,
			})
		}
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		rootLibraries = append(rootLibraries, v1alpha1.JenkinsSharedLibraries{
			Name:         instance.Spec.Name,
			CredentialID: &instance.Spec.CredentialID,
			Tag:          instance.Spec.Tag,
			URL:          instance.Spec.URL,
		})
	}

	return rootLibraries, nil
}

func (r Reconcile) createLibrariesScript(rootJenkins *v1alpha1.Jenkins,
	sharedLibraries []v1alpha1.JenkinsSharedLibraries) error {
	buffer, err := platformHelper.ParseTemplate(
		platformHelper.JenkinsScriptData{JenkinsSharedLibraries: sharedLibraries},
		path.Join(r.templatesPath, jenkinsService.SharedLibrariesTemplateName),
		jenkinsService.SharedLibrariesTemplateName)
	if err != nil {
		return errors.Wrap(err, "unable to parse template")
	}

	isUpdated, err := r.platformService.CreateConfigMapWithUpdate(rootJenkins, scriptConfigMapName,
		map[string]string{consts.JenkinsDefaultScriptConfigMapKey: buffer.String()})
	if err != nil {
		return errors.Wrap(err, "unable to create config map")
	}

	_, err = r.platformService.CreateJenkinsScript(rootJenkins.Namespace, scriptConfigMapName, isUpdated)
	if err != nil {
		return errors.Wrap(err, "unable to create jenkins script")
	}

	return nil
}
