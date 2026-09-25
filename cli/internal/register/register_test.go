package register

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// bundleRoot returns a temp bundle root pinning fdf_version pin, or the
// current version when none is given.
func bundleRoot(t *testing.T, pin ...string) string {
	t.Helper()
	root := t.TempDir()
	v := scaffold.CurrentVersion()
	if len(pin) > 0 {
		v = pin[0]
	}
	index := "---\nfdf_version: \"" + v + "\"\n---\n\n# Bundle\n\n* [Spec](/SPEC.md) - the format.\n"
	if err := os.WriteFile(filepath.Join(root, "INDEX.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

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
	root := bundleRoot(t)
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

// FILED is the UTC date: an RFC 3339 time with an offset is converted before
// it is cut to its date, and a date is shown as it is.
func TestListFiledIsTheUTCDate(t *testing.T) {
	root := bundleRoot(t)
	for id, stamp := range map[string]string{
		"late-evening": "2026-09-16T23:30:00-05:00",
		"early-hours":  "2026-09-16T01:30:00+05:30",
		"utc":          "2026-09-16T12:00:00Z",
		"date-only":    "2026-09-16",
	} {
		os.MkdirAll(filepath.Join(root, "debts"), 0o755)
		os.WriteFile(filepath.Join(root, "debts", id+".md"), []byte("---\ntype: Debt\nstatus: open\ntitle: "+id+"\ntimestamp: "+stamp+"\n---\n\n# Gap\n\nx\n"), 0o644)
	}
	var out bytes.Buffer
	if code := Debt.List(root, "", &out); code != 0 {
		t.Fatalf("list: %d\n%s", code, out.String())
	}
	for id, want := range map[string]string{"late-evening": "2026-09-17", "early-hours": "2026-09-15", "utc": "2026-09-16", "date-only": "2026-09-16"} {
		found := false
		for _, line := range strings.Split(out.String(), "\n") {
			if f := strings.Fields(line); len(f) >= 3 && f[1] == "debts/"+id {
				found = true
				if f[2] != want {
					t.Errorf("debts/%s: FILED %s, want %s", id, f[2], want)
				}
			}
		}
		if !found {
			t.Errorf("debts/%s not listed:\n%s", id, out.String())
		}
	}
}

func TestCleanupLogsAndClearsOnlyResolved(t *testing.T) {
	root := bundleRoot(t)
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
	root := bundleRoot(t)
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
	root := bundleRoot(t)
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

// A disk that ignores case reads index.md as INDEX.md and log.md as LOG.md,
// so neither is an entry's name: `fdf bug log` would write over the log
// that records the cleared bugs.
func TestNewRefusesTheNamesOfTheReservedFiles(t *testing.T) {
	root := bundleRoot(t)
	var out bytes.Buffer
	if code := Bug.New(root, "first", nil, nil, &out); code != 0 {
		t.Fatalf("fdf bug first: exit %d\n%s", code, out.String())
	}
	log := "# Bug Log\n\n## 2026-09-01\n* **bugs/old-one** — Old one. Fixed by changes/x.\n"
	os.WriteFile(filepath.Join(root, "bugs", "LOG.md"), []byte(log), 0o644)
	for _, tc := range []struct {
		kind       Kind
		name, says string
	}{
		{Bug, "log", "log is not a slug: a disk that ignores case reads log.md as the LOG.md beside it (F3)"},
		{Bug, "index", "index is not a slug"},
		{Debt, "platform/log", "log is not a slug"},
	} {
		out.Reset()
		if code := tc.kind.New(root, tc.name, nil, nil, &out); code != 1 || !strings.Contains(out.String(), tc.says) {
			t.Errorf("fdf %s %s: exit %d, want a refusal saying %q:\n%s", strings.ToLower(tc.kind.Type), tc.name, code, tc.says, out.String())
		}
	}
	if raw, _ := os.ReadFile(filepath.Join(root, "bugs", "LOG.md")); string(raw) != log {
		t.Errorf("bugs/LOG.md is untouched:\n%s", raw)
	}
}

// An entry is a document layout files in the register. A file in a hidden
// directory, or in a directory beside an entry, has no place in the bundle:
// it is neither listed nor cleared.
func TestTheRegisterIsWhatLayoutFilesThere(t *testing.T) {
	root := bundleRoot(t)
	seed(t, root, "rounding", "resolved", "\n# Resolution\n\nRounded in changes/x.\n")
	seed(t, root, ".archive/old", "resolved", "\n# Resolution\n\nLong gone.\n")
	seed(t, root, "rounding/worked-example", "resolved", "\n# Resolution\n\nAn example.\n")
	var out bytes.Buffer
	if code := Debt.List(root, "", &out); code != 0 || !strings.Contains(out.String(), "debts/rounding ") ||
		strings.Contains(out.String(), "archive") || strings.Contains(out.String(), "worked-example") {
		t.Errorf("only debts/rounding is an entry: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := Debt.Cleanup(root, false, true, &out); code != 0 {
		t.Fatalf("cleanup: exit %d\n%s", code, out.String())
	}
	for _, rel := range []string{".archive/old.md", "rounding/worked-example.md"} {
		if _, err := os.Stat(filepath.Join(root, "debts", filepath.FromSlash(rel))); err != nil {
			t.Errorf("debts/%s is not the register's to clear:\n%s", rel, out.String())
		}
	}
}

func TestNewRejectsStatusWordAsSlug(t *testing.T) {
	root := bundleRoot(t)
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
	root := bundleRoot(t)
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
	root := bundleRoot(t)
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
	root := bundleRoot(t)
	os.MkdirAll(filepath.Join(root, "features", "venues"), 0o755)
	os.WriteFile(filepath.Join(root, "features", "venues", "hours.md"), []byte("---\ntype: Feature\n---\n"), 0o644)

	var out bytes.Buffer
	if code := Bug.New(root, "late-close", []string{"features/venues/ghost"}, nil, &out); code != 1 {
		t.Fatalf("an unknown feature in --affects must be refused:\n%s", out.String())
	}
	out.Reset()
	if code := Bug.New(root, "late-close", []string{"venues/hours"}, nil, &out); code != 1 ||
		out.String() != "error: --affects names venues/hours, which is not a feature in this bundle — did you mean features/venues/hours?\n" {
		t.Fatalf("a feature ID written the 0.7 way gets its full ID suggested: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := Bug.New(root, "late-close", []string{"features/venues/hours"}, []string{"internal/hours.go"}, &out); code != 0 {
		t.Fatalf("new bug: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "bugs", "late-close.md"))
	for _, want := range []string{"type: Bug", "status: open", "affects: [features/venues/hours]", "resource: [internal/hours.go]", "# Symptom", "# Expected"} {
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
	root := bundleRoot(t)
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
// Every placeholder is one whole line: deleting an optional section's TODO
// line leaves nothing of it behind.
func TestTemplatePlaceholdersAreWholeLines(t *testing.T) {
	root := bundleRoot(t)
	for _, k := range []Kind{Debt, Bug} {
		var out bytes.Buffer
		if code := k.New(root, "gap", nil, nil, &out); code != 0 {
			t.Fatalf("fdf %s: exit %d\n%s", k.noun, code, out.String())
		}
		raw, _ := os.ReadFile(filepath.Join(root, k.Dir, "gap.md"))
		lines := strings.Split(string(raw), "\n")
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i-1], "TODO —") && strings.TrimSpace(lines[i]) != "" {
				t.Errorf("%s: %q continues a TODO line:\n%s", k.Type, lines[i], raw)
			}
		}
	}
}

func TestNewTakesTheFullID(t *testing.T) {
	root := bundleRoot(t)
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
	root := bundleRoot(t)
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

// Clearing a group's last entry leaves nothing for the group to hold: its
// index would list nothing (a validation warning) and still be linked from
// the register's. The group goes — index, directory, listing — and a dry run
// says so first. A group that holds anything more, its log say, stays.
func TestCleanupRemovesAGroupItEmpties(t *testing.T) {
	root := bundleRoot(t)
	var out bytes.Buffer
	for _, id := range []string{"platform/slow-boot", "venues/stale-cache", "venues/slow-hours"} {
		if code := Debt.New(root, id, nil, nil, &out); code != 0 {
			t.Fatalf("fdf debt %s: exit %d\n%s", id, code, out.String())
		}
	}
	seed(t, root, "platform/slow-boot", "resolved", "\n# Resolution\n\nCached in changes/x.\n")
	seed(t, root, "venues/slow-hours", "resolved", "\n# Resolution\n\nIndexed in changes/y.\n")
	os.WriteFile(filepath.Join(root, "debts", "platform", "slow-boot.log.md"), []byte("---\ntype: Log\n---\n\n## 2026-09-16\n\n* note\n"), 0o644)
	// macOS Finder leaves one in any directory it has shown.
	os.WriteFile(filepath.Join(root, "debts", "platform", ".DS_Store"), []byte("finder"), 0o644)
	read := func(rel string) string {
		raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		return string(raw)
	}

	out.Reset()
	if code := Debt.Cleanup(root, true, false, &out); code != 0 {
		t.Fatalf("dry run: exit %d\n%s", code, out.String())
	}
	want := "would remove debts/platform/INDEX.md and debts/platform/: the group would hold no entry\n" +
		"  would unlist debts/platform/ from debts/INDEX.md\n"
	if !strings.Contains(out.String(), want) || strings.Contains(out.String(), "debts/venues/:") {
		t.Errorf("the dry run names the emptied group, and only it:\n%s\nwant:\n%s", out.String(), want)
	}
	if !strings.Contains(read("debts/INDEX.md"), "/debts/platform/INDEX.md") {
		t.Fatal("a dry run changes nothing")
	}

	out.Reset()
	if code := Debt.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "platform")); err == nil {
		t.Errorf("the emptied group should be gone:\n%s", out.String())
	}
	if top := read("debts/INDEX.md"); strings.Contains(top, "platform") || !strings.Contains(top, "/debts/venues/INDEX.md") {
		t.Errorf("debts/INDEX.md unlists the emptied group, and only it:\n%s", top)
	}
	for _, want := range []string{
		"removed debts/platform/INDEX.md and debts/platform/: the group holds no entry\n",
		"unlisted debts/platform/ from debts/INDEX.md\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("cleanup should say %q:\n%s", want, out.String())
		}
	}

	// A group with its own log keeps it, and the index beside it.
	if code := Bug.New(root, "ui/label", nil, nil, &out); code != 0 {
		t.Fatalf("fdf bug: exit %d\n%s", code, out.String())
	}
	seedBug(t, root, "ui/label", "resolved", "\n# Resolution\n\nRepaired by changes/label-fix.\n")
	os.WriteFile(filepath.Join(root, "bugs", "ui", "LOG.md"), []byte("# UI — log\n\n## 2026-09-16\n* A decision.\n"), 0o644)
	out.Reset()
	if code := Bug.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("bug cleanup: exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "bugs", "ui", "LOG.md")); err != nil {
		t.Fatal("a group's log is never removed")
	}
	if !strings.Contains(out.String(), "note: bugs/ui/INDEX.md lists nothing now, but bugs/ui/ holds more than its entries — both stay\n") {
		t.Errorf("the kept group is named:\n%s", out.String())
	}
}

// The commands work on spec 1.0 bundles: a 0.x bundle is upgraded with
// `fdf migrate` before its registers are filed, listed or cleared.
func TestRegistersPointA0xBundleAtMigrate(t *testing.T) {
	for _, tc := range []struct {
		k   Kind
		pin string
	}{
		{Bug, "0.7"},
		{Debt, "0.6"},
	} {
		root := bundleRoot(t, tc.pin)
		for name, run := range map[string]func(*bytes.Buffer) int{
			"file":    func(out *bytes.Buffer) int { return tc.k.New(root, "late-close", nil, nil, out) },
			"list":    func(out *bytes.Buffer) int { return tc.k.List(root, "", out) },
			"cleanup": func(out *bytes.Buffer) int { return tc.k.Cleanup(root, false, false, out) },
		} {
			var out bytes.Buffer
			want := "error: this bundle pins fdf_version " + tc.pin + "; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n"
			if code := run(&out); code != 1 || out.String() != want {
				t.Errorf("%s %s on a %s bundle: exit %d\n got: %q\nwant: %q", tc.k.noun, name, tc.pin, code, out.String(), want)
			}
		}
		if _, err := os.Stat(filepath.Join(root, tc.k.Dir)); err == nil {
			t.Errorf("a refused %s must write nothing, %s/ included", tc.k.noun, tc.k.Dir)
		}
	}
}

// Entries file in groups nested to any depth, each new group listed in its
// parent's index; a cleanup that empties a nested group removes it, and a
// parent it empties in turn, listings and all. A group that holds anything
// more stays.
func TestNestedGroupsAreFiledAndClearedAtAnyDepth(t *testing.T) {
	root := bundleRoot(t)
	var out bytes.Buffer
	for _, id := range []string{"platform/payments/rounding", "platform/payments/refunds/partial", "venues/slow-hours"} {
		if code := Debt.New(root, id, nil, nil, &out); code != 0 {
			t.Fatalf("fdf debt %s: exit %d\n%s", id, code, out.String())
		}
	}
	read := func(rel string) string {
		raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		return string(raw)
	}
	for rel, want := range map[string]string{
		"debts/INDEX.md":                           "* [Platform](/debts/platform/INDEX.md) - debts in platform.\n",
		"debts/platform/INDEX.md":                  "# Platform\n\n* [Payments](/debts/platform/payments/INDEX.md) - debts in payments.\n",
		"debts/platform/payments/INDEX.md":         "* [Refunds](/debts/platform/payments/refunds/INDEX.md) - debts in refunds.\n",
		"debts/platform/payments/refunds/INDEX.md": "# Refunds\n\n* [Partial](/debts/platform/payments/refunds/partial.md) - debt.\n",
	} {
		if !strings.Contains(read(rel), want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, read(rel))
		}
	}

	seed(t, root, "platform/payments/rounding", "resolved", "\n# Resolution\n\nRounded in changes/x.\n")
	seed(t, root, "platform/payments/refunds/partial", "resolved", "\n# Resolution\n\nRefunded in changes/y.\n")
	out.Reset()
	if code := Debt.Cleanup(root, true, false, &out); code != 0 {
		t.Fatalf("dry run: exit %d\n%s", code, out.String())
	}
	want := "would remove debts/platform/payments/refunds/INDEX.md and debts/platform/payments/refunds/: the group would hold no entry\n" +
		"would remove debts/platform/payments/INDEX.md and debts/platform/payments/: the group would hold no entry\n" +
		"would remove debts/platform/INDEX.md and debts/platform/: the group would hold no entry\n" +
		"  would unlist debts/platform/ from debts/INDEX.md\n"
	if !strings.Contains(out.String(), want) {
		t.Errorf("the dry run names every group it would empty, deepest first:\n%s\nwant:\n%s", out.String(), want)
	}
	out.Reset()
	if code := Debt.Cleanup(root, false, false, &out); code != 0 {
		t.Fatalf("cleanup: exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "debts", "platform")); err == nil {
		t.Errorf("debts/platform/ holds no entry once both are cleared:\n%s", out.String())
	}
	if top := read("debts/INDEX.md"); strings.Contains(top, "platform") || !strings.Contains(top, "/debts/venues/INDEX.md") {
		t.Errorf("debts/INDEX.md unlists the emptied group, and only it:\n%s", top)
	}
	if strings.Contains(out.String(), "unlisted debts/platform/payments/") {
		t.Errorf("a group whose parent goes too is not unlisted from it:\n%s", out.String())
	}
}
