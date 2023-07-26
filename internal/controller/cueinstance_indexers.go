package controller

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	cueinstancev1a1 "github.com/akirill0v/cue-flux-controller/api/v1alpha1"
)

func (r *CueInstanceReconciler) requestsForRevisionChangeOf(indexKey string) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx)
		repo, ok := obj.(interface {
			GetArtifact() *sourcev1.Artifact
		})
		if !ok {
			log.Error(fmt.Errorf("expected an object conformed with GetArtifact() method, but got a %T", obj),
				"failed to get reconcile requests for revision change")
			return nil
		}
		// If we do not have an artifact, we have no requests to make
		if repo.GetArtifact() == nil {
			return nil
		}

		var list cueinstancev1a1.CueInstanceList
		if err := r.List(ctx, &list, client.MatchingFields{
			indexKey: client.ObjectKeyFromObject(obj).String(),
		}); err != nil {
			log.Error(err, "failed to list objects for revision change")
			return nil
		}

		var dd []dependency.Dependent
		for _, d := range list.Items {
			// If the revision of the artifact equals to the last attempted revision,
			// we should not make a request for this CueInstance
			if repo.GetArtifact().HasRevision(d.Status.LastAttemptedRevision) {
				continue
			}
			dd = append(dd, d.DeepCopy())
		}
		sorted, err := dependency.Sort(dd)
		if err != nil {
			log.Error(err, "failed to sort dependencies for revision change")
			return nil
		}
		reqs := make([]reconcile.Request, len(sorted))
		for i := range sorted {
			reqs[i].NamespacedName.Name = sorted[i].Name
			reqs[i].NamespacedName.Namespace = sorted[i].Namespace
		}
		return reqs
	}
}

func (r *CueInstanceReconciler) indexBy(kind string) func(o client.Object) []string {
	return func(o client.Object) []string {
		c, ok := o.(*cueinstancev1a1.CueInstance)
		if !ok {
			panic(fmt.Sprintf("Expected a CueInstance, got %T", o))
		}

		if c.Spec.SourceRef.Kind == kind {
			namespace := c.GetNamespace()
			if c.Spec.SourceRef.Namespace != "" {
				namespace = c.Spec.SourceRef.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", namespace, c.Spec.SourceRef.Name)}
		}

		return nil
	}
}
