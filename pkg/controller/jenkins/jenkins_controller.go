package jenkins

import (
	"context"
	"fmt"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/go-logr/logr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	"github.com/pkg/errors"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	StatusInstall          = "installing"
	StatusFailed           = "failed"
	StatusCreated          = "created"
	StatusConfiguring      = "configuring"
	StatusConfigured       = "configured"
	StatusIntegrationStart = "integration started"
	StatusReady            = "ready"
)

func NewReconcileJenkins(client client.Client, scheme *runtime.Scheme, log logr.Logger, ps platform.PlatformService) *ReconcileJenkins {
	return &ReconcileJenkins{
		client:  client,
		scheme:  scheme,
		service: jenkins.NewJenkinsService(ps, client, scheme),
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
			oldObject := e.ObjectOld.(*jenkinsApi.Jenkins)
			newObject := e.ObjectNew.(*jenkinsApi.Jenkins)
			return !reflect.DeepEqual(oldObject.Status, newObject.Status)
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.Jenkins{}, builder.WithPredicates(p)).
		Complete(r)
}

func (r *ReconcileJenkins) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("Reconciling has been started")

	instance := &jenkinsApi.Jenkins{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("instance not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Status.Status == "" || instance.Status.Status == StatusFailed {
		log.Info("Installation has been started")
		err = r.updateStatus(ctx, instance, StatusInstall)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusInstall {
		log.Info("Installation has finished")
		err = r.updateStatus(ctx, instance, StatusCreated)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	// Create Admin password secret
	err = r.service.CreateAdminPassword(*instance)
	if err != nil {
		log.Error(err, "Admin password secret creation has failed")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Admin password secret creation has failed")
	}

	if dcIsReady, err := r.service.IsDeploymentReady(*instance); err != nil {
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Checking if Deployment configs is ready has been failed")
	} else if !dcIsReady {
		log.Info("Deployment configs is not ready for configuration yet")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusCreated || instance.Status.Status == "" {
		log.Info("Configuration has started")
		err := r.updateStatus(ctx, instance, StatusConfiguring)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	instance, isFinished, err := r.service.Configure(*instance)
	if err != nil {
		log.Error(err, "Configuration has failed")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Configuration failed")
	} else if !isFinished {
		log.Info("Configuration is not finished")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusConfiguring {
		log.Info("Configuration has finished")
		err = r.updateStatus(ctx, instance, StatusConfigured)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	if instance.Status.Status == StatusConfigured {
		log.Info("Exposing configuration has started")
		err = r.updateStatus(ctx, instance, StatusIntegrationStart)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	instance, upd, err := r.service.ExposeConfiguration(*instance)
	if err != nil {
		log.Error(err, "Expose configuration has failed")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Expose configuration failed")
	}

	if upd {
		err = r.updateInstanceStatus(ctx, instance)
		if err != nil {
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
		}
	}

	instance, isFinished, err = r.service.Integration(*instance)
	if err != nil {
		log.Error(err, "Integration has failed")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, errors.Wrapf(err, "Integration failed")
	} else if !isFinished {
		log.Info("Integration is not finished")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
	}

	if instance.Status.Status == StatusIntegrationStart {
		log.Info("Configuration has been finished", instance.Namespace, instance.Name)
		err = r.updateStatus(ctx, instance, StatusReady)
		if err != nil {
			log.Info("Couldn't update status")
			return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, nil
		}
	}

	err = r.updateAvailableStatus(ctx, instance, true)
	if err != nil {
		log.Info("Failed to update availability status")
		return reconcile.Result{RequeueAfter: helper.DefaultRequeueTime * time.Second}, err
	}

	log.Info("Reconciling has been finished")
	return reconcile.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *ReconcileJenkins) updateStatus(ctx context.Context, instance *jenkinsApi.Jenkins, newStatus string) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	currentStatus := instance.Status.Status
	instance.Status.Status = newStatus
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err := r.client.Update(ctx, instance)
		if err != nil {
			return errors.Wrapf(err, "Couldn't update status from '%v' to '%v'", currentStatus, newStatus)
		}
	}
	log.Info(fmt.Sprintf("Status has been updated to '%v'", newStatus))
	return nil
}

func (r ReconcileJenkins) updateAvailableStatus(ctx context.Context, instance *jenkinsApi.Jenkins, value bool) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	if instance.Status.Available != value {
		instance.Status.Available = value
		instance.Status.LastTimeUpdated = time.Now()
		err := r.client.Status().Update(ctx, instance)
		if err != nil {
			err := r.client.Update(ctx, instance)
			if err != nil {
				return errors.Wrapf(err, "Couldn't update availability status to %v", value)
			}
		}
	}
	log.Info(fmt.Sprintf("Availability status has been updated to '%v'", value))
	return nil
}

func (r ReconcileJenkins) updateInstanceStatus(ctx context.Context, instance *jenkinsApi.Jenkins) error {
	log := r.log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name).WithName("status_update")
	instance.Status.LastTimeUpdated = time.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil {
		err := r.client.Update(ctx, instance)
		if err != nil {
			return errors.Wrapf(err, "Couldn't update instance status")
		}
	}
	log.Info("Instance status has been updated")
	return nil
}
