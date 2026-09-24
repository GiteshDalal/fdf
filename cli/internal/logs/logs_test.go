package logs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
)

// fixture copies a conformance fixture's bundle into a temp dir and pins the
// clock, so every entry lands under a known date newer than the fixture's.
func fixture(t *testing.T, name string) string {
	t.Helper()
	clock = func() time.Time { return time.Date(2027, 1, 15, 9, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { clock = time.Now })
	src := filepath.Join("..", "..", "..", "testdata", name, "bundle")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(raw)
}

func logEntry(t *testing.T, root, id, entry string) string {
	t.Helper()
	var out bytes.Buffer
	if code := Append(root, id, entry, &out); code != 0 {
		t.Fatalf("fdf log %q %q: exit %d\n%s", id, entry, code, out.String())
	}
	return out.String()
}

// validates checks the bundle still passes, and that no log the test wrote
// draws even a warning.
func validates(t *testing.T, root string, written ...string) {
	t.Helper()
	var out bytes.Buffer
	if code := bundle.Validate(root, bundle.Options{Out: &out}); code != 0 {
		t.Fatalf("bundle does not validate:\n%s", out.String())
	}
	for _, rel := range written {
		if strings.Contains(out.String(), rel+":") {
			t.Errorf("%s, written by fdf log, draws a warning:\n%s", rel, out.String())
		}
	}
}

func TestFeatureEntryStartsItsLog(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	out := logEntry(t, root, "venues/opening-hours", "**Decision**: holidays follow the Venue's calendar.")
	if !strings.Contains(out, "venues/opening-hours.log.md (new log)") {
		t.Errorf("should say the log was created:\n%s", out)
	}
	got := read(t, root, "venues/opening-hours.log.md")
	for _, want := range []string{
		"type: Log\n",
		"title: Opening hours — log\n",
		"description: What happened to this feature and why, newest first.\n",
		"timestamp: 2027-01-15T09:30:00Z\n",
		"## 2027-01-15\n* **Decision**: holidays follow the Venue's calendar.\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("new log missing %q:\n%s", want, got)
		}
	}
	validates(t, root, "venues/opening-hours.log.md")
}

func TestSameDayEntriesShareOneHeadingNewestFirst(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	logEntry(t, root, "venues/opening-hours", "first")
	logEntry(t, root, "venues/opening-hours", "second")
	got := read(t, root, "venues/opening-hours.log.md")
	if n := strings.Count(got, "## 2027-01-15"); n != 1 {
		t.Errorf("want one heading for the day, got %d:\n%s", n, got)
	}
	if !strings.Contains(got, "## 2027-01-15\n* second\n* first\n") {
		t.Errorf("the newest entry of a day goes first:\n%s", got)
	}
	validates(t, root, "venues/opening-hours.log.md")
}

func TestRootEntryGoesAboveOlderDays(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	logEntry(t, root, "", "**Checkpoint**: Context documents re-read.")
	got := read(t, root, "LOG.md")
	if !strings.Contains(got, "# Bundle Update Log\n\n## 2027-01-15\n* **Checkpoint**: Context documents re-read.\n\n## 2026-09-23\n") {
		t.Errorf("root entry should head the log, above older days:\n%s", got)
	}
	validates(t, root, "LOG.md")
}

