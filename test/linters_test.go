package test

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/golangci/golangci-lint/test/testshared"

	assert "github.com/stretchr/testify/require"

	yaml "gopkg.in/yaml.v2"
)

func runGoErrchk(c *exec.Cmd, t *testing.T) {
	output, err := c.CombinedOutput()
	assert.NoError(t, err, "Output:\n%s", output)

	// Can't check exit code: tool only prints to output
	assert.False(t, bytes.Contains(output, []byte("BUG")), "Output:\n%s", output)
}

func testSourcesFromDir(t *testing.T, dir string) {
	t.Log(filepath.Join(dir, "*.go"))

	findSources := func(pathPatterns ...string) []string {
		sources, err := filepath.Glob(filepath.Join(pathPatterns...))
		assert.NoError(t, err)
		assert.NotEmpty(t, sources)
		return sources
	}
	sources := findSources(dir, "*.go")

	testshared.NewLintRunner(t).Install()

	for _, s := range sources {
		s := s
		t.Run(filepath.Base(s), func(t *testing.T) {
			t.Parallel()
			testOneSource(t, s)
		})
	}
}

func TestSourcesFromTestdataWithIssuesDir(t *testing.T) {
	testSourcesFromDir(t, testdataDir)
}

func TestTypecheck(t *testing.T) {
	testSourcesFromDir(t, filepath.Join(testdataDir, "notcompiles"))
}

func TestGoimportsLocal(t *testing.T) {
	sourcePath := filepath.Join(testdataDir, "goimports", "goimports.go")
	args := []string{
		"--disable-all", "--print-issued-lines=false", "--print-linter-name=false", "--out-format=line-number",
		sourcePath,
	}
	rc := extractRunContextFromComments(t, sourcePath)
	args = append(args, rc.args...)

	cfg, err := yaml.Marshal(rc.config)
	assert.NoError(t, err)

	testshared.NewLintRunner(t).RunWithYamlConfig(string(cfg), args...).
		ExpectHasIssue("testdata/goimports/goimports.go:8: File is not `goimports`-ed")
}

func saveConfig(t *testing.T, cfg map[string]interface{}) (cfgPath string, finishFunc func()) {
	f, err := ioutil.TempFile("", "golangci_lint_test")
	assert.NoError(t, err)

	cfgPath = f.Name() + ".yml"
	err = os.Rename(f.Name(), cfgPath)
	assert.NoError(t, err)

	err = yaml.NewEncoder(f).Encode(cfg)
	assert.NoError(t, err)

	return cfgPath, func() {
		assert.NoError(t, f.Close())
		assert.NoError(t, os.Remove(cfgPath))
	}
}

func testOneSource(t *testing.T, sourcePath string) {
	goErrchkBin := filepath.Join(runtime.GOROOT(), "test", "errchk")
	args := []string{
		binName, "run",
		"--disable-all",
		"--print-issued-lines=false",
		"--print-linter-name=false",
		"--out-format=line-number",
	}

	rc := extractRunContextFromComments(t, sourcePath)
	var cfgPath string

	if rc.config != nil {
		p, finish := saveConfig(t, rc.config)
		defer finish()
		cfgPath = p
	}

	for _, addArg := range []string{"", "-Etypecheck"} {
		caseArgs := append([]string{}, args...)
		caseArgs = append(caseArgs, rc.args...)
		if addArg != "" {
			caseArgs = append(caseArgs, addArg)
		}
		if cfgPath == "" {
			caseArgs = append(caseArgs, "--no-config")
		} else {
			caseArgs = append(caseArgs, "-c", cfgPath)
		}

		caseArgs = append(caseArgs, sourcePath)

		cmd := exec.Command(goErrchkBin, caseArgs...)
		t.Log(caseArgs)
		runGoErrchk(cmd, t)
	}
}

type runContext struct {
	args   []string
	config map[string]interface{}
}

func buildConfigFromShortRepr(t *testing.T, repr string, config map[string]interface{}) {
	kv := strings.Split(repr, "=")
	assert.Len(t, kv, 2)

	keyParts := strings.Split(kv[0], ".")
	assert.True(t, len(keyParts) >= 2, len(keyParts))

	lastObj := config
	for _, k := range keyParts[:len(keyParts)-1] {
		var v map[string]interface{}
		if lastObj[k] == nil {
			v = map[string]interface{}{}
		} else {
			v = lastObj[k].(map[string]interface{})
		}

		lastObj[k] = v
		lastObj = v
	}

	lastObj[keyParts[len(keyParts)-1]] = kv[1]
}

func extractRunContextFromComments(t *testing.T, sourcePath string) *runContext {
	f, err := os.Open(sourcePath)
	assert.NoError(t, err)
	defer f.Close()

	rc := &runContext{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "//") {
			return rc
		}

		line = strings.TrimPrefix(line, "//")
		if strings.HasPrefix(line, "args: ") {
			assert.Nil(t, rc.args)
			args := strings.TrimPrefix(line, "args: ")
			assert.NotEmpty(t, args)
			rc.args = strings.Split(args, " ")
			continue
		}

		if strings.HasPrefix(line, "config: ") {
			repr := strings.TrimPrefix(line, "config: ")
			assert.NotEmpty(t, repr)
			if rc.config == nil {
				rc.config = map[string]interface{}{}
			}
			buildConfigFromShortRepr(t, repr, rc.config)
			continue
		}
	}

	return rc
}

func TestExtractRunContextFromComments(t *testing.T) {
	rc := extractRunContextFromComments(t, filepath.Join(testdataDir, "goimports", "goimports.go"))
	assert.Equal(t, []string{"-Egoimports"}, rc.args)
}

func TestGolintConsumesXTestFiles(t *testing.T) {
	dir := getTestDataDir("withxtest")
	const expIssue = "if block ends with a return statement, so drop this else and outdent its block"

	r := testshared.NewLintRunner(t)
	r.Run("--no-config", "--disable-all", "-Egolint", dir).ExpectHasIssue(expIssue)
	r.Run("--no-config", "--disable-all", "-Egolint", filepath.Join(dir, "p_test.go")).ExpectHasIssue(expIssue)
}
