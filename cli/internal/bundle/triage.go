package bundle

// Cost and effort (spec 1.2). A debt or bug may carry two numbers from 0 to
// 9: `cost`, what it costs the project while it stays, and `effort`, the work
// it would take to close it. Each step of either scale is a multiple of the
// one below, so a register sorts by them and `cost − effort` compares harm
// against work. F16 checks their form; the soft checks ask that a cost be
// explained in `# Cost`, and count the entries nobody has rated yet.

import (
	"fmt"
	"regexp"
	"strings"
)

// scoreRe is a cost or effort: one digit.
var scoreRe = regexp.MustCompile(`^[0-9]$`)

// noScore is a cost or effort an entry does not carry.
const noScore = -1

// readScore reads a debt's or bug's `cost` or `effort`: its value, or noScore
// when it is absent or empty. Any other value is F16.
func readScore(rel, field string, v any, errs *[]string) int {
	switch t := v.(type) {
	case nil:
		return noScore
	case []string:
		if len(t) == 0 {
			return noScore // `cost:` with nothing after it: an empty value is a missing one
		}
		*errs = append(*errs, fmt.Sprintf("%s: `%s` must be one whole number from 0 to 9, not a list (F16)", rel, field))
	case string:
		if s := strings.TrimSpace(t); s == "" {
			return noScore
		} else if scoreRe.MatchString(s) {
			return int(s[0] - '0')
		}
		hint := "see SPEC.md, *Cost and effort*, for what each level means"
		if strings.Contains(t, "#") {
			hint = "a `#` after a value is part of it: put the comment on a line of its own"
		}
		*errs = append(*errs, fmt.Sprintf("%s: `%s` must be one whole number from 0 to 9, got %q — %s (F16)", rel, field, t, hint))
	}
	return noScore
}

// triage is what the soft checks read from one debt or bug.
type triage struct {
	rel, status  string
	cost, effort int
	costText     bool // a non-empty `# Cost`
}

// checkTriage is the soft half of cost and effort: a cost with no
// `# Cost` to explain it, and, once per register, the open and accepted
// entries that have no cost yet.
func checkTriage(register, noun string, entries []triage, warns *[]string) {
	untriaged := 0
	for _, e := range entries {
		if e.status != "open" && e.status != "accepted" {
			continue
		}
		if e.cost == noScore {
			untriaged++
			continue
		}
		if !e.costText {
			*warns = append(*warns, fmt.Sprintf("%s: `cost: %d` with no `# Cost` to say why — write what the %s costs while it stays, so the number can be checked", e.rel, e.cost, noun))
		}
	}
	if untriaged > 0 {
		*warns = append(*warns, fmt.Sprintf("%s/: %d open or accepted %s(s) with no `cost` — rate each from 0 to 9 (SPEC.md, *Cost and effort*); `fdf %s --untriaged` lists them", register, untriaged, noun, noun))
	}
}
