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

var DefaultExcludePatterns = []string{
	// errcheck
	"Error return value of .((os\\.)?std(out|err)\\..*|.*Close|os\\.Remove(All)?|.*printf?|os\\.(Un)?Setenv). is not checked",

	// golint
	"should have comment",
	"comment on exported method",

	// gas
	"G103:", // Use of unsafe calls should be audited
	"G104:", // disable what errcheck does: it reports on Close etc
	"G204:", // Subprocess launching should be audited: too lot false positives
	"G301:", // Expect directory permissions to be 0750 or less
	"G302:", // Expect file permissions to be 0600 or less
	"G304:", // Potential file inclusion via variable: `src, err := ioutil.ReadFile(filename)`

	// govet
	"possible misuse of unsafe.Pointer",
	"should have signature",

	// megacheck
	"ineffective break statement. Did you mean to break out of the outer loop", // developers tend to write in C-style with break in switch
}

type Common struct {
	IsVerbose      bool
	CPUProfilePath string
	Concurrency    int
}

type Run struct { // nolint:maligned
	Args []string

	BuildTags []string

	OutFormat           string
	PrintIssuedLine     bool
	PrintLinterName     bool
	PrintWelcomeMessage bool

	ExitCodeIfIssuesFound int

	Errcheck struct {
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

	Presets []string

	ExcludePatterns    []string
	UseDefaultExcludes bool

	Deadline time.Duration

	MaxIssuesPerLinter int
	MaxSameIssues      int

	DiffFromRevision  string
	DiffPatchFilePath string
	Diff              bool

	AnalyzeTests bool
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
