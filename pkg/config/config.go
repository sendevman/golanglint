package config

import (
	"time"
)

type OutFormat string

const (
	OutFormatJSON              = "json"
	OutFormatLineNumber        = "line-number"
	OutFormatColoredLineNumber = "colored-line-number"
)

var OutFormats = []string{OutFormatColoredLineNumber, OutFormatLineNumber, OutFormatJSON}

var DefaultExcludePatterns = []string{"should have comment", "comment on exported method"}

type Common struct {
	IsVerbose      bool
	CPUProfilePath string
	Concurrency    int
}

type Run struct {
	Args []string

	BuildTags []string

	OutFormat             string
	ExitCodeIfIssuesFound int

	Errcheck struct {
		CheckClose          bool
		CheckTypeAssertions bool
		CheckAssignToBlank  bool
	}
	Govet struct {
		CheckShadowing bool
	}
	Golint struct {
		MinConfidence float64
	}
	Gofmt struct {
		Simplify bool
	}
	Gocyclo struct {
		MinComplexity int
	}
	Varcheck struct {
		CheckExportedFields bool
	}
	Structcheck struct {
		CheckExportedFields bool
	}
	Maligned struct {
		SuggestNewOrder bool
	}
	Megacheck struct {
		EnableStaticcheck bool
		EnableUnused      bool
		EnableGosimple    bool
	}
	Dupl struct {
		Threshold int
	}
	Goconst struct {
		MinStringLen        int
		MinOccurrencesCount int
	}

	EnabledLinters    []string
	DisabledLinters   []string
	EnableAllLinters  bool
	DisableAllLinters bool

	ExcludePatterns []string

	Deadline time.Duration

	MaxIssuesPerLinter int

	DiffFromRevision  string
	DiffPatchFilePath string
	Diff              bool
}

type Config struct {
	Common Common
	Run    Run
}

func NewDefault() *Config {
	return &Config{
		Run: Run{
			OutFormat: OutFormatColoredLineNumber,
		},
	}
}
