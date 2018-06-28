package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/exitcodes"
	"github.com/golangci/golangci-lint/pkg/lint"
	"github.com/golangci/golangci-lint/pkg/lint/lintersdb"
	"github.com/golangci/golangci-lint/pkg/logutils"
	"github.com/golangci/golangci-lint/pkg/printers"
	"github.com/golangci/golangci-lint/pkg/result"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func getDefaultExcludeHelp() string {
	parts := []string{"Use or not use default excludes:"}
	for _, ep := range config.DefaultExcludePatterns {
		parts = append(parts, fmt.Sprintf("  # %s: %s", ep.Linter, ep.Why))
		parts = append(parts, fmt.Sprintf("  - %s", color.YellowString(ep.Pattern)))
		parts = append(parts, "")
	}
	return strings.Join(parts, "\n")
}

const welcomeMessage = "Run this tool in cloud on every github pull " +
	"request in https://golangci.com for free (public repos)"

func wh(text string) string {
	return color.GreenString(text)
}

func initFlagSet(fs *pflag.FlagSet, cfg *config.Config) {
	hideFlag := func(name string) {
		if err := fs.MarkHidden(name); err != nil {
			panic(err)
		}
	}

	// Output config
	oc := &cfg.Output
	fs.StringVar(&oc.Format, "out-format",
		config.OutFormatColoredLineNumber,
		wh(fmt.Sprintf("Format of output: %s", strings.Join(config.OutFormats, "|"))))
	fs.BoolVar(&oc.PrintIssuedLine, "print-issued-lines", true, wh("Print lines of code with issue"))
	fs.BoolVar(&oc.PrintLinterName, "print-linter-name", true, wh("Print linter name in issue line"))
	fs.BoolVar(&oc.PrintWelcomeMessage, "print-welcome", false, wh("Print welcome message"))
	hideFlag("print-welcome") // no longer used

	// Run config
	rc := &cfg.Run
	fs.IntVar(&rc.ExitCodeIfIssuesFound, "issues-exit-code",
		exitcodes.IssuesFound, wh("Exit code when issues were found"))
	fs.StringSliceVar(&rc.BuildTags, "build-tags", nil, wh("Build tags"))
	fs.DurationVar(&rc.Deadline, "deadline", time.Minute, wh("Deadline for total work"))
	fs.BoolVar(&rc.AnalyzeTests, "tests", true, wh("Analyze tests (*_test.go)"))
	fs.BoolVar(&rc.PrintResourcesUsage, "print-resources-usage", false,
		wh("Print avg and max memory usage of golangci-lint and total time"))
	fs.StringVarP(&rc.Config, "config", "c", "", wh("Read config from file path `PATH`"))
	fs.BoolVar(&rc.NoConfig, "no-config", false, wh("Don't read config"))
	fs.StringSliceVar(&rc.SkipDirs, "skip-dirs", nil, wh("Regexps of directories to skip"))
	fs.StringSliceVar(&rc.SkipFiles, "skip-files", nil, wh("Regexps of files to skip"))

	// Linters settings config
	lsc := &cfg.LintersSettings

	// Hide all linters settings flags: they were initially visible,
	// but when number of linters started to grow it became ovious that
	// we can't fill 90% of flags by linters settings: common flags became hard to find.
	// New linters settings should be done only through config file.
	fs.BoolVar(&lsc.Errcheck.CheckTypeAssertions, "errcheck.check-type-assertions",
		false, "Errcheck: check for ignored type assertion results")
	hideFlag("errcheck.check-type-assertions")

	fs.BoolVar(&lsc.Errcheck.CheckAssignToBlank, "errcheck.check-blank", false,
		"Errcheck: check for errors assigned to blank identifier: _ = errFunc()")
	hideFlag("errcheck.check-blank")

	fs.BoolVar(&lsc.Govet.CheckShadowing, "govet.check-shadowing", false,
		"Govet: check for shadowed variables")
	hideFlag("govet.check-shadowing")

	fs.Float64Var(&lsc.Golint.MinConfidence, "golint.min-confidence", 0.8,
		"Golint: minimum confidence of a problem to print it")
	hideFlag("golint.min-confidence")

	fs.BoolVar(&lsc.Gofmt.Simplify, "gofmt.simplify", true, "Gofmt: simplify code")
	hideFlag("gofmt.simplify")

	fs.IntVar(&lsc.Gocyclo.MinComplexity, "gocyclo.min-complexity",
		30, "Minimal complexity of function to report it")
	hideFlag("gocyclo.min-complexity")

	fs.BoolVar(&lsc.Maligned.SuggestNewOrder, "maligned.suggest-new", false,
		"Maligned: print suggested more optimal struct fields ordering")
	hideFlag("maligned.suggest-new")

	fs.IntVar(&lsc.Dupl.Threshold, "dupl.threshold",
		150, "Dupl: Minimal threshold to detect copy-paste")
	hideFlag("dupl.threshold")

	fs.IntVar(&lsc.Goconst.MinStringLen, "goconst.min-len",
		3, "Goconst: minimum constant string length")
	hideFlag("goconst.min-len")
	fs.IntVar(&lsc.Goconst.MinOccurrencesCount, "goconst.min-occurrences",
		3, "Goconst: minimum occurrences of constant string count to trigger issue")
	hideFlag("goconst.min-occurrences")

	// (@dixonwille) These flag is only used for testing purposes.
	fs.StringSliceVar(&lsc.Depguard.Packages, "depguard.packages", nil,
		"Depguard: packages to add to the list")
	hideFlag("depguard.packages")

	fs.BoolVar(&lsc.Depguard.IncludeGoRoot, "depguard.include-go-root", false,
		"Depguard: check list against standard lib")
	hideFlag("depguard.include-go-root")

	// Linters config
	lc := &cfg.Linters
	fs.StringSliceVarP(&lc.Enable, "enable", "E", nil, wh("Enable specific linter"))
	fs.StringSliceVarP(&lc.Disable, "disable", "D", nil, wh("Disable specific linter"))
	fs.BoolVar(&lc.EnableAll, "enable-all", false, wh("Enable all linters"))
	fs.BoolVar(&lc.DisableAll, "disable-all", false, wh("Disable all linters"))
	fs.StringSliceVarP(&lc.Presets, "presets", "p", nil,
		wh(fmt.Sprintf("Enable presets (%s) of linters. Run 'golangci-lint linters' to see "+
			"them. This option implies option --disable-all", strings.Join(lintersdb.AllPresets(), "|"))))
	fs.BoolVar(&lc.Fast, "fast", false, wh("Run only fast linters from enabled linters set"))

	// Issues config
	ic := &cfg.Issues
	fs.StringSliceVarP(&ic.ExcludePatterns, "exclude", "e", nil, wh("Exclude issue by regexp"))
	fs.BoolVar(&ic.UseDefaultExcludes, "exclude-use-default", true, getDefaultExcludeHelp())

	fs.IntVar(&ic.MaxIssuesPerLinter, "max-issues-per-linter", 50,
		wh("Maximum issues count per one linter. Set to 0 to disable"))
	fs.IntVar(&ic.MaxSameIssues, "max-same-issues", 3,
		wh("Maximum count of issues with the same text. Set to 0 to disable"))

	fs.BoolVarP(&ic.Diff, "new", "n", false,
		wh("Show only new issues: if there are unstaged changes or untracked files, only those changes "+
			"are analyzed, else only changes in HEAD~ are analyzed.\nIt's a super-useful option for integration "+
			"of golangci-lint into existing large codebase.\nIt's not practical to fix all existing issues at "+
			"the moment of integration: much better don't allow issues in new code"))
	fs.StringVar(&ic.DiffFromRevision, "new-from-rev", "",
		wh("Show only new issues created after git revision `REV`"))
	fs.StringVar(&ic.DiffPatchFilePath, "new-from-patch", "",
		wh("Show only new issues created in git patch with file path `PATH`"))

}

