package main

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/clusterreader"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/config"

	"github.com/fluxcd/pkg/runtime/acl"
	"github.com/fluxcd/pkg/runtime/client"
	runtimeClient "github.com/fluxcd/pkg/runtime/client"
	runtimeCtrl "github.com/fluxcd/pkg/runtime/controller"
	"github.com/fluxcd/pkg/runtime/events"
	feathelper "github.com/fluxcd/pkg/runtime/features"
	"github.com/fluxcd/pkg/runtime/leaderelection"
	"github.com/fluxcd/pkg/runtime/logger"
	"github.com/fluxcd/pkg/runtime/pprof"
	"github.com/fluxcd/pkg/runtime/probes"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	// +kubebuilder:scaffold:imports
	"github.com/akirill0v/cue-flux-controller/internal/controller"
	"github.com/akirill0v/cue-flux-controller/internal/features"
	// intkube "github.com/fluxcd/helm-controller/internal/kube"
	// "github.com/fluxcd/helm-controller/internal/oomwatch"
)

const controllerName = "cue-flux-controller"

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// init initializes the scheme for the API server.
func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = sourcev1b2.AddToScheme(scheme)
	// _ = cuev1alpha1.AddToScheme(scheme) // TODO: Add to scheme

	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr             string
		eventsAddr              string
		healthAddr              string
		concurrent              int
		requeueDependency       time.Duration
		gracefulShutdownTimeout time.Duration
		httpRetry               int
		clientOptions           client.Options
		kubeConfigOpts          client.KubeConfigOptions
		featureGates            feathelper.FeatureGates
		logOptions              logger.Options
		aclOptions              acl.Options
		noRemoteBases           bool
		leaderElectionOptions   leaderelection.Options
		rateLimiterOptions      runtimeCtrl.RateLimiterOptions
		watchOptions            runtimeCtrl.WatchOptions
		defaultServiceAccount   string
	)

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&eventsAddr, "events-addr", "",
		"The address of the events receiver.")
	flag.StringVar(&healthAddr, "health-addr", ":9440",
		"The address the health endpoint binds to.")
	flag.IntVar(&concurrent, "concurrent", 4,
		"The number of concurrent CueInstance reconciles.")
	flag.DurationVar(&requeueDependency, "requeue-dependency", 30*time.Second,
		"The interval at which failing dependencies are reevaluated.")
	flag.DurationVar(&gracefulShutdownTimeout, "graceful-shutdown-timeout", 600*time.Second,
		"The duration given to the reconciler to finish before forcibly stopping.")
	flag.IntVar(&httpRetry, "http-retry", 9,
		"The maximum number of retries when failing to fetch artifacts over HTTP.")
	// flag.StringVar(&intkube.DefaultServiceAccountName, "default-service-account", "",
	// 	"Default service account used for impersonation.")

	clientOptions.BindFlags(flag.CommandLine)
	logOptions.BindFlags(flag.CommandLine)
	leaderElectionOptions.BindFlags(flag.CommandLine)
	aclOptions.BindFlags(flag.CommandLine)
	kubeConfigOpts.BindFlags(flag.CommandLine)
	rateLimiterOptions.BindFlags(flag.CommandLine)
	featureGates.BindFlags(flag.CommandLine)
	watchOptions.BindFlags(flag.CommandLine)

	flag.Parse()

	logger.SetLogger(logger.NewLogger(logOptions))

	ctx := ctrl.SetupSignalHandler()

	if err := featureGates.WithLogger(setupLog).SupportedFeatures(features.FeatureGates()); err != nil {
		setupLog.Error(err, "unable to load feature gates")
		os.Exit(1)
	}

	watchNamespace := ""
	if !watchOptions.AllNamespaces {
		watchNamespace = os.Getenv("RUNTIME_NAMESPACE")
	}

	// watchSelector, err := runtimeCtrl.GetWatchSelector(watchOptions)
	// if err != nil {
	// 	setupLog.Error(err, "unable to configure watch label selector for manager")
	// 	os.Exit(1)
	// }

	var disableCacheFor []ctrlclient.Object
	shouldCache, err := features.Enabled(features.CacheSecretsAndConfigMaps)
	if err != nil {
		setupLog.Error(err, "unable to check feature gate CacheSecretsAndConfigMaps")
		os.Exit(1)
	}
	if !shouldCache {
		disableCacheFor = append(disableCacheFor, &corev1.Secret{}, &corev1.ConfigMap{})
	}

	leaderElectionId := fmt.Sprintf("%s-%s", controllerName, "leader-election")
	if watchOptions.LabelSelector != "" {
		leaderElectionId = leaderelection.GenerateID(leaderElectionId, watchOptions.LabelSelector)
	}

	restConfig := runtimeClient.GetConfigOrDie(clientOptions)
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            metricsAddr,
		HealthProbeBindAddress:        healthAddr,
		LeaderElection:                leaderElectionOptions.Enable,
		LeaderElectionReleaseOnCancel: leaderElectionOptions.ReleaseOnCancel,
		LeaseDuration:                 &leaderElectionOptions.LeaseDuration,
		RenewDeadline:                 &leaderElectionOptions.RenewDeadline,
		RetryPeriod:                   &leaderElectionOptions.RetryPeriod,
		LeaderElectionID:              leaderElectionId,
		Logger:                        ctrl.Log,
		Client: ctrlclient.Options{
			Cache: &ctrlclient.CacheOptions{
				DisableFor: disableCacheFor,
			},
		},
		Cache: ctrlcache.Options{
			ByObject: map[ctrlclient.Object]ctrlcache.ByObject{
				// TODO: Add CueInstance to cache
				// &kustomizev1.Kustomization{}: {Label: watchSelector},
			},
			Namespaces: []string{watchNamespace},
		},
		Controller: ctrlcfg.Controller{
			MaxConcurrentReconciles: concurrent,
			RecoverPanic:            pointer.Bool(true),
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	probes.SetupChecks(mgr, setupLog)
	pprof.SetupHandlers(mgr, setupLog)

	var eventRecorder *events.Recorder
	if eventRecorder, err = events.NewRecorder(mgr, ctrl.Log, eventsAddr, controllerName); err != nil {
		setupLog.Error(err, "unable to create event recorder")
		os.Exit(1)
	}

	metricsH := runtimeCtrl.MustMakeMetrics(mgr)

	// jobStatusReader := statusreaders.NewCustomJobStatusReader(mgr.GetRESTMapper())
	pollingOpts := polling.Options{
		// CustomStatusReaders: []engine.StatusReader{jobStatusReader},
	}

	if ok, _ := features.Enabled(features.DisableStatusPollerCache); ok {
		pollingOpts.ClusterReaderFactory = engine.ClusterReaderFactoryFunc(clusterreader.NewDirectClusterReader)
	}

	if err = (&controller.CueInstanceReconciler{
		ControllerName:        controllerName,
		DefaultServiceAccount: defaultServiceAccount,
		Client:                mgr.GetClient(),
		Metrics:               metricsH,
		EventRecorder:         eventRecorder,
		NoCrossNamespaceRefs:  aclOptions.NoCrossNamespaceRefs,
		NoRemoteBases:         noRemoteBases,
		KubeConfigOpts:        kubeConfigOpts,
		PollingOpts:           pollingOpts,
		StatusPoller:          polling.NewStatusPoller(mgr.GetClient(), mgr.GetRESTMapper(), pollingOpts),
	}).SetupWithManager(ctx, mgr, controller.CueInstanceReconcilerOptions{
		DependencyRequeueInterval: requeueDependency,
		HTTPRetry:                 httpRetry,
		RateLimiter:               runtimeCtrl.GetRateLimiter(rateLimiterOptions),
	}); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", controllerName)
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
