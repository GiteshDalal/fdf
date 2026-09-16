package debt

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
	if code := List(root, "", &out); code != 0 {
		t.Fatalf("list: %d", code)
	}
	s := out.String()
	for _, want := range []string{"STATUS", "one", "two", "three", "1 open, 1 accepted, 1 resolved"} {
		if !strings.Contains(s, want) {
			t.Fatalf("list output missing %q:\n%s", want, s)
		}
	}

	out.Reset()
	List(root, "open", &out)
	if s := out.String(); !strings.Contains(s, "debts/one") || strings.Contains(s, "debts/two") {
		t.Fatalf("--open should show only open debts:\n%s", s)
	}

	out.Reset()
	List(root, "accepted", &out)
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
	if code := Cleanup(root, true, false, &out); code != 0 {
		t.Fatalf("dry run: %d", code)
	}
	if !strings.Contains(out.String(), "would remove") {
		t.Fatalf("dry run should not promise removal:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "three.md")); err != nil {
		t.Fatal("dry run must change nothing")
	}

	out.Reset()
	if code := Cleanup(root, false, false, &out); code != 0 {
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
	if code := Cleanup(root, false, false, &out); code != 0 || !strings.Contains(out.String(), "no resolved debts") {
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
	Cleanup(root, false, false, &out)
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
	if code := Cleanup(root, false, true, &out); code != 0 {
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
	if code := New(root, "open", &out); code != 1 {
		t.Fatal("a status word is not a slug")
	}
	if !strings.Contains(out.String(), "--open") {
		t.Fatalf("should suggest the flag:\n%s", out.String())
	}
}
