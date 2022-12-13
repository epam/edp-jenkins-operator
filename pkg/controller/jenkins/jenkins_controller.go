package jenkins

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
)

const (
	StatusInstall          = "installing"
	StatusFailed           = "failed"
	StatusCreated          = "created"
	StatusConfiguring      = "configuring"
	StatusConfigured       = "configured"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
	logNamespaceKey        = "Request.Namespace"
	logNameKey             = "Request.Name"
	requeueAfter           = 60 * time.Second
)

func NewReconcileJenkins(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkins {
	return &ReconcileJenkins{
		client:  k8sClient,
		scheme:  scheme,
		service: jenkins.NewJenkinsService(ps, k8sClient, scheme),
		log:     log.WithName("jenkins"),
	}
}

type ReconcileJenkins struct {
	client  client.Client
	scheme  *runtime.Scheme
	service jenkins.JenkinsService
	log     logr.Logger
}

func (r *ReconcileJenkins) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObject, ok := e.ObjectOld.(*jenkinsApi.Jenkins)
			if !ok {
				return false
			}

			newObject, ok := e.ObjectNew.(*jenkinsApi.Jenkins)
			if !ok {
				return false
			}

			return !reflect.DeepEqual(oldObject.Status, newObject.Status)
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.Jenkins{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

//nolint:funlen,cyclop // TODO: remove nolint and fix issues.
func (r *ReconcileJenkins) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues(logNamespaceKey, request.Namespace, logNameKey, request.Name)
	log.Info("Reconciling has been started")

	instance := &jenkinsApi.Jenkins{}

	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to Get Jenkins instance: %w", err)
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info("Installation has been started")

		if err := r.updateStatus(ctx, instance, StatusInstall); err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info("Installation has finished")

		if err := r.updateStatus(ctx, instance, StatusCreated); err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	// Create Admin password secret
	if err := r.service.CreateAdminPassword(instance); err != nil {
		log.Error(err, "Admin password secret creation has failed")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to create admin password secret creation: %w", err)
	}

	dcIsReady, err := r.service.IsDeploymentReady(instance)
	if err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to check if Deployment configs are ready: %w", err)
	}

	if !dcIsReady {
		log.Info("Deployment configs is not ready for configuration yet")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		log.Info("Configuration has started")

		if updErr := r.updateStatus(ctx, instance, StatusConfiguring); updErr != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, updErr
		}
	}

	instance, isFinished, err := r.service.Configure(instance)
	if err != nil {
		log.Error(err, "Configuration has failed")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
			fmt.Errorf("failed to finish configuration: %w", err)
	}

	if !isFinished {
		log.Info("Configuration is not finished")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		log.Info("Configuration has finished")

		if updErr := r.updateStatus(ctx, instance, StatusConfigured); updErr != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, updErr
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")

		if updErr := r.updateStatus(ctx, instance, StatusIntegrationStart); updErr != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, updErr
		}
	}

	instance, upd, err := r.service.ExposeConfiguration(instance)
	if err != nil {
		log.Error(err, "Expose configuration has failed")

		return reconcile.Result{
			RequeueAfter: helper.DefaultRequeueTime * time.Second,
		}, fmt.Errorf("failed to expose configuration: %w", err)
	}

	if upd {
		if err = r.updateInstanceStatus(ctx, instance); err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second},
				fmt.Errorf("failed to update instance status: %w", err)
		}
	}

	instance, isFinished, err = r.service.Integration(instance)
	if err != nil {
		log.Error(err, "Integration has failed")

		return reconcile.Result{
			RequeueAfter: helper.DefaultRequeueTime * time.Second,
		}, fmt.Errorf("integration failed: %w", err)
	}

	if !isFinished {
		log.Info("Integration is not finished")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusIntegrationStart {
		log.Info("Configuration has been finished", instance.Namespace, instance.Name)

		if err = r.updateStatus(ctx, instance, StatusReady); err != nil {
			log.Info("Couldn't update status")

			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
		}
	}

	if err = r.updateAvailableStatus(ctx, instance, true); err != nil {
		log.Info("Failed to update availability status")

		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{RequeueAfter: requeueAfter}, nil
}

func (r *ReconcileJenkins) updateStatus(ctx context.Context, instance *jenkinsApi.Jenkins, newStatus string) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).
		WithName("status_update")
	currentStatus := instance.Status.Status

	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

	if err := r.client.Status().Update(ctx, instance); err != nil {
		if err := r.client.Update(ctx, instance); err != nil {
			return fmt.Errorf("failed to update status from '%v' to '%v': %w", currentStatus, newStatus, err)
		}
	}

	log.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))

	return nil
}

func (r *ReconcileJenkins) updateAvailableStatus(ctx context.Context, instance *jenkinsApi.Jenkins, value bool) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).
		WithName("status_update")

	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

		if err := r.client.Status().Update(ctx, instance); err != nil {
			if err := r.client.Update(ctx, instance); err != nil {
				return fmt.Errorf("failed to update availability status to %v: %w", value, err)
			}
		}
	}

	log.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))

	return nil
}

func (r *ReconcileJenkins) updateInstanceStatus(ctx context.Context, instance *jenkinsApi.Jenkins) error {
	log := r.log.WithValues(logNamespaceKey, instance.Namespace, logNameKey, instance.Name).
		WithName("status_update")

	instance.Status.LastTimeUpdated = metav1.NewTime(time.Now())

	if err := r.client.Status().Update(ctx, instance); err != nil {
		if err := r.client.Update(ctx, instance); err != nil {
			return fmt.Errorf("failed to update instance status: %w", err)
		}
	}

	log.Info("Instance status has been updated")

	return nil
}
