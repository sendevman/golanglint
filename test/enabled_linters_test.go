package test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/golangci/golangci-lint/pkg/lint/lintersdb"
	"github.com/golangci/golangci-lint/test/testshared"
)

func inSlice(s []string, v string) bool {
	for _, sv := range s {
		if sv == v {
			return true
		}
	}

	return false
}

func getEnabledByDefaultFastLintersExcept(except ...string) []string {
	m := lintersdb.NewManager(nil, nil)
	ebdl := m.GetAllEnabledByDefaultLinters()
	var ret []string
	for _, lc := range ebdl {
		if lc.IsSlowLinter() {
			continue
		}

		if !inSlice(except, lc.Name()) {
			ret = append(ret, lc.Name())
		}
	}

	return ret
}

func getAllFastLintersWith(with ...string) []string {
	linters := lintersdb.NewManager(nil, nil).GetAllSupportedLinterConfigs()
	ret := append([]string{}, with...)
	for _, lc := range linters {
		if lc.IsSlowLinter() {
			continue
		}
		ret = append(ret, lc.Name())
	}

	return ret
}

func getEnabledByDefaultLinters() []string {
	ebdl := lintersdb.NewManager(nil, nil).GetAllEnabledByDefaultLinters()
	var ret []string
	for _, lc := range ebdl {
		ret = append(ret, lc.Name())
	}

	return ret
}

func getEnabledByDefaultFastLintersWith(with ...string) []string {
	ebdl := lintersdb.NewManager(nil, nil).GetAllEnabledByDefaultLinters()
	ret := append([]string{}, with...)
	for _, lc := range ebdl {
		if lc.IsSlowLinter() {
			continue
		}

		ret = append(ret, lc.Name())
	}

	return ret
}

//nolint:funlen
func TestEnabledLinters(t *testing.T) {
	cases := []struct {
		name           string
		cfg            string
		enabledLinters []string
		args           []string
		noImplicitFast bool
	}{
		{
			name: "disable govet in config",
			cfg: `
			linters:
				disable:
					- govet
			`,
			enabledLinters: getEnabledByDefaultFastLintersExcept("govet"),
		},
		{
			name: "enable golint in config",
			cfg: `
			linters:
				enable:
					- golint
			`,
			enabledLinters: getEnabledByDefaultFastLintersWith("golint"),
		},
		{
			name:           "disable govet in cmd",
			args:           []string{"-Dgovet"},
			enabledLinters: getEnabledByDefaultFastLintersExcept("govet"),
		},
		{
			name: "enable gofmt in cmd and enable golint in config",
			args: []string{"-Egofmt"},
			cfg: `
			linters:
				enable:
					- golint
			`,
			enabledLinters: getEnabledByDefaultFastLintersWith("golint", "gofmt"),
		},
		{
			name: "fast option in config",
			cfg: `
			linters:
				fast: true
			`,
			enabledLinters: getEnabledByDefaultFastLintersWith(),
			noImplicitFast: true,
		},
		{
			name: "explicitly unset fast option in config",
			cfg: `
			linters:
				fast: false
			`,
			enabledLinters: getEnabledByDefaultLinters(),
			noImplicitFast: true,
		},
		{
			name:           "set fast option in command-line",
			args:           []string{"--fast"},
			enabledLinters: getEnabledByDefaultFastLintersWith(),
			noImplicitFast: true,
		},
		{
			name: "fast option in command-line has higher priority to enable",
			cfg: `
			linters:
				fast: false
			`,
			args:           []string{"--fast"},
			enabledLinters: getEnabledByDefaultFastLintersWith(),
			noImplicitFast: true,
		},
		{
			name: "fast option in command-line has higher priority to disable",
			cfg: `
			linters:
				fast: true
			`,
			args:           []string{"--fast=false"},
			enabledLinters: getEnabledByDefaultLinters(),
			noImplicitFast: true,
		},
		{
			name:           "fast option combined with enable and enable-all",
			args:           []string{"--enable-all", "--fast", "--enable=unused"},
			enabledLinters: getAllFastLintersWith("unused"),
			noImplicitFast: true,
		},
	}

	testshared.InstallGolangciLint(t)

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"--verbose"}
			if !c.noImplicitFast {
				args = append(args, "--fast")
			}

			r := testshared.NewRunnerBuilder(t).
				WithCommand("linters").
				WithArgs(args...).
				WithArgs(c.args...).
				WithConfig(c.cfg).
				Runner().
				Run()

			sort.StringSlice(c.enabledLinters).Sort()

			r.ExpectOutputContains(fmt.Sprintf("Active %d linters: [%s]",
				len(c.enabledLinters), strings.Join(c.enabledLinters, " ")))
		})
	}
}
