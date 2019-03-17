package test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golangci/golangci-lint/test/testshared"

	"github.com/golangci/golangci-lint/pkg/exitcodes"
)

func getCommonRunArgs() []string {
	return []string{"--skip-dirs", "testdata_etc/,pkg/golinters/goanalysis/(checker|passes)"}
}

func withCommonRunArgs(args ...string) []string {
	return append(getCommonRunArgs(), args...)
}

func TestAutogeneratedNoIssues(t *testing.T) {
	testshared.NewLintRunner(t).Run(getTestDataDir("autogenerated")).ExpectNoIssues()
}

func TestEmptyDirRun(t *testing.T) {
	testshared.NewLintRunner(t).Run(getTestDataDir("nogofiles")).
		ExpectExitCode(exitcodes.NoGoFiles).
		ExpectOutputContains(": no go files to analyze")
}

func TestNotExistingDirRun(t *testing.T) {
	testshared.NewLintRunner(t).Run(getTestDataDir("no_such_dir")).
		ExpectExitCode(exitcodes.WarningInTest).
		ExpectOutputContains(`cannot find package \"./testdata/no_such_dir\"`)
}

func TestSymlinkLoop(t *testing.T) {
	testshared.NewLintRunner(t).Run(getTestDataDir("symlink_loop", "...")).ExpectNoIssues()
}

func TestDeadline(t *testing.T) {
	testshared.NewLintRunner(t).Run("--deadline=1ms", getProjectRoot()).
		ExpectExitCode(exitcodes.Timeout).
		ExpectOutputContains(`Deadline exceeded: try increase it by passing --deadline option`)
}

func TestTestsAreLintedByDefault(t *testing.T) {
	testshared.NewLintRunner(t).Run(getTestDataDir("withtests")).
		ExpectHasIssue("`if` block ends with a `return`")
}

func TestCgoOk(t *testing.T) {
	testshared.NewLintRunner(t).Run("--enable-all", getTestDataDir("cgo")).ExpectNoIssues()
}

func TestCgoWithIssues(t *testing.T) {
	testshared.NewLintRunner(t).Run("--enable-all", getTestDataDir("cgo_with_issues")).
		ExpectHasIssue("Printf format %t has arg cs of wrong type")
}

func TestUnsafeOk(t *testing.T) {
	testshared.NewLintRunner(t).Run("--enable-all", getTestDataDir("unsafe")).ExpectNoIssues()
}

func TestSkippedDirsNoMatchArg(t *testing.T) {
	dir := getTestDataDir("skipdirs", "skip_me", "nested")
	r := testshared.NewLintRunner(t).Run("--print-issued-lines=false", "--no-config", "--skip-dirs", dir, "-Egolint", dir)

	r.ExpectExitCode(exitcodes.IssuesFound).
		ExpectOutputEq("testdata/skipdirs/skip_me/nested/with_issue.go:8:9: `if` block ends with " +
			"a `return` statement, so drop this `else` and outdent its block (golint)\n")
}

func TestSkippedDirsTestdata(t *testing.T) {
	r := testshared.NewLintRunner(t).Run("--print-issued-lines=false", "--no-config", "-Egolint", getTestDataDir("skipdirs", "..."))

	r.ExpectNoIssues() // all was skipped because in testdata
}

func TestDeadcodeNoFalsePositivesInMainPkg(t *testing.T) {
	testshared.NewLintRunner(t).Run("--no-config", "--disable-all", "-Edeadcode", getTestDataDir("deadcode_main_pkg")).ExpectNoIssues()
}

func TestIdentifierUsedOnlyInTests(t *testing.T) {
	testshared.NewLintRunner(t).Run("--no-config", "--disable-all", "-Eunused", getTestDataDir("used_only_in_tests")).ExpectNoIssues()
}

func TestConfigFileIsDetected(t *testing.T) {
	checkGotConfig := func(r *testshared.RunResult) {
		r.ExpectExitCode(exitcodes.Success).
			ExpectOutputEq("test\n") // test config contains InternalTest: true, it triggers such output
	}

	r := testshared.NewLintRunner(t)
	checkGotConfig(r.Run(getTestDataDir("withconfig", "pkg")))
	checkGotConfig(r.Run(getTestDataDir("withconfig", "...")))
}

func TestEnableAllFastAndEnableCanCoexist(t *testing.T) {
	r := testshared.NewLintRunner(t)
	r.Run(withCommonRunArgs("--fast", "--enable-all", "--enable=typecheck")...).ExpectNoIssues()
	r.Run(withCommonRunArgs("--enable-all", "--enable=typecheck")...).ExpectExitCode(exitcodes.Failure)
}

func TestEnabledPresetsAreNotDuplicated(t *testing.T) {
	testshared.NewLintRunner(t).Run("--no-config", "-v", "-p", "style,bugs").
		ExpectOutputContains("Active presets: [bugs style]")
}

func TestAbsPathDirAnalysis(t *testing.T) {
	dir := filepath.Join("testdata_etc", "abspath") // abs paths don't work with testdata dir
	absDir, err := filepath.Abs(dir)
	assert.NoError(t, err)

	r := testshared.NewLintRunner(t).Run("--print-issued-lines=false", "--no-config", "-Egolint", absDir)
	r.ExpectHasIssue("`if` block ends with a `return` statement")
}

func TestAbsPathFileAnalysis(t *testing.T) {
	dir := filepath.Join("testdata_etc", "abspath", "with_issue.go") // abs paths don't work with testdata dir
	absDir, err := filepath.Abs(dir)
	assert.NoError(t, err)

	r := testshared.NewLintRunner(t).Run("--print-issued-lines=false", "--no-config", "-Egolint", absDir)
	r.ExpectHasIssue("`if` block ends with a `return` statement")
}

func TestDisallowedOptionsInConfig(t *testing.T) {
	type tc struct {
		cfg    string
		option string
	}

	cases := []tc{
		{
			cfg: `
				ruN:
					Args:
						- 1
			`,
		},
		{
			cfg: `
				run:
					CPUProfilePath: path
			`,
			option: "--cpu-profile-path=path",
		},
		{
			cfg: `
				run:
					MemProfilePath: path
			`,
			option: "--mem-profile-path=path",
		},
		{
			cfg: `
				run:
					Verbose: true
			`,
			option: "-v",
		},
	}

	r := testshared.NewLintRunner(t)
	for _, c := range cases {
		// Run with disallowed option set only in config
		r.RunWithYamlConfig(c.cfg, getCommonRunArgs()...).ExpectExitCode(exitcodes.Failure)

		if c.option == "" {
			continue
		}

		args := []string{c.option, "--fast"}

		// Run with disallowed option set only in command-line
		r.Run(withCommonRunArgs(args...)...).ExpectExitCode(exitcodes.Success)

		// Run with disallowed option set both in command-line and in config
		r.RunWithYamlConfig(c.cfg, withCommonRunArgs(args...)...).ExpectExitCode(exitcodes.Failure)
	}
}
