package main

import (
	"flag"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1alpha1"
	gerritApi "github.com/epam/edp-gerrit-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"

	jenkinsdeployment "github.com/epam/edp-jenkins-operator/v2/pkg/controller/cdstagejenkinsdeployment"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/helper"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins"
	jenkinsFolder "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_folder"
	jenkinsJob "github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkins_job"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkinsscript"
	"github.com/epam/edp-jenkins-operator/v2/pkg/controller/jenkinsserviceaccount"
	jenkinsService "github.com/epam/edp-jenkins-operator/v2/pkg/service/jenkins"
	"github.com/epam/edp-jenkins-operator/v2/pkg/service/platform"
	clusterUtil "github.com/epam/edp-jenkins-operator/v2/pkg/util"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const jenkinsOperatorLock = "edp-jenkins-operator-lock"

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(jenkinsApi.AddToScheme(scheme))

	utilruntime.Must(cdPipeApi.AddToScheme(scheme))

	utilruntime.Must(codebaseApi.AddToScheme(scheme))

	utilruntime.Must(edpCompApi.AddToScheme(scheme))

	utilruntime.Must(gerritApi.AddToScheme(scheme))

	utilruntime.Must(keycloakApi.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", clusterUtil.RunningInCluster(),
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	mode, err := clusterUtil.GetDebugMode()
	if err != nil {
		setupLog.Error(err, "unable to get debug mode value")
		os.Exit(1)
	}

	opts := zap.Options{
		Development: mode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ns, err := clusterUtil.GetWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get watch namespace")
		os.Exit(1)
	}

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       jenkinsOperatorLock,
		MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
			return apiutil.NewDynamicRESTMapper(cfg)
		},
		Namespace: ns,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctrlLog := ctrl.Log.WithName("controllers")
	client := mgr.GetClient()
	if err = (&jenkinsdeployment.ReconcileCDStageJenkinsDeployment{
		Client: client,
		Scheme: mgr.GetScheme(),
		Log:    ctrlLog.WithName("cd-stage-jenkins-deployment"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "cd-stage-jenkins-deployment")
		os.Exit(1)
	}

	ps, err := platform.NewPlatformService(helper.GetPlatformTypeEnv(), mgr.GetScheme(), &client)
	if err != nil {
		setupLog.Error(err, "unable to create platform service")
		os.Exit(1)
	}

	js := jenkinsService.NewJenkinsService(ps, client, mgr.GetScheme())
	if err = (&jenkins.ReconcileJenkins{
		Client:  client,
		Scheme:  mgr.GetScheme(),
		Service: js,
		Log:     ctrlLog.WithName("jenkins"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jenkins")
		os.Exit(1)
	}

	if err = (&jenkinsFolder.ReconcileJenkinsFolder{
		Client:   client,
		Scheme:   mgr.GetScheme(),
		Platform: ps,
		Log:      ctrlLog.WithName("jenkins-folder"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jenkins-folder")
		os.Exit(1)
	}

	if err = (&jenkinsJob.ReconcileJenkinsJob{
		Client:   client,
		Scheme:   mgr.GetScheme(),
		Platform: ps,
		Log:      ctrlLog.WithName("jenkins-job"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jenkins-job")
		os.Exit(1)
	}

	if err = (&jenkinsscript.ReconcileJenkinsScript{
		Client:   client,
		Scheme:   mgr.GetScheme(),
		Platform: ps,
		Log:      ctrlLog.WithName("jenkins-script"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jenkins-script")
		os.Exit(1)
	}

	if err = (&jenkinsserviceaccount.ReconcileJenkinsServiceAccount{
		Client:   client,
		Scheme:   mgr.GetScheme(),
		Platform: ps,
		Log:      ctrlLog.WithName("jenkins-service-account"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "jenkins-service-account")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
