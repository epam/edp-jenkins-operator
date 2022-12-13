package sharedLibrary

import (
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
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

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.JenkinsSharedLibrary{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func specUpdated(e event.UpdateEvent) bool {
	oldObject, ok := e.ObjectOld.(*jenkinsApi.JenkinsSharedLibrary)
	if !ok {
		return false
	}

	newObject, ok := e.ObjectNew.(*jenkinsApi.JenkinsSharedLibrary)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oldObject.Spec, newObject.Spec) ||
		(oldObject.GetDeletionTimestamp().IsZero() && !newObject.GetDeletionTimestamp().IsZero())
}

func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconciling JenkinsSharedLibrary has been started")

	var (
		instance jenkinsApi.JenkinsSharedLibrary
		result   reconcile.Result
	)

	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8serrors.IsNotFound(err) {
			reqLogger.Info("instance not found")

			return result, nil
		}

		return result, fmt.Errorf("failed to get JenkinsSharedLibrary instance: %w", err)
	}

	if err := r.tryToReconcile(ctx, &instance); err != nil {
		r.log.Error(err, "error during reconciliation", "instance", instance)
		instance.Status.Value = err.Error()
		result = reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}
	} else {
		instance.Status.Value = helper.StatusSuccess
	}

	if err := r.client.Status().Update(ctx, &instance); err != nil {
		return result, fmt.Errorf("failed to update status: %w", err)
	}

	reqLogger.V(2).Info("Reconciling JenkinsSharedLibrary has been finished")

	return result, nil
}

func (r *Reconcile) tryToReconcile(ctx context.Context, instance *jenkinsApi.JenkinsSharedLibrary) error {
	if instance.Status.Value == helper.StatusSuccess && instance.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	rootJenkins, err := plutil.GetJenkinsInstanceOwner(r.client, instance.Name, instance.Namespace,
		instance.Spec.OwnerName, instance.GetOwnerReferences())
	if err != nil {
		return fmt.Errorf("failed to get owner for jenkins folder %s: %w", instance.Name, err)
	}

	if rootJenkins.Status.Status != jenkins.StatusReady {
		return errors.New("root jenkins is not ready")
	}

	sharedLibraries, err := r.prepareSharedLibraries(ctx, instance, rootJenkins.Spec.SharedLibraries)
	if err != nil {
		return fmt.Errorf("failed to prepare shared libraries: %w", err)
	}

	if err = r.createLibrariesScript(rootJenkins, sharedLibraries); err != nil {
		return fmt.Errorf("failed to create libraries script: %w", err)
	}

	needToUpdate, err := helper.TryToDelete(instance, finalizer, func() error {
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if !needToUpdate {
		return nil
	}

	if err := r.client.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

func (r *Reconcile) prepareSharedLibraries(ctx context.Context,
	instance *jenkinsApi.JenkinsSharedLibrary,
	rootLibraries []jenkinsApi.JenkinsSharedLibraries,
) ([]jenkinsApi.JenkinsSharedLibraries, error) {
	var libList jenkinsApi.JenkinsSharedLibraryList

	if err := r.client.List(ctx, &libList, &client.ListOptions{Namespace: instance.Namespace}); err != nil {
		return nil, fmt.Errorf("failed to list jenkins shared libraries: %w", err)
	}

	for i := 0; i < len(libList.Items); i++ {
		libOwnerName, instanceOwnerName := "", ""

		if libList.Items[i].Spec.OwnerName != nil {
			libOwnerName = *libList.Items[i].Spec.OwnerName
		}

		if instance.Spec.OwnerName != nil {
			instanceOwnerName = *instance.Spec.OwnerName
		}

		if libOwnerName == instanceOwnerName && libList.Items[i].Name != instance.Name {
			credentialID := libList.Items[i].Spec.CredentialID

			rootLibraries = append(rootLibraries, jenkinsApi.JenkinsSharedLibraries{
				Name:         libList.Items[i].Spec.Name,
				CredentialID: &credentialID,
				Tag:          libList.Items[i].Spec.Tag,
				URL:          libList.Items[i].Spec.URL,
			})
		}
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		rootLibraries = append(rootLibraries, jenkinsApi.JenkinsSharedLibraries{
			Name:         instance.Spec.Name,
			CredentialID: &instance.Spec.CredentialID,
			Tag:          instance.Spec.Tag,
			URL:          instance.Spec.URL,
		})
	}

	return rootLibraries, nil
}

func (r *Reconcile) createLibrariesScript(
	rootJenkins *jenkinsApi.Jenkins,
	sharedLibraries []jenkinsApi.JenkinsSharedLibraries,
) error {
	buffer, err := platformHelper.ParseTemplate(
		&platformHelper.JenkinsScriptData{JenkinsSharedLibraries: sharedLibraries},
		path.Join(r.templatesPath, jenkinsService.SharedLibrariesTemplateName),
		jenkinsService.SharedLibrariesTemplateName)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	isUpdated, err := r.platformService.CreateConfigMapWithUpdate(rootJenkins, scriptConfigMapName,
		map[string]string{consts.JenkinsDefaultScriptConfigMapKey: buffer.String()})
	if err != nil {
		return fmt.Errorf("failed to create config map: %w", err)
	}

	if _, err = r.platformService.CreateJenkinsScript(rootJenkins.Namespace, scriptConfigMapName, isUpdated); err != nil {
		return fmt.Errorf("failed to create jenkins script: %w", err)
	}

	return nil
}
