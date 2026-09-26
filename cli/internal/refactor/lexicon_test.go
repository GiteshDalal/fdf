package refactor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
)

// lexiconBundle is a delivered feature whose scenario name uses a banned
// word, joined in every place a name is a join — the Gherkin, slug.test.md,
// a task's acceptance and a done Change's declaration — plus a longer
// scenario that starts with the same words, and prose that exercises the
// replacement rules.
func lexiconBundle(t *testing.T) string {
	root := fixture(t, "valid-bugs")
	write(t, root, "DOMAIN.md", `---
type: Context
title: Domain Language
description: Filled.
timestamp: 2026-09-23T00:00:00Z
---

# Terms

## Venue
A physical location where a merchant sells.
- instead-of: store, shop
- except: data store

## Order
A request for Products.
- instead-of: item order
`)
	f := read(t, root, "features/venues/opening-hours.md")
	f = strings.Replace(f, "Scenario: Venue owner sets opening hours", "Scenario: Shop owner sets hours", 1)
	f += "\n```gherkin\nScenario: Shop owner sets hours twice\n  Given a Venue\n  When the owner taps \"Store hours\"\n  Then it works\n```\n"
	write(t, root, "features/venues/opening-hours.md", f)
	write(t, root, "features/venues/opening-hours.test.md", "---\ntype: Test\ntitle: Tests\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Test Cases\n\n## Shop owner sets hours\n\n`go test ./... -run TestA`\n\n## Shop owner sets hours twice\n\n`go test ./... -run TestB`\n")
	write(t, root, "features/venues/opening-hours/01-build.md", "---\ntype: Task\nstatus: done\ntitle: Build it\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Objective\n\nBuild it for a store owner.\n\n# Acceptance\n\n- Scenario \"Shop owner sets hours\" passes.\n")
	write(t, root, "features/venues/opening-hours.spec.md", "---\ntype: Spec\ntitle: Design\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Approach\n\nStores keep hours in the data store. Shops open late.\nWe discussed *store* and chose Venue.\n")
	for _, id := range []string{"changes/closed-hours-fix", "changes/old-fix"} {
		write(t, root, id+".md", strings.ReplaceAll(read(t, root, id+".md"), "Venue owner sets opening hours", "Shop owner sets hours"))
	}
	for _, id := range []string{"bugs/hours-off-by-one", "bugs/closed-hours-shown-open"} {
		write(t, root, id+".md", strings.ReplaceAll(read(t, root, id+".md"), "Venue owner sets opening hours", "Shop owner sets hours"))
	}
	validates(t, root) // the fixture is valid before the sweep: only F12 warnings
	return root
}

func lexicon(t *testing.T, root string, opts LexiconOptions) string {
	t.Helper()
	var out bytes.Buffer
	if code := Lexicon(root, opts, &out); code != 0 {
		t.Fatalf("fdf lexicon %+v: exit %d\n%s", opts, code, out.String())
	}
	return out.String()
}

func TestLexiconReportsPlacesAndNames(t *testing.T) {
	root := lexiconBundle(t)
	out := lexicon(t, root, LexiconOptions{})
	for _, want := range []string{
		`"shop" → "Venue"`,
		"features/venues/opening-hours.spec.md:9:",
		"[italic — a mention? put it in a code span]",
		"[a label quoted in a Gherkin step",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("report missing %q:\n%s", want, out)
		}
	}
}