func (e *Executor) initRun() {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: welcomeMessage,
		Run:   e.executeRun,
	}
	e.rootCmd.AddCommand(runCmd)

	runCmd.SetOutput(logutils.StdOut) // use custom output to properly color it in Windows terminals

	fs := runCmd.Flags()
	fs.SortFlags = false // sort them as they are defined here
	initFlagSet(fs, e.cfg)

	// init e.cfg by values from config: flags parse will see these values
	// like the default ones. It will overwrite them only if the same option
	// is found in command-line: it's ok, command-line has higher priority.

	r := config.NewFileReader(e.cfg, e.log.Child("config_reader"), func(fs *pflag.FlagSet, cfg *config.Config) {
		// Don't do `fs.AddFlagSet(cmd.Flags())` because it shares flags representations:
		// `changed` variable inside string slice vars will be shared.
		// Use another config variable here, not e.cfg, to not
		// affect main parsing by this parsing of only config option.
		initFlagSet(fs, cfg)

		// Parse max options, even force version option: don't want
		// to get access to Executor here: it's error-prone to use
		// cfg vs e.cfg.
		initRootFlagSet(fs, cfg, true)
	})
	if err := r.Read(); err != nil {
		e.log.Fatalf("Can't read config: %s", err)
	}

	// Slice options must be explicitly set for proper merging of config and command-line options.
	fixSlicesFlags(fs)
}

func fixSlicesFlags(fs *pflag.FlagSet) {
	// It's a dirty hack to set flag.Changed to true for every string slice flag.
	// It's necessary to merge config and command-line slices: otherwise command-line
	// flags will always overwrite ones from the config.
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Value.Type() != "stringSlice" {
			return
		}

		s, err := fs.GetStringSlice(f.Name)
		if err != nil {
			return
		}

		if s == nil { // assume that every string slice flag has nil as the default
			return
		}

		// calling Set sets Changed to true: next Set calls will append, not overwrite
		_ = f.Value.Set(strings.Join(s, ","))
	})
}

