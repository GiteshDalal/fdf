package register

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seed(t *testing.T, root, id, status, extra string) {
	t.Helper()
	p := filepath.Join(root, "debts", filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: Debt\nstatus: " + status + "\ntitle: " + id + " title\n" +
		"timestamp: 2026-09-16T00:00:00Z\n---\n\n# Gap\n\nSomething is missing.\n" + extra
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListFiltersAndCounts(t *testing.T) {
	root := t.TempDir()
	seed(t, root, "one", "open", "")
	seed(t, root, "two", "accepted", "\n# Rationale\n\nNot worth it.\n")
	seed(t, root, "three", "resolved", "\n# Resolution\n\nDone in changes/x.\n")

	var out bytes.Buffer
	if code := Debt.List(root, "", &out); code != 0 {
		t.Fatalf("list: %d", code)
	}
	s := out.String()
	for _, want := range []string{"STATUS", "one", "two", "three", "1 open, 1 accepted, 1 resolved"} {
		if !strings.Contains(s, want) {
			t.Fatalf("list output missing %q:\n%s", want, s)
		}
	}

	out.Reset()
	Debt.List(root, "open", &out)
	if s := out.String(); !strings.Contains(s, "debts/one") || strings.Contains(s, "debts/two") {
		t.Fatalf("--open should show only open debts:\n%s", s)
	}

	out.Reset()
	Debt.List(root, "accepted", &out)
	if s := out.String(); !strings.Contains(s, "debts/two") || strings.Contains(s, "debts/three") {
		t.Fatalf("--accepted should show only accepted debts:\n%s", s)
	}
}

func TestCleanupLogsAndClearsOnlyResolved(t *testing.T) {
	root := t.TempDir()
	seed(t, root, "one", "open", "")
	seed(t, root, "two", "accepted", "\n# Rationale\n\nNot worth it.\n")
	seed(t, root, "three", "resolved", "\n# Resolution\n\nUnified in changes/config.\n")
	// A resolved debt's own log sibling goes with it.
	logSib := filepath.Join(root, "debts", "three.log.md")
	os.WriteFile(logSib, []byte("---\ntype: Log\n---\n\n## 2026-09-16\n\nnote\n"), 0o644)

	var out bytes.Buffer
	if code := Debt.Cleanup(root, true, false, &out); code != 0 {
		t.Fatalf("dry run: %d", code)
	}
	if !strings.Contains(out.String(), "would remove") {
		t.Fatalf("dry run should not promise removal:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "three.md")); err != nil {
		t.Fatal("dry run must change nothing")
	}

	out.Reset()
	if code := Debt.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "three.md")); err == nil {
		t.Fatal("resolved debt should be cleared")
	}
	if _, err := os.Stat(logSib); err == nil {
		t.Fatal("the debt's log sibling should go with it")
	}
	for _, keep := range []string{"one.md", "two.md"} {
		if _, err := os.Stat(filepath.Join(root, "debts", keep)); err != nil {
			t.Fatalf("%s must survive cleanup: %v", keep, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "debts", "LOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Unified in changes/config") {
		t.Fatalf("resolution must reach the log:\n%s", raw)
	}

	out.Reset()
	if code := Debt.Cleanup(root, false, false, &out); code != 0 || !strings.Contains(out.String(), "no resolved debts") {
		t.Fatalf("second cleanup should be a no-op: %d %q", code, out.String())
	}
}

// The log is newest-first like every FDF log, so a later run's heading must
// land above an earlier one rather than being appended to the bottom.
func TestCleanupLogStaysNewestFirst(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "debts"), 0o755)
	os.WriteFile(filepath.Join(root, "debts", "LOG.md"),
		[]byte("# Debt Log\n\n## 2020-01-01\n* **debts/ancient** — old one. done.\n"), 0o644)
	seed(t, root, "fresh", "resolved", "\n# Resolution\n\nClosed today.\n")

	var out bytes.Buffer
	Debt.Cleanup(root, false, false, &out)
	raw, _ := os.ReadFile(filepath.Join(root, "debts", "LOG.md"))
	s := string(raw)
	newIdx, oldIdx := strings.Index(s, "Closed today."), strings.Index(s, "## 2020-01-01")
	if newIdx < 0 || oldIdx < 0 || newIdx > oldIdx {
		t.Fatalf("new entry must sit above the older heading:\n%s", s)
	}
}

func TestCleanupNoLogSkipsTheLog(t *testing.T) {
	root := t.TempDir()
	seed(t, root, "gone", "resolved", "\n# Resolution\n\nDone.\n")
	var out bytes.Buffer
	if code := Debt.Cleanup(root, false, true, &out); code != 0 {
		t.Fatalf("cleanup --no-log: %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "LOG.md")); err == nil {
		t.Fatal("--no-log must not write debts/LOG.md")
	}
	if !strings.Contains(out.String(), "--no-log") {
		t.Fatalf("report should say nothing was logged:\n%s", out.String())
	}
}

func TestNewRejectsStatusWordAsSlug(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if code := Debt.New(root, "open", nil, nil, &out); code != 1 {
		t.Fatal("a status word is not a slug")
	}
	if !strings.Contains(out.String(), "--open") {
		t.Fatalf("should suggest the flag:\n%s", out.String())
	}
}

// A resolution is prose, wrapped like any other: the log gets its whole first
// paragraph, never a line cut off mid-sentence.
func TestCleanupLogsTheWholeFirstParagraph(t *testing.T) {
	root := t.TempDir()
	seed(t, root, "wrapped", "resolved", "\n# Resolution\n\nClosed by changes/config, which moved every\nhandler onto the shared loader.\n\nA second paragraph stays out of the log.\n")
	var out bytes.Buffer
	if code := Debt.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "debts", "LOG.md"))
	if !strings.Contains(string(raw), "Closed by changes/config, which moved every handler onto the shared loader.") {
		t.Fatalf("the log must carry the whole first paragraph:\n%s", raw)
	}
	if strings.Contains(string(raw), "second paragraph") {
		t.Fatalf("only the first paragraph belongs in the log:\n%s", raw)
	}
}

