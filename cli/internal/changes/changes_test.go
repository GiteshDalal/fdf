package changes

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func bundle(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("INDEX.md", "---\nfdf_version: \"1.0\"\n---\n\n# Bundle\n\n* [Features](/features/INDEX.md) - what the software does.\n")
	mk("features/INDEX.md", "# Features\n\n* [Payments](/features/payments/INDEX.md) - features in payments.\n")
	mk("features/payments/instant-refunds.md", "---\ntype: Feature\ntitle: Instant refunds\nstatus: done\n---\n\n# Feature\n")
	return root
}

// The commands write spec 1.0: a 0.x bundle, whose features sit at the root,
// is upgraded with `fdf migrate` before a Change or Fix is filed in it.
func TestNewPointsA0xBundleAtMigrate(t *testing.T) {
	root := bundle(t)
	os.WriteFile(filepath.Join(root, "INDEX.md"), []byte("---\nfdf_version: \"0.7\"\n---\n\n# Bundle\n"), 0o644)
	for _, docType := range []string{"Change", "Fix"} {
		var out bytes.Buffer
		if code := New(root, "refund-window", docType, []string{"features/payments/instant-refunds"}, &out); code != 1 ||
			out.String() != "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — upgrading the bundle is the user's decision, since `fdf migrate` moves its documents and rewrites references to them across the project: `fdf migrate --dry-run` shows the plan\n" {
			t.Errorf("%s on a v0.7 bundle: exit %d\n%s", docType, code, out.String())
		}
	}
	var out bytes.Buffer
	if code := History(root, "features/payments/instant-refunds", &out); code != 1 || !strings.Contains(out.String(), "`fdf migrate --dry-run` shows the plan") {
		t.Errorf("history on a v0.7 bundle: exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "changes")); err == nil {
		t.Error("a refused Change or Fix writes nothing")
	}
}

func TestNewFixScaffoldsRegressionSectionAndIndex(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "refund-rounding", "Fix", []string{"features/payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "refund-rounding.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"type: Fix", "status: draft", "affects: features/payments/instant-refunds",
		"# Regression cases", "## features/payments/instant-refunds"} {
		if !strings.Contains(body, want) {
			t.Errorf("scaffolded Fix missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# Scenario changes") {
		t.Error("a Fix must not carry a Change's declaration section")
	}
	idx, _ := os.ReadFile(filepath.Join(root, "changes", "INDEX.md"))
	if !strings.Contains(string(idx), "refund-rounding.md") {
		t.Errorf("changes/INDEX.md does not list the new fix:\n%s", idx)
	}
}

func TestNewChangeIsGroupedAndCarriesScenarioChanges(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "payments/refund-window", "Change", []string{"features/payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "payments", "refund-window.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"type: Change", "# Scenario changes", "- add:", "- modify:", "- remove:"} {
		if !strings.Contains(body, want) {
			t.Errorf("scaffolded Change missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# Regression cases") {
		t.Error("a Change must not carry a Fix's declaration section")
	}
	if _, err := os.Stat(filepath.Join(root, "changes", "payments", "INDEX.md")); err != nil {
		t.Error("a changes/ group must get its own INDEX.md")
	}
}

// The fdf-change skill deletes each scaffold line that starts `TODO —` once it
// is answered or does not apply. Every such placeholder is one whole line, so
// deleting it leaves nothing behind.
func TestTodoPlaceholdersAreWholeLines(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	for _, c := range []struct{ id, docType string }{{"refund-window", "Change"}, {"refund-rounding", "Fix"}} {
		if code := New(root, c.id, c.docType, []string{"features/payments/instant-refunds"}, &out); code != 0 {
			t.Fatalf("exit %d\n%s", code, out.String())
		}
		raw, _ := os.ReadFile(filepath.Join(root, "changes", c.id+".md"))
		lines := strings.Split(string(raw), "\n")
		for i := 1; i < len(lines); i++ {
			// A placeholder line — `TODO —` after any bullet or `key:` — must
			// not run on into the next: that line would be left behind.
			prev, line := todoLineRe.MatchString(lines[i-1]), lines[i]
			if prev && strings.TrimSpace(line) != "" && !newBlockRe.MatchString(line) {
				t.Errorf("%s: %q continues a TODO line, and deleting that line leaves it behind:\n%s", c.docType, line, raw)
			}
		}
	}
}

var (
	todoLineRe = regexp.MustCompile(`^(?:- )?(?:[a-z-]+: )?TODO —`)
	newBlockRe = regexp.MustCompile(`^(?:- |#|---|[a-z-]+:)`)
)

// A directory under changes/ is the task directory of the change named like
// it, or else a group (F3), so neither may take the other's place.
func TestNewRefusesAPlaceAChangeOrGroupHolds(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	for _, id := range []string{"refund-window", "payments/refund-rounding"} {
		if code := New(root, id, "Change", []string{"features/payments/instant-refunds"}, &out); code != 0 {
			t.Fatalf("exit %d\n%s", code, out.String())
		}
	}
	for _, c := range []struct{ id, want string }{
		{"refund-window/extra", "changes/refund-window/ is the task directory of changes/refund-window"},
		{"payments", "changes/payments/ is a group"},
	} {
		out.Reset()
		if code := New(root, c.id, "Fix", []string{"features/payments/instant-refunds"}, &out); code != 1 || !strings.Contains(out.String(), c.want) {
			t.Errorf("fdf fix %s: exit %d, want a refusal containing %q:\n%s", c.id, code, c.want, out.String())
		}
		if _, err := os.Stat(filepath.Join(root, "changes", filepath.FromSlash(c.id)+".md")); err == nil {
			t.Errorf("fdf fix %s wrote the document it refused", c.id)
		}
	}
}

// A Change from a bug whose `# Expected` is still unwritten says so on a
// line of its own, where validation's placeholder check finds it.
func TestNewChangeFromBugMarksAMissingExpectation(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/slow-refunds", "---\ntype: Bug\nstatus: open\ntitle: Slow refunds\naffects: features/payments/instant-refunds\n---\n\n# Symptom\n\nA refund takes a day.\n\n# Expected\n\nTODO — what should happen instead.\n")
	var out bytes.Buffer
	if code := NewFrom(root, "slow-refunds", "Change", nil, "bugs/slow-refunds", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "changes", "slow-refunds.md"))
	if !strings.Contains(string(raw), "\nTODO — what should happen instead.\n") || strings.Contains(string(raw), "instead: TODO") {
		t.Fatalf("the missing expectation should be a placeholder line of its own:\n%s", raw)
	}
}

// A changes/ group is listed like every register's groups: its index is
// titled after the group and listed once in changes/INDEX.md.
func TestNewChangeGroupIsTitledAndListed(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "payments/refund-window", "Change", []string{"features/payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if code := New(root, "payments/refund-rounding", "Fix", []string{"features/payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	group, _ := os.ReadFile(filepath.Join(root, "changes", "payments", "INDEX.md"))
	if want := "# Payments\n\n* [Refund window](/changes/payments/refund-window.md) - change.\n* [Refund rounding](/changes/payments/refund-rounding.md) - fix.\n"; string(group) != want {
		t.Errorf("changes/payments/INDEX.md:\n%s\nwant:\n%s", group, want)
	}
	top, _ := os.ReadFile(filepath.Join(root, "changes", "INDEX.md"))
	if n := strings.Count(string(top), "* [Payments](/changes/payments/INDEX.md) - changes and fixes in payments.\n"); n != 1 {
		t.Errorf("changes/INDEX.md should list the group once, got %d:\n%s", n, top)
	}
	for _, want := range []string{
		"wrote changes/INDEX.md",
		"wrote changes/payments/INDEX.md\n",
		`updated changes/INDEX.md (now lists "Payments")`,
		`updated changes/payments/INDEX.md (now lists "Refund window")`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should say %q:\n%s", want, out.String())
		}
	}
}

// affects is the whole link between a change and the features it touches, so
// a typo there would silently produce an orphan document.
func TestNewRejectsUnknownOrMissingAffects(t *testing.T) {
	root := bundle(t)
	for _, tc := range []struct {
		name    string
		affects []string
		want    string
	}{
		{"none", nil, "--affects is required"},
		{"old-style", []string{"payments/instant-refunds"},
			"error: --affects names payments/instant-refunds, which is not a feature in this bundle — did you mean features/payments/instant-refunds?\n"},
		{"unknown", []string{"features/payments/nope"}, "error: --affects names features/payments/nope, which is not a feature in this bundle\n"},
		{"not-a-feature", []string{"changes/refund-window"}, "error: --affects names changes/refund-window, which is not a feature in this bundle\n"},
	} {
		var out bytes.Buffer
		if code := New(root, "c-"+tc.name, "Fix", tc.affects, &out); code == 0 {
			t.Errorf("%s: expected refusal", tc.name)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("%s: want %q, got %q", tc.name, tc.want, out.String())
		}
	}
	// A missing required flag is a usage error.
	var out bytes.Buffer
	if code := New(root, "c-bare", "Fix", nil, &out); code != 2 {
		t.Errorf("no --affects: exit %d, want 2\n%s", code, out.String())
	}
}

// The full ID files the document where it says, not under changes/changes/.
func TestNewTakesTheFullID(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "changes/refund-window", "Change", []string{"features/payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "changes", "refund-window.md")); err != nil {
		t.Fatalf("changes/refund-window.md was not written:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "changes", "changes")); err == nil {
		t.Fatalf("changes/changes/ must not exist:\n%s", out.String())
	}
}

func TestHistoryFindsChangesByAffects(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	New(root, "refund-rounding", "Fix", []string{"features/payments/instant-refunds"}, &out)
	New(root, "payments/refund-window", "Change", []string{"features/payments/instant-refunds"}, &out)

	// A document in a hidden directory has no place in the bundle.
	hidden := filepath.Join(root, "changes", ".drafts", "old-idea.md")
	os.MkdirAll(filepath.Dir(hidden), 0o755)
	os.WriteFile(hidden, []byte("---\ntype: Change\ntitle: Old idea\nstatus: draft\naffects: features/payments/instant-refunds\n---\n"), 0o644)

	out.Reset()
	if code := History(root, "features/payments/instant-refunds", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	got := out.String()
	for _, want := range []string{"2 post-delivery document(s)", "changes/refund-rounding", "changes/payments/refund-window", "Fix", "Change"} {
		if !strings.Contains(got, want) {
			t.Errorf("history missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "old-idea") {
		t.Errorf("history lists a document in a hidden directory:\n%s", got)
	}
}

func TestHistoryOnUntouchedFeatureSaysSo(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := History(root, "features/payments/instant-refunds", &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "no Change or Fix names it in `affects`") {
		t.Errorf("unexpected: %s", out.String())
	}
}

func writeBug(t *testing.T, root, id, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const splitCaptureBug = `---
type: Bug
status: open
title: A full refund misses the second capture
affects: features/payments/instant-refunds
timestamp: 2026-09-23T00:00:00Z
---

# Symptom

A 40.00 payment settled as two captures refunds 25.00.

# Expected

The whole payment is refunded.

# Violates

## features/payments/instant-refunds

- Full refund of a settled payment — only the first capture is refunded

# Root cause

refund.go refunds the first capture only.
`

// A Fix from a bug takes over its analysis and names it in `resolves`; the
// scenarios it violates become the regression cases, names verbatim.
func TestNewFixFromBugTakesOverTheAnalysis(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/split-capture", splitCaptureBug)
	var out bytes.Buffer
	if code := NewFrom(root, "split-capture-fix", "Fix", nil, "bugs/split-capture", &out); code != 0 {
		t.Fatalf("new fix from bug: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "changes", "split-capture-fix.md"))
	body := string(raw)
	for _, want := range []string{
		"affects: features/payments/instant-refunds",
		"resolves: bugs/split-capture",
		"A 40.00 payment settled as two captures refunds 25.00.",
		"refund.go refunds the first capture only.",
		"- Full refund of a settled payment — TODO the command",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("fix missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "only the first capture is refunded") {
		t.Fatalf("a violation's note is not part of the scenario name:\n%s", body)
	}
	if !strings.Contains(out.String(), "resolves bugs/split-capture") {
		t.Fatalf("output should say what the work resolves:\n%s", out.String())
	}
}

// A `# Violates` entry an editor wrapped is one entry: its whole scenario name
// becomes the regression case, as validation reads it.
func TestNewFixFromBugReadsAWrappedViolation(t *testing.T) {
	root := bundle(t)
	wrapped := strings.Replace(splitCaptureBug,
		"- Full refund of a settled payment — only the first capture is refunded",
		"- Full refund of a settled\n  payment — only the first capture is refunded", 1)
	writeBug(t, root, "bugs/split-capture", wrapped)
	var out bytes.Buffer
	if code := NewFrom(root, "split-capture-fix", "Fix", nil, "bugs/split-capture", &out); code != 0 {
		t.Fatalf("new fix from bug: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "changes", "split-capture-fix.md"))
	if !strings.Contains(string(raw), "- Full refund of a settled payment — TODO the command") {
		t.Fatalf("the wrapped name should be copied whole:\n%s", raw)
	}
}

// A defect in code no feature documents has nothing for a Fix to amend: the
// capability is adopted first.
func TestNewFromBugWithoutAffectsPointsAtAdoption(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/orphan", "---\ntype: Bug\nstatus: open\ntitle: Orphan\n---\n\n# Symptom\n\nIt breaks.\n\n# Expected\n\nIt works.\n")
	var out bytes.Buffer
	if code := NewFrom(root, "orphan-fix", "Fix", nil, "bugs/orphan", &out); code != 1 {
		t.Fatalf("want refusal, got %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "fdf adopt --resource") {
		t.Fatalf("refusal should point at adoption:\n%s", out.String())
	}
}

// A Fix from a bug takes the bug's `affects`. When the bug names a feature
// wrongly, the error says so of the bug, which is where it is corrected, not
// of --affects, which nobody passed.
func TestNewFromBugNamesTheBugsOwnAffects(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/stale", "---\ntype: Bug\nstatus: open\ntitle: Stale\naffects: payments/instant-refunds\n---\n\n# Symptom\n\nIt breaks.\n\n# Expected\n\nIt works.\n")
	var out bytes.Buffer
	want := "error: bugs/stale's `affects` names payments/instant-refunds, which is not a feature in this bundle (F14); correct it there, then retry — did you mean features/payments/instant-refunds?\n"
	if code := NewFrom(root, "stale-fix", "Fix", nil, "bugs/stale", &out); code != 1 || out.String() != want {
		t.Fatalf("exit %d, want 1 saying %q:\n%s", code, want, out.String())
	}
}

func TestNewFromRejectsWhatIsNotABug(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := NewFrom(root, "x", "Fix", nil, "bugs/missing", &out); code != 1 || !strings.Contains(out.String(), "not a bug on the register") {
		t.Fatalf("an unknown bug must be refused: %d\n%s", code, out.String())
	}
	out.Reset()
	if code := NewFrom(root, "x", "Fix", nil, "debts/gap", &out); code != 1 || !strings.Contains(out.String(), "bugs/[<group>/…]<slug>") {
		t.Fatalf("a non-bug ID must be refused: %d\n%s", code, out.String())
	}
}

func TestHistoryListsKnownBugsAndWhatResolvesThem(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/split-capture", splitCaptureBug)
	var out bytes.Buffer
	NewFrom(root, "split-capture-fix", "Fix", nil, "bugs/split-capture", &out)
	out.Reset()
	if code := History(root, "features/payments/instant-refunds", &out); code != 0 {
		t.Fatalf("history: %d", code)
	}
	for _, want := range []string{"resolves bugs/split-capture", "known bugs — 1 on the register", "A full refund misses the second capture"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("history missing %q:\n%s", want, out.String())
		}
	}
}

// Changes and Fixes file flat or in groups nested to any depth, each new
// group listed in its parent's index; a bug filed in nested groups is repaired
// by its full ID.
func TestNewFilesInNestedGroupsAndFromANestedBug(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/platform/payments/split-capture", splitCaptureBug)
	var out bytes.Buffer
	if code := NewFrom(root, "platform/payments/split-capture-fix", "Fix", nil, "bugs/platform/payments/split-capture", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "platform", "payments", "split-capture-fix.md"))
	if err != nil || !strings.Contains(string(raw), "resolves: bugs/platform/payments/split-capture") {
		t.Fatalf("the Fix is filed in its groups and resolves the nested bug: %v\n%s", err, raw)
	}
	for rel, want := range map[string]string{
		"changes/INDEX.md":                   "* [Platform](/changes/platform/INDEX.md) - changes and fixes in platform.\n",
		"changes/platform/INDEX.md":          "# Platform\n\n* [Payments](/changes/platform/payments/INDEX.md) - changes and fixes in payments.\n",
		"changes/platform/payments/INDEX.md": "# Payments\n\n* [Split capture fix](/changes/platform/payments/split-capture-fix.md) - fix.\n",
	} {
		if got, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); !strings.Contains(string(got), want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, got)
		}
	}
	if !strings.Contains(out.String(), "done: Fix changes/platform/payments/split-capture-fix affects features/payments/instant-refunds\n") {
		t.Errorf("the Fix is named by its full ID:\n%s", out.String())
	}
}

// history takes a feature's full ID, at any depth, and suggests it for one
// written the 0.7 way.
func TestHistoryTakesTheFullFeatureID(t *testing.T) {
	root := bundle(t)
	os.MkdirAll(filepath.Join(root, "features", "platform", "payouts"), 0o755)
	os.WriteFile(filepath.Join(root, "features", "platform", "payouts", "weekly.md"), []byte("---\ntype: Feature\ntitle: Weekly\nstatus: done\n---\n\n# Feature\n"), 0o644)
	var out bytes.Buffer
	if code := New(root, "payout-day", "Change", []string{"features/platform/payouts/weekly"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := History(root, "features/platform/payouts/weekly", &out); code != 0 || !strings.Contains(out.String(), "changes/payout-day") {
		t.Errorf("history of a nested feature: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := History(root, "payments/instant-refunds", &out); code != 1 ||
		out.String() != "error: payments/instant-refunds is not a feature in this bundle — did you mean features/payments/instant-refunds?\n" {
		t.Errorf("a 0.7-style ID: exit %d\n%q", code, out.String())
	}
}