func (e *Executor) runAnalysis(ctx context.Context, args []string) (<-chan result.Issue, error) {
	e.cfg.Run.Args = args

	linters, err := lintersdb.GetEnabledLinters(e.cfg, e.log.Child("lintersdb"))
	if err != nil {
		return nil, err
	}

	for _, lc := range lintersdb.GetAllSupportedLinterConfigs() {
		isEnabled := false
		for _, linter := range linters {
			if linter.Linter.Name() == lc.Linter.Name() {
				isEnabled = true
				break
			}
		}
		e.reportData.AddLinter(lc.Linter.Name(), isEnabled, lc.EnabledByDefault)
	}

	lintCtx, err := lint.LoadContext(ctx, linters, e.cfg, e.log.Child("load"))
	if err != nil {
		return nil, err
	}

	runner, err := lint.NewRunner(lintCtx.ASTCache, e.cfg, e.log.Child("runner"))
	if err != nil {
		return nil, err
	}

	return runner.Run(ctx, linters, lintCtx), nil
}

func (e *Executor) setOutputToDevNull() (savedStdout, savedStderr *os.File) {
	savedStdout, savedStderr = os.Stdout, os.Stderr
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		e.log.Warnf("Can't open null device %q: %s", os.DevNull, err)
		return
	}

	os.Stdout, os.Stderr = devNull, devNull
	return
}

func (e *Executor) runAndPrint(ctx context.Context, args []string) error {
	if !logutils.HaveDebugTag("linters_output") {
		// Don't allow linters and loader to print anything
		log.SetOutput(ioutil.Discard)
		savedStdout, savedStderr := e.setOutputToDevNull()
		defer func() {
			os.Stdout, os.Stderr = savedStdout, savedStderr
		}()
	}

	issues, err := e.runAnalysis(ctx, args)
	if err != nil {
		return err
	}

	p, err := e.createPrinter()
	if err != nil {
		return err
	}

	gotAnyIssues, err := p.Print(ctx, issues)
	if err != nil {
		return fmt.Errorf("can't print %d issues: %s", len(issues), err)
	}

	if gotAnyIssues {
		e.exitCode = e.cfg.Run.ExitCodeIfIssuesFound
		return nil
	}

	return nil
}

func (e *Executor) createPrinter() (printers.Printer, error) {
	var p printers.Printer
	format := e.cfg.Output.Format
	switch format {
	case config.OutFormatJSON:
		p = printers.NewJSON(&e.reportData)
	case config.OutFormatColoredLineNumber, config.OutFormatLineNumber:
		p = printers.NewText(e.cfg.Output.PrintIssuedLine,
			format == config.OutFormatColoredLineNumber, e.cfg.Output.PrintLinterName, e.cfg.Run.Silent,
			e.log.Child("text_printer"))
	case config.OutFormatTab:
		p = printers.NewTab(e.cfg.Output.PrintLinterName, e.cfg.Run.Silent,
			e.log.Child("tab_printer"))
	case config.OutFormatCheckstyle:
		p = printers.NewCheckstyle()
	default:
		return nil, fmt.Errorf("unknown output format %s", format)
	}

	return p, nil
}

func (e *Executor) executeRun(cmd *cobra.Command, args []string) {
	needTrackResources := e.cfg.Run.IsVerbose || e.cfg.Run.PrintResourcesUsage
	trackResourcesEndCh := make(chan struct{})
	defer func() { // XXX: this defer must be before ctx.cancel defer
		if needTrackResources { // wait until resource tracking finished to print properly
			<-trackResourcesEndCh
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), e.cfg.Run.Deadline)
	defer cancel()

	if needTrackResources {
		go watchResources(ctx, trackResourcesEndCh, e.log)
	}

	if err := e.runAndPrint(ctx, args); err != nil {
		e.log.Errorf("Running error: %s", err)
		if e.exitCode == exitcodes.Success {
			e.exitCode = exitcodes.Failure
		}
	}

	if e.exitCode == exitcodes.Success && ctx.Err() != nil {
		e.exitCode = exitcodes.Timeout
	}
}

func watchResources(ctx context.Context, done chan struct{}, log logutils.Log) {
	startedAt := time.Now()

	rssValues := []uint64{}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		rssValues = append(rssValues, m.Sys)

		stop := false
		select {
		case <-ctx.Done():
			stop = true
		case <-ticker.C: // track every second
		}

		if stop {
			break
		}
	}

	var avg, max uint64
	for _, v := range rssValues {
		avg += v
		if v > max {
			max = v
		}
	}
	avg /= uint64(len(rssValues))

	const MB = 1024 * 1024
	maxMB := float64(max) / MB
	log.Infof("Memory: %d samples, avg is %.1fMB, max is %.1fMB",
		len(rssValues), float64(avg)/MB, maxMB)
	log.Infof("Execution took %s", time.Since(startedAt))
	close(done)
}
