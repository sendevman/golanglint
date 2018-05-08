package golinters

import (
	"context"

	megacheckAPI "github.com/golangci/go-tools/cmd/megacheck"
	"github.com/golangci/golangci-lint/pkg/result"
)

type Megacheck struct{}

func (Megacheck) Name() string {
	return "megacheck"
}

func (Megacheck) Desc() string {
	return "Megacheck: 3 sub-linters in one: staticcheck, gosimple and unused"
}

func (m Megacheck) Run(ctx context.Context, lintCtx *Context) ([]result.Issue, error) {
	c := lintCtx.RunCfg().Megacheck
	issues := megacheckAPI.Run(lintCtx.Program, lintCtx.LoaderConfig, lintCtx.SSAProgram, c.EnableStaticcheck, c.EnableGosimple, c.EnableUnused)

	var res []result.Issue
	for _, i := range issues {
		res = append(res, result.Issue{
			Pos:        i.Position,
			Text:       i.Text,
			FromLinter: m.Name(),
		})
	}
	return res, nil
}