func seedBug(t *testing.T, root, id, status, extra string) {
	t.Helper()
	p := filepath.Join(root, "bugs", filepath.FromSlash(id)+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: Bug\nstatus: " + status + "\ntitle: " + id + " title\n" +
		"timestamp: 2026-09-23T00:00:00Z\n---\n\n# Symptom\n\nIt breaks.\n\n# Expected\n\nIt works.\n" + extra
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBugRegisterListsAndClearsBugsOnly(t *testing.T) {
	root := t.TempDir()
	seedBug(t, root, "crash", "open", "")
	seedBug(t, root, "ui/label", "resolved", "\n# Resolution\n\nRepaired by changes/label-fix.\n")
	seed(t, root, "gap", "open", "") // a debt is not a bug

	var out bytes.Buffer
	Bug.List(root, "", &out)
	s := out.String()
	for _, want := range []string{"BUG", "bugs/crash", "bugs/ui/label", "1 open, 0 accepted, 1 resolved", "fdf bug --cleanup"} {
		if !strings.Contains(s, want) {
			t.Fatalf("bug list missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "debts/gap") {
		t.Fatalf("the bug register must not list debts:\n%s", s)
	}

	out.Reset()
	if code := Bug.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "bugs", "LOG.md"))
	// The ID leads the line in bold: F10 finds a cleared bug by it.
	if !strings.Contains(string(raw), "* **bugs/ui/label** — ui/label title. Repaired by changes/label-fix.") {
		t.Fatalf("bugs/LOG.md entry malformed:\n%s", raw)
	}
	if !strings.HasPrefix(string(raw), "# Bug Log") {
		t.Fatalf("bugs/LOG.md needs its own title:\n%s", raw)
	}
}

func TestNewBugScaffoldsAffectsAndChecksThem(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "venues"), 0o755)
	os.WriteFile(filepath.Join(root, "venues", "hours.md"), []byte("---\ntype: Feature\n---\n"), 0o644)

	var out bytes.Buffer
	if code := Bug.New(root, "late-close", []string{"venues/ghost"}, nil, &out); code != 1 {
		t.Fatalf("an unknown feature in --affects must be refused:\n%s", out.String())
	}
	out.Reset()
	if code := Bug.New(root, "late-close", []string{"venues/hours"}, []string{"internal/hours.go"}, &out); code != 0 {
		t.Fatalf("new bug: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "bugs", "late-close.md"))
	for _, want := range []string{"type: Bug", "status: open", "affects: [venues/hours]", "resource: [internal/hours.go]", "# Symptom", "# Expected"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("scaffold missing %q:\n%s", want, raw)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "bugs", "INDEX.md")); err != nil {
		t.Fatal("scaffolding a bug must create bugs/INDEX.md")
	}
	if !strings.Contains(out.String(), "fdf fix --from bugs/late-close") || !strings.Contains(out.String(), "fdf change --from bugs/late-close") {
		t.Fatalf("next step should name both repair routes:\n%s", out.String())
	}
	if strings.Contains(out.String(), "set `affects`") || strings.Contains(out.String(), "set `resource`") {
		t.Fatalf("next step must not ask for what was already given:\n%s", out.String())
	}
}
