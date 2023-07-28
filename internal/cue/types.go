package cue

import (
	"context"

	cueinstancev1a1 "github.com/akirill0v/cue-flux-controller/api/v1alpha1"
)

// Interface for cue dependency manager
type DependencyManager interface {
	Get(ctx context.Context, moduleRootPath, dirPath string, upgrade bool, obj *cueinstancev1a1.CueInstance) error
}
