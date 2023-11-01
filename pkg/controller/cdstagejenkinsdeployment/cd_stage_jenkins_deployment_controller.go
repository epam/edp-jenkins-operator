package cdstagejenkinsdeployment

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-codebase-operator/v2/pkg/util"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/chain"
	cdStageJenkinshelper "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	ps "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	"github.com/epam/edp-jenkins-operator/v2/pkg/util/consts"
)

const (
	baseTimeoutDuration = 500 * time.Millisecond
)

func NewReconcileCDStageJenkinsDeployment(k8sClient client.Client, scheme *runtime.Scheme, log logr.Logger) *ReconcileCDStageJenkinsDeployment {
	return &ReconcileCDStageJenkinsDeployment{
		client: k8sClient,
		scheme: scheme,
		log:    log.WithName("cd-stage-jenkins-deployment"),
	}
}

type ReconcileCDStageJenkinsDeployment struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

func (r *ReconcileCDStageJenkinsDeployment) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&jenkinsApi.CDStageJenkinsDeployment{}, builder.WithPredicates(p)).
		Complete(r); err != nil {
		return fmt.Errorf("failed to create new managed controller: %w", err)
	}

	return nil
}

func (r *ReconcileCDStageJenkinsDeployment) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	log.Info("reconciling has been started")

	cdStageJenkinsDeployment := &jenkinsApi.CDStageJenkinsDeployment{}
	if err := r.client.Get(ctx, request.NamespacedName, cdStageJenkinsDeployment); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("instance not found")

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get CDStageJenkinsDeployment: %w", err)
	}

	defer func() {
		if err := r.updateStatus(ctx, cdStageJenkinsDeployment); err != nil {
			log.Error(err, "error during status updating")
		}
	}()

	if err := r.setOwnerReference(cdStageJenkinsDeployment); err != nil {
		wrappedError := fmt.Errorf("failed to set owner ref for %v CDStageJenkinsDeployment: %w",
			cdStageJenkinsDeployment.Name, err)

		cdStageJenkinsDeployment.SetFailedStatus(wrappedError)

		return reconcile.Result{}, wrappedError
	}

	env, err := helper.GetPlatformTypeEnv()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to GetPlatformTypeEnv: %w", err)
	}

	platform, err := ps.NewPlatformService(env, r.scheme, r.client)
	if err != nil {
		wrappedError := fmt.Errorf("failed to create platform service: %w", err)

		cdStageJenkinsDeployment.SetFailedStatus(wrappedError)

		return reconcile.Result{}, wrappedError
	}

	if err := chain.CreateDefChain(r.client, platform).ServeRequest(cdStageJenkinsDeployment); err != nil {
		cdStageJenkinsDeployment.SetFailedStatus(err)
		p := r.setReconcilationPeriod(cdStageJenkinsDeployment)

		return reconcile.Result{RequeueAfter: p}, nil
	}

	log.Info("Reconciling has been finished")

	return reconcile.Result{}, nil
}

func (r *ReconcileCDStageJenkinsDeployment) setReconcilationPeriod(jd *jenkinsApi.CDStageJenkinsDeployment) time.Duration {
	timeout := util.GetTimeout(jd.Status.FailureCount, baseTimeoutDuration)

	r.log.Info("wait for next reconciliation", "next reconciliation in", timeout)
	jd.Status.FailureCount++

	return timeout
}

func (r *ReconcileCDStageJenkinsDeployment) updateStatus(ctx context.Context, jenkinsDeployment *jenkinsApi.CDStageJenkinsDeployment) error {
	if err := r.client.Status().Update(ctx, jenkinsDeployment); err != nil {
		if err := r.client.Update(ctx, jenkinsDeployment); err != nil {
			return fmt.Errorf("failed to Update jenkinsDeployment: %w", err)
		}
	}

	return nil
}

func (r *ReconcileCDStageJenkinsDeployment) setOwnerReference(jenkinsDeployment *jenkinsApi.CDStageJenkinsDeployment) error {
	s, err := cdStageJenkinshelper.GetCDStageDeploy(r.client, jenkinsDeployment.Labels[consts.CdStageDeployKey], jenkinsDeployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to GetCDStageDeploy: %w", err)
	}

	if err = controllerutil.SetControllerReference(s, jenkinsDeployment, r.scheme); err != nil {
		return fmt.Errorf("failed to SetControllerReference: %w", err)
	}

	if err = r.client.Update(context.TODO(), jenkinsDeployment); err != nil {
		return fmt.Errorf("failed to Update CDStageJenkinsDeployment: %w", err)
	}

	return nil
}
