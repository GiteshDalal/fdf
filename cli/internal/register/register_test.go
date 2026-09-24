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

// A new entry is listed in the index beside it — a group's index is created
// and listed on first use — and a cleared entry's listing goes with its file.
func TestNewListsTheEntryAndCleanupUnlistsIt(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	for _, id := range []string{"venues/slow-hours", "venues/stale-cache", "loose-config"} {
		if code := Debt.New(root, id, nil, nil, &out); code != 0 {
			t.Fatalf("fdf debt %s: exit %d\n%s", id, code, out.String())
		}
	}
	read := func(rel string) string {
		raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		return string(raw)
	}
	group := read("debts/venues/INDEX.md")
	if want := "# Venues\n\n* [Slow hours](/debts/venues/slow-hours.md) - debt.\n* [Stale cache](/debts/venues/stale-cache.md) - debt.\n"; group != want {
		t.Errorf("debts/venues/INDEX.md:\n%s\nwant:\n%s", group, want)
	}
	top := read("debts/INDEX.md")
	if n := strings.Count(top, "](/debts/venues/INDEX.md) - debts in venues."); n != 1 {
		t.Errorf("the group should be listed once in debts/INDEX.md, got %d:\n%s", n, top)
	}
	if !strings.Contains(top, "* [Loose config](/debts/loose-config.md) - debt.\n") {
		t.Errorf("an ungrouped debt is listed in debts/INDEX.md:\n%s", top)
	}

	seed(t, root, "venues/slow-hours", "resolved", "\n# Resolution\n\nCached in changes/x.\n")
	out.Reset()
	if code := Debt.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: exit %d\n%s", code, out.String())
	}
	if strings.Contains(read("debts/venues/INDEX.md"), "slow-hours") {
		t.Errorf("a cleared debt's listing should go:\n%s", read("debts/venues/INDEX.md"))
	}
	if !strings.Contains(out.String(), "unlisted it from debts/venues/INDEX.md") {
		t.Errorf("cleanup should say it unlisted the entry:\n%s", out.String())
	}
}

// The full ID — what `fdf log`, `fdf mv` and --from take — files the entry
// where the ID says, not a level deeper under debts/debts/.
func TestNewTakesTheFullID(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if code := Debt.New(root, "debts/loose-config", nil, nil, &out); code != 0 {
		t.Fatalf("fdf debt debts/loose-config: exit %d\n%s", code, out.String())
	}
	if code := Bug.New(root, "bugs/ui/label", nil, nil, &out); code != 0 {
		t.Fatalf("fdf bug bugs/ui/label: exit %d\n%s", code, out.String())
	}
	for _, rel := range []string{"debts/loose-config.md", "bugs/ui/label.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s was not written: %v\n%s", rel, err, out.String())
		}
	}
	for _, rel := range []string{"debts/debts", "bugs/bugs"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			t.Errorf("%s/ must not exist:\n%s", rel, out.String())
		}
	}
}

// A dry run names everything the real run removes: the entry's log goes with
// it, entries and all, and its listing leaves the index.
func TestCleanupDryRunNamesTheLogAndTheListing(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if code := Debt.New(root, "venues/slow-hours", nil, nil, &out); code != 0 {
		t.Fatalf("fdf debt: exit %d\n%s", code, out.String())
	}
	seed(t, root, "venues/slow-hours", "resolved", "\n# Resolution\n\nCached in changes/x.\n")
	logSib := filepath.Join(root, "debts", "venues", "slow-hours.log.md")
	os.WriteFile(logSib, []byte("---\ntype: Log\n---\n\n## 2026-09-16\n\n* note\n"), 0o644)

	out.Reset()
	if code := Debt.Cleanup(root, true, false, &out); code != 0 {
		t.Fatalf("dry run: exit %d\n%s", code, out.String())
	}
	for _, want := range []string{
		"would remove debts/venues/slow-hours.md — ",
		"  would remove debts/venues/slow-hours.log.md with it — its entries are not kept\n",
		"  would unlist it from debts/venues/INDEX.md\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry run should say %q:\n%s", want, out.String())
		}
	}
	if _, err := os.Stat(logSib); err != nil {
		t.Fatal("a dry run changes nothing")
	}
}