func TestLexiconFixNeedsATerm(t *testing.T) {
	root := lexiconBundle(t)
	var out bytes.Buffer
	if code := Lexicon(root, LexiconOptions{Fix: true}, &out); code != 2 || !strings.Contains(out.String(), "one term at a time") {
		t.Fatalf("an unscoped --fix is a usage error (exit 2): %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "`fdf lexicon --term <Term> --fix --dry-run`") {
		t.Fatalf("the refusal names the command that works:\n%s", out.String())
	}
}

// The report ends with the next step, and that step is a command that runs:
// a sweep is one term at a time, so it names --term.
func TestLexiconReportEndsWithACommandThatRuns(t *testing.T) {
	root := lexiconBundle(t)
	if out := lexicon(t, root, LexiconOptions{}); !strings.Contains(out, "Then, one term at a time:\n`fdf lexicon --term <Term> --fix --dry-run`, review, and `--fix`.") {
		t.Fatalf("the closing hint must name --term:\n%s", out)
	}
	if out := lexicon(t, root, LexiconOptions{Term: "Venue"}); !strings.Contains(out, "`fdf lexicon --term Venue --fix --dry-run`") {
		t.Fatalf("with --term, the hint names that term:\n%s", out)
	}
}

func TestLexiconFixRenamesScenarioJoinsAndKeepsTheBundleValid(t *testing.T) {
	root := lexiconBundle(t)
	out := lexicon(t, root, LexiconOptions{Term: "Venue", Fix: true})
	if !strings.Contains(out, `scenario renamed across its joins: "Shop owner sets hours" → "Venue owner sets hours"`) {
		t.Fatalf("the scenario rename is reported:\n%s", out)
	}
	for _, c := range []struct{ rel, want string }{
		{"features/venues/opening-hours.md", "Scenario: Venue owner sets hours\n"},
		{"features/venues/opening-hours.md", "Scenario: Venue owner sets hours twice"},
		{"features/venues/opening-hours.test.md", "## Venue owner sets hours\n"},
		{"features/venues/opening-hours.test.md", "## Venue owner sets hours twice"},
		{"features/venues/opening-hours/01-build.md", `Scenario "Venue owner sets hours" passes`},
		{"features/venues/opening-hours/01-build.md", "Build it for a Venue owner."},
		{"changes/closed-hours-fix.md", "- Venue owner sets hours — "},
		{"bugs/hours-off-by-one.md", "- Venue owner sets hours — "},
		{"features/venues/opening-hours.spec.md", "Venues keep hours in the data store. Venues open late."},
		{"features/venues/opening-hours.spec.md", "We discussed *store* and chose Venue."},
		{"features/venues/opening-hours.md", `When the owner taps "Store hours"`},
	} {
		if s := read(t, root, c.rel); !strings.Contains(s, c.want) {
			t.Errorf("%s should contain %q:\n%s", c.rel, c.want, s)
		}
	}
	if s := read(t, root, "LOG.md"); !strings.Contains(s, "**Lexicon fix**: `shop` → Venue, `store` → Venue") {
		t.Fatalf("the sweep is logged, the old words in code spans:\n%s", s)
	}
	validates(t, root) // F8 and F10 still join every name
}

func TestLexiconFixMendsArticlesAndCapitals(t *testing.T) {
	root := lexiconBundle(t)
	write(t, root, "features/venues/opening-hours.spec.md", "---\ntype: Spec\ntitle: Design\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Approach\n\nAn item order is placed. A item order waits. ITEM ORDERS are rare.\n")
	lexicon(t, root, LexiconOptions{Term: "Order", Fix: true})
	if s := read(t, root, "features/venues/opening-hours.spec.md"); !strings.Contains(s, "An Order is placed. An Order waits. ORDERS are rare.") {
		t.Fatalf("articles and capitals are mended:\n%s", s)
	}
}

func TestLexiconDryRunChangesNothing(t *testing.T) {
	root := lexiconBundle(t)
	before := read(t, root, "features/venues/opening-hours.md")
	out := lexicon(t, root, LexiconOptions{Term: "Venue", Fix: true, DryRun: true})
	if !strings.Contains(out, "- Scenario: Shop owner sets hours") || !strings.Contains(out, "+ Scenario: Venue owner sets hours") {
		t.Fatalf("the dry run shows the diff:\n%s", out)
	}
	if read(t, root, "features/venues/opening-hours.md") != before {
		t.Fatal("a dry run edits nothing")
	}
}

// The counts say what they count. An INDEX.md is a file F12 reads but not a
// document, and a group's name is a directory's; a dry run counts the files it
// would change, not every file it looked in for a scenario name.
func TestLexiconCountsFilesAndDirectoryNames(t *testing.T) {
	root := lexiconBundle(t)
	write(t, root, "features/venues/INDEX.md", "# Venues\n\n* [opening-hours](opening-hours.md) - hours for each shop.\n")
	write(t, root, "features/shop-floor/INDEX.md", "# Shop-floor\n\n* [Tills](/features/shop-floor/tills.md) - tills.\n")
	lex, _ := bundle.LoadLexicon(root)
	files, dirs := map[string]bool{}, map[string]bool{}
	occ := bundle.ScanBundle(root, lex)
	for _, o := range occ {
		if strings.HasSuffix(o.Rel, "/") {
			dirs[o.Rel] = true
		} else {
			files[o.Rel] = true
		}
	}
	if !files["features/venues/INDEX.md"] || !dirs["features/shop-floor/"] {
		t.Fatalf("the test needs an index and a group name that use a banned word: %v %v", files, dirs)
	}
	want := fmt.Sprintf("F12 — %d banned word(s) in %d file(s) and %d directory name(s);", len(occ), len(files), len(dirs))
	if out := lexicon(t, root, LexiconOptions{}); !strings.Contains(out, want) || strings.Contains(out, "document(s)") {
		t.Fatalf("the report should say %q:\n%s", want, out)
	}

	out := lexicon(t, root, LexiconOptions{Term: "Venue", Fix: true, DryRun: true})
	changed := strings.Count(out, "\n--- ") + map[bool]int{true: 1}[strings.HasPrefix(out, "--- ")]
	if want := fmt.Sprintf(" in %d file(s); nothing was changed.", changed); changed == 0 || !strings.Contains(out, want) {
		t.Fatalf("the dry run counts the %d file(s) it shows, %q:\n%s", changed, want, out)
	}
}

func TestLexiconScanAgreesWithValidate(t *testing.T) {
	root := lexiconBundle(t)
	lex, _ := bundle.LoadLexicon(root)
	var v bytes.Buffer
	bundle.Validate(root, bundle.Options{Out: &v})
	for _, o := range bundle.ScanBundle(root, lex) {
		if !strings.Contains(v.String(), o.Rel) {
			t.Fatalf("validate does not report %s, which fdf lexicon finds:\n%s", o.Rel, v.String())
		}
	}
}

// A name that uses a banned word is fixed by a move, and the move fdf lexicon
// suggests is one fdf mv makes: at any depth, a feature's or a group's.
func TestLexiconSuggestsMovesThatRun(t *testing.T) {
	root := lexiconBundle(t)
	write(t, root, "features/venues/shop-hours.md", strings.Replace(read(t, root, "features/venues/opening-hours.md"), "status: done", "status: draft", 1))
	write(t, root, "features/shop-floor/INDEX.md", "# Shop-floor\n\n* [Tills](/features/shop-floor/tills.md) - tills.\n")
	write(t, root, "features/shop-floor/tills.md", strings.Replace(read(t, root, "features/venues/shop-hours.md"), "Opening hours", "Tills", -1))
	out := lexicon(t, root, LexiconOptions{})
	var moves [][2]string
	for _, line := range strings.Split(out, "\n") {
		if _, cmd, ok := strings.Cut(line, "e.g. fdf mv "); ok {
			ids := strings.Fields(cmd)
			moves = append(moves, [2]string{ids[0], ids[1]})
		}
	}
	want := map[[2]string]bool{
		{"features/venues/shop-hours", "features/venues/venue-hours"}: true,
		{"features/shop-floor", "features/venue-floor"}:               true,
	}
	for _, m := range moves {
		delete(want, m)
	}
	if len(want) > 0 {
		t.Fatalf("fdf lexicon should suggest %v:\n%s", want, out)
	}
	for _, m := range moves {
		move(t, root, m[0], m[1])
	}
	validates(t, root)
}

// fdf lexicon works on spec 1.0 bundles: a 0.x bundle is upgraded with
// `fdf migrate` first.
func TestLexiconPointsA0xBundleAtMigrate(t *testing.T) {
	root := fixture(t, "valid-bugs")
	pinTo(t, root, "0.7")
	var out bytes.Buffer
	if code := Lexicon(root, LexiconOptions{}, &out); code != 1 ||
		out.String() != "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n" {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
}
