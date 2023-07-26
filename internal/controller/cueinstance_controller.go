package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxcd/pkg/http/fetch"
	runtimeClient "github.com/fluxcd/pkg/runtime/client"
	runtimeCtrl "github.com/fluxcd/pkg/runtime/controller"
	kuberecorder "k8s.io/client-go/tools/record"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"

	cueinstancev1a1 "github.com/akirill0v/cue-flux-controller/api/v1alpha1"
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
	const (
		ociRepositoryIndexKey string = ".metadata.ociRepository"
		gitRepositoryIndexKey string = ".metadata.gitRepository"
		bucketIndexKey        string = ".metadata.bucket"
	)

	// Index the CueInstances by the OCIRepository references they (may) point at.
	if err := mgr.GetCache().IndexField(ctx, &cueinstancev1a1.CueInstance{}, ociRepositoryIndexKey,
		r.indexBy(sourcev1b2.OCIRepositoryKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	// Index the CueInstances by the GitRepository references they (may) point at.
	if err := mgr.GetCache().IndexField(ctx, &cueinstancev1a1.CueInstance{}, gitRepositoryIndexKey,
		r.indexBy(sourcev1.GitRepositoryKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	// Index the CueInstances by the Bucket references they (may) point at.
	if err := mgr.GetCache().IndexField(ctx, &cueinstancev1a1.CueInstance{}, bucketIndexKey,
		r.indexBy(sourcev1b2.BucketKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	return nil
}
