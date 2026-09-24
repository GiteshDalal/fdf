package changes

import (
	"bytes"
	"os"
	"path/filepath"
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
	mk("INDEX.md", "---\nfdf_version: \"0.7\"\n---\n\n# Bundle\n\n* [Payments](/payments/INDEX.md) - payments.\n")
	mk("payments/instant-refunds.md", "---\ntype: Feature\ntitle: Instant refunds\nstatus: done\n---\n\n# Feature\n")
	return root
}

// Changes and Fixes are v0.5: under an older pin changes/ is a feature group,
// where a Change fails validation (F3), so neither command writes one there.
func TestNewRefusesAPinBeforeChanges(t *testing.T) {
	root := bundle(t)
	os.WriteFile(filepath.Join(root, "INDEX.md"), []byte("---\nfdf_version: \"0.4\"\n---\n\n# Bundle\n"), 0o644)
	for _, docType := range []string{"Change", "Fix"} {
		var out bytes.Buffer
		if code := New(root, "refund-window", docType, []string{"payments/instant-refunds"}, &out); code != 1 ||
			!strings.HasPrefix(out.String(), "error: Changes and Fixes arrived in spec v0.5, and this bundle pins fdf_version 0.4: under that pin changes/ is a feature group, and a "+docType+" written there fails validation (F3).") {
			t.Errorf("%s on a v0.4 bundle: exit %d\n%s", docType, code, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "changes")); err == nil {
		t.Error("a refused Change or Fix writes nothing")
	}
}

func TestNewFixScaffoldsRegressionSectionAndIndex(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "refund-rounding", "Fix", []string{"payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "refund-rounding.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"type: Fix", "status: draft", "affects: payments/instant-refunds",
		"# Regression cases", "## payments/instant-refunds"} {
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
	if code := New(root, "payments/refund-window", "Change", []string{"payments/instant-refunds"}, &out); code != 0 {
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

// A changes/ group is listed like every reserved directory's groups: its index
// is titled after the group and listed once in changes/INDEX.md.
func TestNewChangeGroupIsTitledAndListed(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "payments/refund-window", "Change", []string{"payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if code := New(root, "payments/refund-rounding", "Fix", []string{"payments/instant-refunds"}, &out); code != 0 {
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
		{"malformed", []string{"instant-refunds"}, "feature IDs of the form"},
		{"unknown", []string{"payments/nope"}, "not a feature in this bundle"},
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
	if code := New(root, "changes/refund-window", "Change", []string{"payments/instant-refunds"}, &out); code != 0 {
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
	New(root, "refund-rounding", "Fix", []string{"payments/instant-refunds"}, &out)
	New(root, "payments/refund-window", "Change", []string{"payments/instant-refunds"}, &out)

	out.Reset()
	if code := History(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	got := out.String()
	for _, want := range []string{"2 post-delivery document(s)", "changes/refund-rounding", "changes/payments/refund-window", "Fix", "Change"} {
		if !strings.Contains(got, want) {
			t.Errorf("history missing %q:\n%s", want, got)
		}
	}
}

func TestHistoryOnUntouchedFeatureSaysSo(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := History(root, "payments/instant-refunds", &out); code != 0 {
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
affects: payments/instant-refunds
timestamp: 2026-09-23T00:00:00Z
---

# Symptom

A 40.00 payment settled as two captures refunds 25.00.

# Expected

The whole payment is refunded.

# Violates

## payments/instant-refunds

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
		"affects: payments/instant-refunds",
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

func TestNewFromRejectsWhatIsNotABug(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := NewFrom(root, "x", "Fix", nil, "bugs/missing", &out); code != 1 || !strings.Contains(out.String(), "not a bug on the register") {
		t.Fatalf("an unknown bug must be refused: %d\n%s", code, out.String())
	}
	out.Reset()
	if code := NewFrom(root, "x", "Fix", nil, "debts/gap", &out); code != 1 || !strings.Contains(out.String(), "bugs/<slug>") {
		t.Fatalf("a non-bug ID must be refused: %d\n%s", code, out.String())
	}
}

func TestHistoryListsKnownBugsAndWhatResolvesThem(t *testing.T) {
	root := bundle(t)
	writeBug(t, root, "bugs/split-capture", splitCaptureBug)
	var out bytes.Buffer
	NewFrom(root, "split-capture-fix", "Fix", nil, "bugs/split-capture", &out)
	out.Reset()
	if code := History(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("history: %d", code)
	}
	for _, want := range []string{"resolves bugs/split-capture", "known bugs — 1 on the register", "A full refund misses the second capture"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("history missing %q:\n%s", want, out.String())
		}
	}
}
