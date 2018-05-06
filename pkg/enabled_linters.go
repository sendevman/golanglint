package pkg

import (
	"context"
	"fmt"
	"sync"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/golinters"
	"github.com/golangci/golangci-shared/pkg/analytics"
)

type LinterConfig struct {
	EnabledByDefault bool
	Desc             string
	Linter           Linter
	DoesFullImport   bool
}

var nameToLC map[string]LinterConfig
var nameToLCOnce sync.Once

func GetLinterConfig(name string) *LinterConfig {
	nameToLCOnce.Do(func() {
		nameToLC = make(map[string]LinterConfig)
		for _, lc := range GetAllSupportedLinterConfigs() {
			nameToLC[lc.Linter.Name()] = lc
		}
	})

	lc, ok := nameToLC[name]
	if !ok {
		return nil
	}

	return &lc
}

func enabledByDefault(linter Linter, desc string, doesFullImport bool) LinterConfig {
	return LinterConfig{
		EnabledByDefault: true,
		Linter:           linter,
		Desc:             desc,
		DoesFullImport:   doesFullImport,
	}
}

func disabledByDefault(linter Linter, desc string, doesFullImport bool) LinterConfig {
	return LinterConfig{
		EnabledByDefault: false,
		Linter:           linter,
		Desc:             desc,
		DoesFullImport:   doesFullImport,
	}
}

func GetAllSupportedLinterConfigs() []LinterConfig {
	return []LinterConfig{
		enabledByDefault(golinters.Govet{}, "Vet examines Go source code and reports suspicious constructs, such as Printf calls whose arguments do not align with the format string", false),
		enabledByDefault(golinters.Errcheck{}, "Errcheck is a program for checking for unchecked errors in go programs. These unchecked errors can be critical bugs in some cases", true),
		enabledByDefault(golinters.Golint{}, "Golint differs from gofmt. Gofmt reformats Go source code, whereas golint prints out style mistakes", false),
		enabledByDefault(golinters.Deadcode{}, "Finds unused code", true),
		enabledByDefault(golinters.Gocyclo{}, "Computes and checks the cyclomatic complexity of functions", false),

		disabledByDefault(golinters.Gofmt{}, "Gofmt checks whether code was gofmt-ed. By default this tool runs with -s option to check for code simplification", false),
		disabledByDefault(golinters.Gofmt{UseGoimports: true}, "Goimports does everything that gofmt does. Additionally it checks unused imports", false),
	}
}

func getAllSupportedLinters() []Linter {
	var ret []Linter
	for _, lc := range GetAllSupportedLinterConfigs() {
		ret = append(ret, lc.Linter)
	}

	return ret
}

func getAllEnabledByDefaultLinters() []Linter {
	var ret []Linter
	for _, lc := range GetAllSupportedLinterConfigs() {
		if lc.EnabledByDefault {
			ret = append(ret, lc.Linter)
		}
	}

	return ret
}

var supportedLintersByName map[string]Linter
var linterByNameMapOnce sync.Once

func getLinterByName(name string) Linter {
	linterByNameMapOnce.Do(func() {
		supportedLintersByName = make(map[string]Linter)
		for _, lc := range GetAllSupportedLinterConfigs() {
			supportedLintersByName[lc.Linter.Name()] = lc.Linter
		}
	})

	return supportedLintersByName[name]
}

func lintersToMap(linters []Linter) map[string]Linter {
	ret := map[string]Linter{}
	for _, linter := range linters {
		ret[linter.Name()] = linter
	}

	return ret
}

func validateEnabledDisabledLintersConfig(cfg *config.Run) error {
	allNames := append([]string{}, cfg.EnabledLinters...)
	allNames = append(allNames, cfg.DisabledLinters...)
	for _, name := range allNames {
		if getLinterByName(name) == nil {
			return fmt.Errorf("no such linter %q", name)
		}
	}

	if cfg.EnableAllLinters && cfg.DisableAllLinters {
		return fmt.Errorf("--enable-all and --disable-all options must not be combined")
	}

	if cfg.DisableAllLinters {
		if len(cfg.EnabledLinters) == 0 {
			return fmt.Errorf("all linters were disabled, but no one linter was enabled: must enable at least one")
		}

		if len(cfg.DisabledLinters) != 0 {
			return fmt.Errorf("can't combine options --disable-all and --disable %s", cfg.DisabledLinters[0])
		}
	}

	if cfg.EnableAllLinters && len(cfg.EnabledLinters) != 0 {
		return fmt.Errorf("can't combine options --enable-all and --enable %s", cfg.EnabledLinters[0])
	}

	enabledLintersSet := map[string]bool{}
	for _, name := range cfg.EnabledLinters {
		enabledLintersSet[name] = true
	}

	for _, name := range cfg.DisabledLinters {
		if enabledLintersSet[name] {
			return fmt.Errorf("linter %q can't be disabled and enabled at one moment", name)
		}
	}

	return nil
}

func GetEnabledLinters(ctx context.Context, cfg *config.Run) ([]Linter, error) {
	if err := validateEnabledDisabledLintersConfig(cfg); err != nil {
		return nil, err
	}
	resultLintersSet := map[string]Linter{}
	switch {
	case cfg.EnableAllLinters:
		resultLintersSet = lintersToMap(getAllSupportedLinters())
	case cfg.DisableAllLinters:
		break
	default:
		resultLintersSet = lintersToMap(getAllEnabledByDefaultLinters())
	}

	for _, name := range cfg.EnabledLinters {
		resultLintersSet[name] = getLinterByName(name)
	}

	for _, name := range cfg.DisabledLinters {
		delete(resultLintersSet, name)
	}

	var resultLinters []Linter
	var resultLinterNames []string
	for name, linter := range resultLintersSet {
		resultLinters = append(resultLinters, linter)
		resultLinterNames = append(resultLinterNames, name)
	}
	analytics.Log(ctx).Infof("Enabled linters: %s", resultLinterNames)

	return resultLinters, nil
}
