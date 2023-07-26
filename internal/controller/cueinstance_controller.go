package controller

import (
	"context"
	"time"

	"github.com/fluxcd/pkg/http/fetch"
	runtimeClient "github.com/fluxcd/pkg/runtime/client"
	runtimeCtrl "github.com/fluxcd/pkg/runtime/controller"
	kuberecorder "k8s.io/client-go/tools/record"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
)

type CueInstanceReconciler struct {
	client.Client
	kuberecorder.EventRecorder
	runtimeCtrl.Metrics

	artifactFetcher       *fetch.ArchiveFetcher
	requeueDependency     time.Duration
	StatusPoller          *polling.StatusPoller
	PollingOpts           polling.Options
	ControllerName        string
	statusManager         string
	NoCrossNamespaceRefs  bool
	NoRemoteBases         bool
	DefaultServiceAccount string
	KubeConfigOpts        runtimeClient.KubeConfigOptions
}

// CueInstanceReconcilerOptions contains options for the CueInstanceReconciler.
type CueInstanceReconcilerOptions struct {
	HTTPRetry                 int
	DependencyRequeueInterval time.Duration
	RateLimiter               ratelimiter.RateLimiter
}

func (r *CueInstanceReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts CueInstanceReconcilerOptions) error {
	return nil
}
