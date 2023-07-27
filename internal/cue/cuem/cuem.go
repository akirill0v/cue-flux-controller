package cuem

import (
	"context"

	cueinstancev1a1 "github.com/akirill0v/cue-flux-controller/api/v1alpha1"
	_ "github.com/akirill0v/cue-flux-controller/internal/cue"
	"github.com/octohelm/cuemod/pkg/cuemod"
)

type CueDependencyManager struct{}

func (m CueDependencyManager) Get(ctx context.Context, rootPath string, upgrade bool, obj *cueinstancev1a1.CueInstance) error {
	cc := cuemod.FromContext(ctx)
	return cc.Get(cuemod.WithOpts(ctx,
		cuemod.OptUpgrade(upgrade),
		cuemod.OptImport("go"),
		cuemod.OptVerbose(true)), rootPath)
}