// A date written by hand ahead of UTC can already head a log; today's UTC
// entry goes below it, so the log stays newest first.
func TestEntryKeepsDateOrderBelowALaterDate(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	log := "# Bundle Update Log\n\n## 2027-01-16\n* ahead of UTC\n\n## 2027-01-10\n* older\n"
	if err := os.WriteFile(filepath.Join(root, "LOG.md"), []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
	logEntry(t, root, "", "today")
	if got, want := read(t, root, "LOG.md"), "# Bundle Update Log\n\n## 2027-01-16\n* ahead of UTC\n\n## 2027-01-15\n* today\n\n## 2027-01-10\n* older\n"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	validates(t, root, "LOG.md")

	clock = func() time.Time { return time.Date(2027, 1, 9, 9, 30, 0, 0, time.UTC) }
	logEntry(t, root, "", "oldest")
	if got := read(t, root, "LOG.md"); !strings.HasSuffix(got, "## 2027-01-10\n* older\n\n## 2027-01-09\n* oldest\n\n") {
		t.Errorf("a date older than every heading goes last:\n%s", got)
	}
	validates(t, root, "LOG.md")
}

func TestOwnersTakeTheirTasksAndTrailsEntries(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	out := logEntry(t, root, "venues/opening-hours/01-build", "task entry")
	if !strings.Contains(out, "a task has no log of its own") || !strings.Contains(out, "venues/opening-hours.log.md") {
		t.Errorf("a task's entry goes in its feature's log, with a note:\n%s", out)
	}
	out = logEntry(t, root, "venues/opening-hours.spec.md", "trail entry")
	if !strings.Contains(out, "part of venues/opening-hours's trail") {
		t.Errorf("a trail document's entry goes in its owner's log, with a note:\n%s", out)
	}
	if got := read(t, root, "venues/opening-hours.log.md"); !strings.Contains(got, "* trail entry\n* task entry\n") {
		t.Errorf("both entries should be in the feature's log:\n%s", got)
	}
	validates(t, root, "venues/opening-hours.log.md")
}

func TestRegisterChangeAndGroupLogs(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	logEntry(t, root, "bugs/closed-hours-shown-open", "**Investigated**: reproduced on staging.")
	logEntry(t, root, "changes/closed-hours-fix", "**Decision**: fix the query, not the cache.")
	logEntry(t, root, "venues", "**Group**: venues split out of payments.")
	if got := read(t, root, "bugs/closed-hours-shown-open.log.md"); !strings.Contains(got, "type: Log") || !strings.Contains(got, "this bug") {
		t.Errorf("bug log:\n%s", got)
	}
	if got := read(t, root, "changes/closed-hours-fix.log.md"); !strings.Contains(got, "this fix") {
		t.Errorf("fix log:\n%s", got)
	}
	if got := read(t, root, "venues/LOG.md"); !strings.HasPrefix(got, "# Venues — log\n\n## 2027-01-15\n* **Group**: venues split out of payments.\n") {
		t.Errorf("group log:\n%s", got)
	}
	validates(t, root, "bugs/closed-hours-shown-open.log.md", "changes/closed-hours-fix.log.md", "venues/LOG.md")
}

func TestContextEntriesGoToTheRootLog(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	out := logEntry(t, root, "DOMAIN", "**Context**: Venue gains `except: data store`.")
	if !strings.Contains(out, "logged in LOG.md") || !strings.Contains(out, "Context document") {
		t.Errorf("a Context document is logged in the root LOG.md:\n%s", out)
	}
}

func TestDraftFeatureMayHaveALogFromV07(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	os.WriteFile(filepath.Join(root, "venues", "holiday-hours.md"), []byte(draftFeature), 0o644)
	logEntry(t, root, "venues/holiday-hours", "**Drafted**: waiting on the legal review of closure notices.")
	validates(t, root, "venues/holiday-hours.log.md")
}

func TestDraftFeatureHasNoLogBeforeV07(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	index := read(t, root, "INDEX.md")
	os.WriteFile(filepath.Join(root, "INDEX.md"), []byte(strings.Replace(index, `fdf_version: "0.7"`, `fdf_version: "0.6"`, 1)), 0o644)
	os.WriteFile(filepath.Join(root, "venues", "holiday-hours.md"), []byte(draftFeature), 0o644)
	var out bytes.Buffer
	if code := Append(root, "venues/holiday-hours", "early note", &out); code != 1 {
		t.Fatalf("under a v0.6 pin a draft has no siblings (F4); want exit 1, got %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "draft") {
		t.Errorf("should explain why:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "venues", "holiday-hours.log.md")); err == nil {
		t.Error("no log may be created beside a draft under a v0.6 pin")
	}
}

// A pin's value may be single-quoted, as scaffold.Pin (and fdf validate)
// already read it; fdf log must agree, not just tolerate double quotes.
func TestDraftFeatureMayHaveALogUnderASingleQuotedPin(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	index := read(t, root, "INDEX.md")
	os.WriteFile(filepath.Join(root, "INDEX.md"), []byte(strings.Replace(index, `fdf_version: "0.7"`, `fdf_version: '0.7'`, 1)), 0o644)
	os.WriteFile(filepath.Join(root, "venues", "holiday-hours.md"), []byte(draftFeature), 0o644)
	logEntry(t, root, "venues/holiday-hours", "**Drafted**: waiting on the legal review of closure notices.")
	validates(t, root, "venues/holiday-hours.log.md")
}

const draftFeature = "---\ntype: Feature\nstatus: draft\ntitle: Holiday hours\ndescription: Close a Venue for a day.\ntimestamp: 2027-01-15\n---\n\n# Feature\n\n```gherkin\nFeature: Holiday hours\n  As a Venue owner\n  I want to close my Venue for a day\n  So that customers are not sent to a closed door\n```\n\n# Scenarios\n\n```gherkin\nScenario: A closed day shows as closed\n  Given a Venue closed on 2027-12-25\n  When a customer views its opening hours\n  Then the day shows as closed\n```\n"

func TestUnknownIDAndMissingBundle(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	var out bytes.Buffer
	if code := Append(root, "venues/nowhere", "x", &out); code != 1 || !strings.Contains(out.String(), "no document or group") {
		t.Errorf("unknown ID: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := Append(t.TempDir(), "", "x", &out); code != 1 || !strings.Contains(out.String(), "no bundle") {
		t.Errorf("no bundle: exit %d\n%s", code, out.String())
	}
}

func TestEntryShape(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                 "* plain\n",
		"- already a bullet":    "- already a bullet\n",
		"first\nsecond\n\nlast": "* first\n  second\n\n  last\n",
		"  padded  \r\n":        "* padded\n",
	} {
		if got := Entry(in); got != want {
			t.Errorf("Entry(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTouchKeepsTheTimestampForm(t *testing.T) {
	clock = func() time.Time { return time.Date(2027, 1, 15, 9, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { clock = time.Now })
	if got := touch("---\ntype: Log\ntimestamp: 2026-02-14\n---\n\n## 2026-02-14\n* x\n"); !strings.Contains(got, "timestamp: 2027-01-15\n") {
		t.Errorf("a date stays a date:\n%s", got)
	}
	if got := touch("---\ntype: Log\ntimestamp: 2026-02-14T10:00:00Z\n---\n"); !strings.Contains(got, "timestamp: 2027-01-15T09:30:00Z\n") {
		t.Errorf("a time stays a time:\n%s", got)
	}
	if got := touch("# Bundle Update Log\n"); got != "# Bundle Update Log\n" {
		t.Errorf("no frontmatter, no change: %q", got)
	}
}

func TestIsID(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	for s, want := range map[string]bool{
		"venues/opening-hours": true,
		"venues":               true,
		"DOMAIN":               true,
		"Spec approved.":       false,
		"approved":             false,
		"":                     false,
	} {
		if got := IsID(root, s); got != want {
			t.Errorf("IsID(%q) = %v, want %v", s, got, want)
		}
	}
}
