package lintersdb

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/lint/linter"
)

func TestGetEnabledLintersSet(t *testing.T) {
	type cs struct {
		cfg  config.Linters
		name string   // test case name
		def  []string // enabled by default linters
		exp  []string // alphabetically ordered enabled linter names
	}
	cases := []cs{
		{
			cfg: config.Linters{
				Disable: []string{"megacheck"},
			},
			name: "disable all linters from megacheck",
			def:  getAllMegacheckSubLinterNames(),
		},
		{
			cfg: config.Linters{
				Disable: []string{"staticcheck"},
			},
			name: "disable only staticcheck",
			def:  getAllMegacheckSubLinterNames(),
			exp:  []string{"megacheck.{unused,gosimple}"},
		},
		{
			name: "merge into megacheck",
			def:  getAllMegacheckSubLinterNames(),
			exp:  []string{"megacheck"},
		},
		{
			name: "don't disable anything",
			def:  []string{"gofmt", "govet"},
			exp:  []string{"gofmt", "govet"},
		},
		{
			name: "enable gosec by gas alias",
			cfg: config.Linters{
				Enable: []string{"gas"},
			},
			exp: []string{"gosec"},
		},
		{
			name: "enable gosec by primary name",
			cfg: config.Linters{
				Enable: []string{"gosec"},
			},
			exp: []string{"gosec"},
		},
		{
			name: "enable gosec by both names",
			cfg: config.Linters{
				Enable: []string{"gosec", "gas"},
			},
			exp: []string{"gosec"},
		},
		{
			name: "disable gosec by gas alias",
			cfg: config.Linters{
				Disable: []string{"gas"},
			},
			def: []string{"gosec"},
		},
		{
			name: "disable gosec by primary name",
			cfg: config.Linters{
				Disable: []string{"gosec"},
			},
			def: []string{"gosec"},
		},
	}

	m := NewManager()
	es := NewEnabledSet(m, NewValidator(m), nil, nil)
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			defaultLinters := []linter.Config{}
			for _, ln := range c.def {
				defaultLinters = append(defaultLinters, *m.GetLinterConfig(ln))
			}
			els := es.build(&c.cfg, defaultLinters)
			var enabledLinters []string
			for ln, lc := range els {
				assert.Equal(t, ln, lc.Name())
				enabledLinters = append(enabledLinters, ln)
			}

			sort.Strings(enabledLinters)
			sort.Strings(c.exp)

			assert.Equal(t, c.exp, enabledLinters)
		})
	}
}
