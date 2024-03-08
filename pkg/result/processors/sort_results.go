package processors

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/result"
)

// Base propose of this functionality to sort results (issues)
// produced by various linters by analyzing code. We're achieving this
// by sorting results.Issues using processor step, and chain based
// rules that can compare different properties of the Issues struct.

const (
	fileOrderName      = "file"
	linterOrderName    = "linter"
	linterSeverityName = "severity"
)

var _ Processor = (*SortResults)(nil)

type SortResults struct {
	cmps map[string][]comparator

	cfg *config.Output
}

func NewSortResults(cfg *config.Config) *SortResults {
	return &SortResults{
		cmps: map[string][]comparator{
			// For sorting we are comparing (in next order):
			// file names, line numbers, position, and finally - giving up.
			fileOrderName: {&byName{}, &byLine{}, &byColumn{}},
			// For sorting we are comparing: linter name
			linterOrderName: {&byLinter{}},
			// For sorting we are comparing: severity
			linterSeverityName: {&bySeverity{}},
		},
		cfg: &cfg.Output,
	}
}

// Process is performing sorting of the result issues.
func (sr SortResults) Process(issues []result.Issue) ([]result.Issue, error) {
	if !sr.cfg.SortResults {
		return issues, nil
	}

	if len(sr.cfg.SortOrder) == 0 {
		sr.cfg.SortOrder = []string{fileOrderName}
	}

	var cmps []comparator
	for _, name := range sr.cfg.SortOrder {
		if c, ok := sr.cmps[name]; ok {
			cmps = append(cmps, c...)
		} else {
			return nil, fmt.Errorf("unsupported sort-order name %q", name)
		}
	}

	cmp, err := mergeComparator(cmps)
	if err != nil {
		return nil, err
	}

	sort.Slice(issues, func(i, j int) bool {
		return cmp.Compare(&issues[i], &issues[j]) == Less
	})

	return issues, nil
}

func (sr SortResults) Name() string { return "sort_results" }

func (sr SortResults) Finish() {}

type compareResult int

const (
	Less compareResult = iota - 1
	Equal
	Greater
	None
)

func (c compareResult) isNeutral() bool {
	// return true if compare result is incomparable or equal.
	return c == None || c == Equal
}

func (c compareResult) String() string {
	switch c {
	case Less:
		return "Less"
	case Equal:
		return "Equal"
	case Greater:
		return "Greater"
	default:
		return "None"
	}
}

// comparator describe how to implement compare for two "issues" lexicographically
type comparator interface {
	Compare(a, b *result.Issue) compareResult
	Next() comparator
	AddNext(comparator) comparator
	fmt.Stringer
}

var (
	_ comparator = (*byName)(nil)
	_ comparator = (*byLine)(nil)
	_ comparator = (*byColumn)(nil)
	_ comparator = (*byLinter)(nil)
	_ comparator = (*bySeverity)(nil)
)

type byName struct{ next comparator }

func (cmp *byName) Next() comparator { return cmp.next }

func (cmp *byName) AddNext(c comparator) comparator {
	cmp.next = c
	return cmp
}

func (cmp *byName) Compare(a, b *result.Issue) compareResult {
	var res compareResult

	if res = compareResult(strings.Compare(a.FilePath(), b.FilePath())); !res.isNeutral() {
		return res
	}

	if next := cmp.Next(); next != nil {
		return next.Compare(a, b)
	}

	return res
}

func (cmp *byName) String() string {
	return comparatorToString("byName", cmp)
}

type byLine struct{ next comparator }

func (cmp *byLine) Next() comparator { return cmp.next }

func (cmp *byLine) AddNext(c comparator) comparator {
	cmp.next = c
	return cmp
}

func (cmp *byLine) Compare(a, b *result.Issue) compareResult {
	var res compareResult

	if res = numericCompare(a.Line(), b.Line()); !res.isNeutral() {
		return res
	}

	if next := cmp.Next(); next != nil {
		return next.Compare(a, b)
	}

	return res
}

func (cmp *byLine) String() string {
	return comparatorToString("byLine", cmp)
}

type byColumn struct{ next comparator }

func (cmp *byColumn) Next() comparator { return cmp.next }

func (cmp *byColumn) AddNext(c comparator) comparator {
	cmp.next = c
	return cmp
}

func (cmp *byColumn) Compare(a, b *result.Issue) compareResult {
	var res compareResult

	if res = numericCompare(a.Column(), b.Column()); !res.isNeutral() {
		return res
	}

	if next := cmp.Next(); next != nil {
		return next.Compare(a, b)
	}

	return res
}

func (cmp *byColumn) String() string {
	return comparatorToString("byColumn", cmp)
}

type byLinter struct{ next comparator }

func (cmp *byLinter) Next() comparator { return cmp.next }

func (cmp *byLinter) AddNext(c comparator) comparator {
	cmp.next = c
	return cmp
}

func (cmp *byLinter) Compare(a, b *result.Issue) compareResult {
	var res compareResult

	if res = compareResult(strings.Compare(a.FromLinter, b.FromLinter)); !res.isNeutral() {
		return res
	}

	if next := cmp.Next(); next != nil {
		return next.Compare(a, b)
	}

	return res
}

func (cmp *byLinter) String() string {
	return comparatorToString("byLinter", cmp)
}

type bySeverity struct{ next comparator }

func (cmp *bySeverity) Next() comparator { return cmp.next }

func (cmp *bySeverity) AddNext(c comparator) comparator {
	cmp.next = c
	return cmp
}

func (cmp *bySeverity) Compare(a, b *result.Issue) compareResult {
	var res compareResult

	if res = severityCompare(a.Severity, b.Severity); !res.isNeutral() {
		return res
	}

	if next := cmp.Next(); next != nil {
		return next.Compare(a, b)
	}

	return res
}

func (cmp *bySeverity) String() string {
	return comparatorToString("bySeverity", cmp)
}

func mergeComparator(cmps []comparator) (comparator, error) {
	if len(cmps) == 0 {
		return nil, errors.New("no comparator")
	}

	for i := 0; i < len(cmps)-1; i++ {
		cmps[i].AddNext(cmps[i+1])
	}

	return cmps[0], nil
}

func severityCompare(a, b string) compareResult {
	// The position inside the slice define the importance (lower to higher).
	classic := []string{"low", "medium", "high", "warning", "error"}

	if slices.Contains(classic, a) && slices.Contains(classic, b) {
		switch {
		case slices.Index(classic, a) > slices.Index(classic, b):
			return Greater
		case slices.Index(classic, a) < slices.Index(classic, b):
			return Less
		default:
			return Equal
		}
	}

	return compareResult(strings.Compare(a, b))
}

func numericCompare(a, b int) compareResult {
	var (
		isValuesInvalid  = a < 0 || b < 0
		isZeroValuesBoth = a == 0 && b == 0
		isEqual          = a == b
		isZeroValueInA   = b > 0 && a == 0
		isZeroValueInB   = a > 0 && b == 0
	)

	switch {
	case isZeroValuesBoth || isEqual:
		return Equal
	case isValuesInvalid || isZeroValueInA || isZeroValueInB:
		return None
	case a > b:
		return Greater
	case a < b:
		return Less
	}

	return Equal
}

func comparatorToString(name string, c comparator) string {
	s := name
	if c.Next() != nil {
		s += " > " + c.Next().String()
	}

	return s
}
