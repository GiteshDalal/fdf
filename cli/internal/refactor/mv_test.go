package refactor

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
)

// fixture copies a conformance fixture's bundle into a temp dir, so a move
// can be made on it and the result validated.
func fixture(t *testing.T, name string) string {
	t.Helper()
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

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validates(t *testing.T, root string) string {
	t.Helper()
	var out bytes.Buffer
	if code := bundle.Validate(root, bundle.Options{Out: &out}); code != 0 {
		t.Fatalf("bundle does not validate after the move:\n%s", out.String())
	}
	return out.String()
}

func move(t *testing.T, root, from, to string) string {
	t.Helper()
	var out bytes.Buffer
	if code := Move(root, "", from, to, false, &out); code != 0 {
		t.Fatalf("fdf mv %s %s: exit %d\n%s", from, to, code, out.String())
	}
	return out.String()
}

func gone(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
		t.Fatalf("%s should have moved away", rel)
	}
}

// A mention of an ID is the ID where it ends: bare, as one of its documents,
// or as its task directory. A path that goes on past it names something
// else, code or a route's template. Where the IDs are 0.x ones, which name
// no register (routes), a bare /<id> reads as a URL's path, even for a 0.6
// group named bugs/; a 1.0 ID's /features/venues/opening-hours is the
// document.
func TestIDMentionsReadAnIDWhereItEnds(t *testing.T) {
	const old, new = "venues/opening-hours", "features/venues/opening-hours"
	for _, c := range []struct {
		old, new, text, want string
		routes               bool
	}{
		{old, new, "affects: venues/opening-hours\n", "affects: features/venues/opening-hours\n", true},
		{old, new, "See venues/opening-hours.", "See features/venues/opening-hours.", true},
		{old, new, "`venues/opening-hours.spec.md`, venues/opening-hours/01-build.md", "`features/venues/opening-hours.spec.md`, features/venues/opening-hours/01-build.md", true},
		{old, new, "its spec, venues/opening-hours.spec; its code, venues/opening-hours.spec.ts", "its spec, features/venues/opening-hours.spec; its code, venues/opening-hours.spec.ts", true},
		{old, new, "its tasks, in venues/opening-hours/.", "its tasks, in features/venues/opening-hours/.", true},
		{old, new, "from the root, /venues/opening-hours.md", "from the root, /features/venues/opening-hours.md", true},
		{old, new, "served as `GET /venues/opening-hours`", "served as `GET /venues/opening-hours`", true},
		{old, new, "`GET /venues/opening-hours/ lists them`", "`GET /venues/opening-hours/ lists them`", true},
		{old, new, "`PUT /venues/opening-hours/{id}`, `DELETE /venues/opening-hours/:id`", "`PUT /venues/opening-hours/{id}`, `DELETE /venues/opening-hours/:id`", true},
		{old, new, "the route venues/opening-hours/{id}", "the route venues/opening-hours/{id}", true},
		{old, new, "in venues/opening-hours/handler.go", "in venues/opening-hours/handler.go", true},
		{old, new, "venues/opening-hours.go, venues/opening-hours-v2", "venues/opening-hours.go, venues/opening-hours-v2", true},
		{old, new, "src/venues/opening-hours", "src/venues/opening-hours", true},
		{"bugs/crash-report", "features/bugs/crash-report", "`GET /bugs/crash-report`, and bugs/crash-report", "`GET /bugs/crash-report`, and features/bugs/crash-report", true},
		{"features/venues/opening-hours", "features/venues/hours", "see /features/venues/opening-hours, and features/venues/opening-hours.spec", "see /features/venues/hours, and features/venues/hours.spec", false},
	} {
		got := c.text
		reps := IDMentions(c.text, map[string]string{c.old: c.new}, "", nil, c.routes)
		for i := len(reps) - 1; i >= 0; i-- {
			got = got[:reps[i].Start] + reps[i].Text + got[reps[i].End:]
		}
		if got != c.want {
			t.Errorf("%q\n got %q\nwant %q", c.text, got, c.want)
		}
	}
}

func TestMoveRenamesAFeatureWithItsTrailAndRepairsEveryReference(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	// A release listing the feature, and a sample link in a code fence that
	// must stay a sample.
	write(t, root, "features/venues/opening-hours.spec.md", read(t, root, "features/venues/opening-hours.spec.md")+
		"\nSee [the plan](opening-hours.plan.md).\n\n```markdown\n[x](/features/venues/opening-hours.md)\n```\n")
	out := move(t, root, "features/venues/opening-hours", "features/venues/trading-hours")

	for _, rel := range []string{"features/venues/opening-hours.md", "features/venues/opening-hours.spec.md", "features/venues/opening-hours/01-build.md"} {
		gone(t, root, rel)
	}
	for _, rel := range []string{"features/venues/trading-hours.md", "features/venues/trading-hours.spec.md", "features/venues/trading-hours.plan.md", "features/venues/trading-hours.test.md", "features/venues/trading-hours/01-build.md"} {
		read(t, root, rel)
	}
	if s := read(t, root, "features/venues/trading-hours.plan.md"); !strings.Contains(s, "(trading-hours/01-build.md)") {
		t.Fatalf("the plan's task link must follow its task directory:\n%s", s)
	}
	if s := read(t, root, "features/venues/trading-hours.spec.md"); !strings.Contains(s, "(trading-hours.plan.md)") || !strings.Contains(s, "[x](/features/venues/opening-hours.md)") {
		t.Fatalf("a real link is repaired, a sample in a code fence is not:\n%s", s)
	}
	fix := read(t, root, "changes/closed-hours-fix.md")
	if !strings.Contains(fix, "affects: features/venues/trading-hours") || !strings.Contains(fix, "## features/venues/trading-hours") {
		t.Fatalf("a Fix's affects and declaration heading follow the move:\n%s", fix)
	}
	bug := read(t, root, "bugs/hours-off-by-one.md")
	if !strings.Contains(bug, "affects: features/venues/trading-hours") || !strings.Contains(bug, "## features/venues/trading-hours") {
		t.Fatalf("a bug's affects and # Violates heading follow the move:\n%s", bug)
	}
	if s := read(t, root, "features/venues/INDEX.md"); !strings.Contains(s, "(trading-hours.md)") {
		t.Fatalf("the group index link follows the move:\n%s", s)
	}
	if s := read(t, root, "LOG.md"); !strings.Contains(s, "**Moved**: `features/venues/opening-hours` → `features/venues/trading-hours`") {
		t.Fatalf("the move is logged:\n%s", s)
	}
	if !strings.Contains(out, "reference(s) repaired") {
		t.Fatalf("the command reports what it repaired:\n%s", out)
	}
	// The repaired count includes features/venues/INDEX.md, a file but not a document.
	if !strings.Contains(out, "  features/venues/INDEX.md: 1 reference(s) repaired") || !strings.Contains(out, " file(s); logged in LOG.md.") || strings.Contains(out, "document(s)") {
		t.Fatalf("the counts say files, INDEX.md among them:\n%s", out)
	}
	if s := read(t, root, "LOG.md"); !strings.Contains(s, " file(s)).") {
		t.Fatalf("the log line counts files too:\n%s", s)
	}
	validates(t, root)
}

func TestMoveToANewGroupMovesTheListing(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	out := move(t, root, "features/venues/opening-hours", "features/sites/opening-hours")
	if s := read(t, root, "features/venues/INDEX.md"); strings.Contains(s, "opening-hours") {
		t.Fatalf("the old group index no longer lists the feature:\n%s", s)
	}
	if s := read(t, root, "features/sites/INDEX.md"); !strings.HasPrefix(s, "# Sites\n") || !strings.Contains(s, "(opening-hours.md)") {
		t.Fatalf("the new group index lists it:\n%s", s)
	}
	// The new group is listed beside the others, as `fdf new` lists one.
	if s := read(t, root, "features/INDEX.md"); !strings.Contains(s, "* [Venues](/features/venues/INDEX.md) - features in venues.\n* [Sites](/features/sites/INDEX.md) - features in sites.\n") {
		t.Fatalf("features/INDEX.md lists the new group:\n%s", s)
	}
	if !strings.Contains(out, "group features/sites/ listed in features/INDEX.md") {
		t.Fatalf("the report says the group was listed:\n%s", out)
	}
	validates(t, root)

	// A register's new group is listed in the register's index.
	write(t, root, "bugs/INDEX.md", read(t, root, "bugs/INDEX.md")+"* [Hours off by one](/bugs/hours-off-by-one.md) - bug.\n")
	move(t, root, "bugs/hours-off-by-one", "bugs/backend/hours-off-by-one")
	if s := read(t, root, "bugs/backend/INDEX.md"); !strings.HasPrefix(s, "# Backend\n") || !strings.Contains(s, "(/bugs/backend/hours-off-by-one.md)") {
		t.Fatalf("the register group's index lists the bug:\n%s", s)
	}
	if s := read(t, root, "bugs/INDEX.md"); !strings.Contains(s, "* [Backend](/bugs/backend/INDEX.md) - bugs in backend.\n") || strings.Contains(s, "(/bugs/hours-off-by-one.md)") {
		t.Fatalf("bugs/INDEX.md lists the group, not the moved bug:\n%s", s)
	}
	validates(t, root)
}

func TestMoveToANewGroupKeepsURLsAnchorsAndMailLinksInTheListing(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "features/venues/INDEX.md", strings.Replace(read(t, root, "features/venues/INDEX.md"),
		"* [opening-hours](opening-hours.md) - a capability.",
		"* [opening-hours](opening-hours.md) - a capability. See [the site](https://example.com/hours), [the top](#top) and [the team](mailto:team@example.com).", 1))
	move(t, root, "features/venues/opening-hours", "features/sites/opening-hours")
	s := read(t, root, "features/sites/INDEX.md")
	for _, want := range []string{"(opening-hours.md)", "(https://example.com/hours)", "(#top)", "(mailto:team@example.com)"} {
		if !strings.Contains(s, want) {
			t.Fatalf("want %q in the moved listing line:\n%s", want, s)
		}
	}
	validates(t, root)
}

// A listing line identifies its own document by its FIRST link, the way
// scaffold.ListingTarget reads every other index (Unlist, Listed, ListedIn).
// A line that goes on to mention the moved document as a later link is some
// other document's listing, not the moved one's: it stays where it is, and
// only the mention inside it is repaired, like any other reference.
func TestMoveToANewGroupLeavesAnotherListingsSecondLinkRepaired(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "features/venues/INDEX.md", read(t, root, "features/venues/INDEX.md")+
		"* [Venues](/features/venues/INDEX.md) - the group; see [opening-hours](opening-hours.md) for its hours.\n")
	move(t, root, "features/venues/opening-hours", "features/sites/opening-hours")
	if s := read(t, root, "features/venues/INDEX.md"); !strings.Contains(s, "* [Venues](/features/venues/INDEX.md) - the group; see [opening-hours](../sites/opening-hours.md) for its hours.\n") {
		t.Fatalf("another document's listing stays behind, its mention of the moved document repaired:\n%s", s)
	}
	if s := read(t, root, "features/sites/INDEX.md"); strings.Contains(s, "Venues") {
		t.Fatalf("the other document's listing must not be carried off to the new index:\n%s", s)
	}
	validates(t, root)
}

func TestMoveRenamesAWholeGroup(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	move(t, root, "features/venues", "features/sites")
	gone(t, root, "features/venues")
	if s := read(t, root, "features/INDEX.md"); !strings.Contains(s, "(/features/sites/INDEX.md)") {
		t.Fatalf("features/INDEX.md follows the group:\n%s", s)
	}
	if s := read(t, root, "changes/old-fix.md"); !strings.Contains(s, "affects: features/sites/opening-hours") {
		t.Fatalf("every edge into the group follows it:\n%s", s)
	}
	validates(t, root)
}

func TestMoveRefilesADebtAsABug(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	os.RemoveAll(filepath.Join(root, "bugs"))
	write(t, root, "debts/INDEX.md", "# Debt\n\n* [Deferred batch import](deferred-batch-import.md) - deferred.\n")
	write(t, root, "debts/deferred-batch-import.md", "---\ntype: Debt\nstatus: open\ntitle: Deferred batch import\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Gap\n\nBatch import waits for the nightly job.\n")
	var out bytes.Buffer
	if code := Move(root, "", "debts/deferred-batch-import", "bugs/deferred-batch-import", false, &out); code != 0 {
		t.Fatalf("re-file: %d\n%s", code, out.String())
	}
	s := read(t, root, "bugs/deferred-batch-import.md")
	if !strings.Contains(s, "type: Bug") || !strings.Contains(s, "# Symptom") || strings.Contains(s, "# Gap") {
		t.Fatalf("a re-filed debt is typed and headed as a bug:\n%s", s)
	}
	// Its repairs are reported where the document now is.
	if !strings.Contains(out.String(), "  bugs/deferred-batch-import.md: ") || strings.Contains(out.String(), "  debts/deferred-batch-import.md: ") {
		t.Fatalf("the repairs are reported under the new path:\n%s", out.String())
	}
	if idx := read(t, root, "bugs/INDEX.md"); !strings.HasPrefix(idx, "# Bugs\n") || !strings.Contains(idx, "deferred-batch-import") {
		t.Fatalf("the listing moves to a new bugs/INDEX.md, headed as a register:\n%s", idx)
	}
	var v bytes.Buffer
	bundle.Validate(root, bundle.Options{Out: &v})
	if !strings.Contains(v.String(), "bugs/deferred-batch-import.md: a bug needs a non-empty `# Expected`") {
		t.Fatalf("validation asks for the analysis a debt never had to state:\n%s", v.String())
	}
}

func TestMoveRenumbersATask(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "features/venues/opening-hours/02-docs.md", "---\ntype: Task\nstatus: done\ntitle: Docs\ndepends-on: 01-build\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Objective\n\nDocs.\n")
	plan := read(t, root, "features/venues/opening-hours.plan.md")
	write(t, root, "features/venues/opening-hours.plan.md", plan+"2. [02-docs.md](opening-hours/02-docs.md)\n")
	move(t, root, "features/venues/opening-hours/01-build", "features/venues/opening-hours/01-build-hours")
	if s := read(t, root, "features/venues/opening-hours.plan.md"); !strings.Contains(s, "(opening-hours/01-build-hours.md)") {
		t.Fatalf("the plan links the renamed task:\n%s", s)
	}
	if s := read(t, root, "features/venues/opening-hours/02-docs.md"); !strings.Contains(s, "depends-on: 01-build-hours") {
		t.Fatalf("a sibling's depends-on follows the rename:\n%s", s)
	}
	validates(t, root)
}

func TestMoveDryRunChangesNothing(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	before := read(t, root, "changes/closed-hours-fix.md")
	var out bytes.Buffer
	if code := Move(root, "", "features/venues/opening-hours", "features/venues/trading-hours", true, &out); code != 0 {
		t.Fatalf("dry run: %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "would move features/venues/opening-hours.md -> features/venues/trading-hours.md") || !strings.Contains(out.String(), "Nothing was changed") {
		t.Fatalf("dry run reports the plan:\n%s", out.String())
	}
	read(t, root, "features/venues/opening-hours.md")
	if read(t, root, "changes/closed-hours-fix.md") != before {
		t.Fatal("a dry run edits nothing")
	}
}

func TestLogsKeepTheirWordsButNotBrokenLinks(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "bugs/hours-off-by-one.log.md", read(t, root, "bugs/hours-off-by-one.log.md")+
		"\nFirst seen in features/venues/opening-hours; see [it](/features/venues/opening-hours.md).\n")
	move(t, root, "features/venues/opening-hours", "features/venues/trading-hours")
	s := read(t, root, "bugs/hours-off-by-one.log.md")
	if !strings.Contains(s, "First seen in features/venues/opening-hours;") {
		t.Fatalf("a log keeps what things were called then:\n%s", s)
	}
	if !strings.Contains(s, "(/features/venues/trading-hours.md)") {
		t.Fatalf("but its links still resolve:\n%s", s)
	}
}

// fdf mv works on spec 1.0 bundles: a 0.x bundle is upgraded with
// `fdf migrate` first, and nothing in it moves.
func TestMovePointsA0xBundleAtMigrate(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	var out bytes.Buffer
	if code := Move(root, "", "venues/opening-hours", "venues/trading-hours", false, &out); code != 1 ||
		out.String() != "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n" {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	read(t, root, "venues/opening-hours.md")
}

// No name is reserved inside a register: a feature group called bugs/ moves,
// and takes features, like any other.
func TestMoveTreatsAGroupCalledBugsAsAnyOther(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	move(t, root, "features/venues/opening-hours", "features/bugs/opening-hours")
	read(t, root, "features/bugs/opening-hours.md")
	if s := read(t, root, "features/INDEX.md"); !strings.Contains(s, "* [Bugs](/features/bugs/INDEX.md) - features in bugs.\n") {
		t.Fatalf("features/INDEX.md lists the new group:\n%s", s)
	}
	validates(t, root)
	move(t, root, "features/bugs", "features/triage")
	read(t, root, "features/triage/opening-hours.md")
	validates(t, root)
}

// A document moves between flat and grouped, and into groups nested to any
// depth: its listing goes to the index of its new directory, each new group
// on the way gets an index listed in its parent's, and a group the move
// leaves empty is named.
func TestMoveBetweenFlatAndNestedGroups(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	// macOS Finder leaves one in any directory it has shown. It is not the
	// group's, so the group it is left in is emptied all the same.
	write(t, root, "features/venues/.DS_Store", "finder")
	out := move(t, root, "features/venues/opening-hours", "features/opening-hours")
	if s := read(t, root, "features/INDEX.md"); !strings.Contains(s, "* [opening-hours](opening-hours.md) - a capability.\n") {
		t.Fatalf("a flat feature is listed in features/INDEX.md:\n%s", s)
	}
	if !strings.Contains(out, "note: features/venues/ now holds no documents") {
		t.Fatalf("the emptied group is named:\n%s", out)
	}
	validates(t, root)

	out = move(t, root, "features/opening-hours", "features/platform/sites/opening-hours")
	for rel, want := range map[string]string{
		"features/INDEX.md":                "* [Platform](/features/platform/INDEX.md) - features in platform.\n",
		"features/platform/INDEX.md":       "# Platform\n\n* [Sites](/features/platform/sites/INDEX.md) - features in sites.\n",
		"features/platform/sites/INDEX.md": "# Sites\n\n* [opening-hours](opening-hours.md) - a capability.\n",
	} {
		if s := read(t, root, rel); !strings.Contains(s, want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, s)
		}
	}
	for _, want := range []string{"group features/platform/ listed in features/INDEX.md", "group features/platform/sites/ listed in features/platform/INDEX.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("the report should say %q:\n%s", want, out)
		}
	}
	if s := read(t, root, "changes/old-fix.md"); !strings.Contains(s, "affects: features/platform/sites/opening-hours") {
		t.Fatalf("every edge follows the feature to any depth:\n%s", s)
	}
	validates(t, root)

	// A group moves to another parent, its listing with it.
	move(t, root, "features/platform/sites", "features/sites")
	if s := read(t, root, "features/INDEX.md"); !strings.Contains(s, "* [Sites](/features/sites/INDEX.md) - features in sites.\n") {
		t.Fatalf("the group's listing moves to its new parent's index:\n%s", s)
	}
	if s := read(t, root, "features/platform/INDEX.md"); strings.Contains(s, "sites") {
		t.Fatalf("and leaves its old parent's:\n%s", s)
	}
	validates(t, root)

	// Register documents nest too.
	move(t, root, "bugs/ui/unclear-error", "bugs/platform/ui/unclear-error")
	if s := read(t, root, "bugs/platform/ui/INDEX.md"); !strings.Contains(s, "(/bugs/platform/ui/unclear-error.md)") && !strings.Contains(s, "(unclear-error.md)") {
		t.Fatalf("the bug is listed in its new group:\n%s", s)
	}
	validates(t, root)
}

// A practice, debt or bug owns no directory, so one beside it stays where
// it is when the entry moves, and the links into it are repaired. One that
// holds Markdown is an F3 error, and moving the entry away is the repair
// the validator asks for.
func TestMoveLeavesADirectoryBesideAnEntry(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	bug := read(t, root, "bugs/hours-off-by-one.md")
	write(t, root, "bugs/hours-off-by-one.md", strings.Replace(bug, "# Symptom\n", "# Symptom\n\n![Screenshot](hours-off-by-one/screenshot.png)\n", 1))
	write(t, root, "bugs/hours-off-by-one/screenshot.png", "PNG")
	out := move(t, root, "bugs/hours-off-by-one", "bugs/venues/hours-off-by-one")
	if !strings.Contains(out, "  bugs/hours-off-by-one/ stays where it is: a bug owns no directory\n") {
		t.Errorf("the move says the directory stays:\n%s", out)
	}
	if read(t, root, "bugs/hours-off-by-one/screenshot.png") != "PNG" ||
		!strings.Contains(read(t, root, "bugs/venues/hours-off-by-one.md"), "![Screenshot](../hours-off-by-one/screenshot.png)") {
		t.Errorf("the screenshot stays, and the link to it is repaired:\n%s", read(t, root, "bugs/venues/hours-off-by-one.md"))
	}
	validates(t, root)

	practice := "---\ntype: Practice\nstatus: active\ntitle: Permission checks\ndescription: How a permission is checked.\ntimestamp: 2026-09-25T00:00:00Z\n---\n\n# Rules\n\n- Check it in one place.\n\n# How\n\nThrough one helper.\n"
	write(t, root, "practices/INDEX.md", "# Practices\n\n* [Permission checks](/practices/permission-checks.md) - practice.\n")
	write(t, root, "practices/permission-checks.md", practice)
	write(t, root, "practices/permission-checks/examples.md", strings.Replace(practice, "Permission checks", "Examples", 1))
	var before bytes.Buffer
	if bundle.Validate(root, bundle.Options{Out: &before}) == 0 || !strings.Contains(before.String(), "rename one of them (F3)") {
		t.Fatalf("a directory of Markdown beside a practice is an F3 error:\n%s", before.String())
	}
	move(t, root, "practices/permission-checks", "practices/permission-rules")
	validates(t, root)
}

func TestMoveRefusals(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	cases := []struct{ from, to, want string }{
		{"features/venues/opening-hours", "features/venues/opening-hours", "already where it is"},
		{"features/venues/ghost", "features/venues/x", "is not a document, group or task"},
		{"changes", "work", "changes/ is a register"},
		{"features/venues/opening-hours", "venues/opening-hours", "a feature stays under features/"},
		// An ID written the 0.7 way gets its full ID suggested.
		{"venues/opening-hours", "features/venues/hours", "venues/opening-hours is not a document, group or task in this bundle — did you mean features/venues/opening-hours?"},
		{"venues", "features/places", "venues is not a document, group or task in this bundle — did you mean features/venues?"},
		{"features/venues/opening-hours", "venues/hours", "a feature stays under features/ (a debt and a bug can be re-filed as each other; nothing else changes register) — did you mean features/venues/hours?"},
		{"features/venues", "places", "a features/ group moves to another group in features/, not places — did you mean features/places?"},
		{"changes/old-fix", "practices/old-fix", "stays under changes/"},
		{"features/venues/opening-hours/01-build", "features/venues/other/01-build", "within its own task directory"},
		{"features/venues/opening-hours", "bugs/hours", "a feature stays under features/"},
		{"features/venues/opening-hours", "features/Venues/Hours", "lowercase"},
		{"features/venues", "features/venues/inner", "cannot move into itself"},
		{"features/venues", "bugs/venues", "a features/ group moves to another group in features/"},
		{"releases/1.2.0", "releases/1.3.0", "nothing in releases/ moves"},
		// A directory beside a document belongs to it (F3).
		{"changes/old-fix", "changes/closed-hours-fix/old-fix", "task directory of changes/closed-hours-fix"},
		{"changes/old-fix", "changes/hours", "changes/hours/ is a group"},
		{"features/venues/opening-hours", "features/venues/opening-hours/nested", "task directory of features/venues/opening-hours"},
		{"features/sites", "features/venues/opening-hours/sites", "features/venues/opening-hours/ is the task directory of features/venues/opening-hours"},
		// A directory that holds no Markdown is outside the bundle.
		{"features/images", "features/assets/images", "features/images/ holds no Markdown, so it is outside the bundle"},
	}
	write(t, root, "changes/hours/INDEX.md", "# Hours\n")
	write(t, root, "features/sites/INDEX.md", "# Sites\n")
	write(t, root, "features/images/logo.png", "PNG")
	write(t, root, "releases/1.2.0.md", "---\ntype: Release\ntitle: 1.2.0\nstatus: planned\n---\n\n# Features\n\n* (none)\n")
	for _, c := range cases {
		var out bytes.Buffer
		if code := Move(root, "", c.from, c.to, false, &out); code != 1 || !strings.Contains(out.String(), c.want) {
			t.Errorf("fdf mv %s %s: exit %d, want a refusal containing %q:\n%s", c.from, c.to, code, c.want, out.String())
		}
	}
	// A stray file where a sibling would land is never overwritten either.
	write(t, root, "features/venues/fresh.spec.md", "---\ntype: Spec\n---\n")
	var stray bytes.Buffer
	if code := Move(root, "", "features/venues/opening-hours", "features/venues/fresh", false, &stray); code != 1 || !strings.Contains(stray.String(), "would overwrite features/venues/fresh.spec.md") {
		t.Fatalf("a stray target sibling is refused:\n%s", stray.String())
	}
	// A target that exists is never overwritten.
	write(t, root, "features/venues/other.md", "---\ntype: Feature\n---\n")
	var out bytes.Buffer
	if code := Move(root, "", "features/venues/opening-hours", "features/venues/other", false, &out); code != 1 || !strings.Contains(out.String(), "already exists") {
		t.Fatalf("an existing target is refused:\n%s", out.String())
	}
}

// A group directory spelled in capitals that holds no Markdown is outside
// the bundle, but on a disk that ignores case it is where a move into the
// group spelled in lowercase would land, and the bundle would fail F3. The
// move is refused, naming the directory as it is spelled, and nothing moves.
func TestMoveRefusesAGroupSpelledInAnotherCase(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "features/Places/map.png", "PNG")
	for _, tc := range []struct{ from, to string }{
		{"features/venues/opening-hours", "features/places/opening-hours"},
		{"features/venues", "features/places/venues"},
	} {
		var out bytes.Buffer
		want := "error: features/Places/ is already there: a disk that ignores case would file " + tc.to + " in it, and directory names are lowercase (F3); rename that directory, or choose another name\n"
		if code := Move(root, "", tc.from, tc.to, false, &out); code != 1 || out.String() != want {
			t.Errorf("fdf mv %s %s: exit %d\n got: %q\nwant: %q", tc.from, tc.to, code, out.String(), want)
		}
	}
	read(t, root, "features/venues/opening-hours.md")
}

// A move that changes a document's depth repairs the links that leave the
// bundle too: the file that holds them moved, so its path to them changed,
// even though they did not. A footnote is not a link, and a reference
// definition inside a code fence is a sample; neither changes.
func TestMoveDeeperRepairsLinksThatLeaveTheBundle(t *testing.T) {
	root := fixture(t, "valid-bugs-v10")
	write(t, root, "changes/old-fix.md", read(t, root, "changes/old-fix.md")+
		"\n- [auth](../../okf/modules/auth.md#login)\n"+
		"- [angle](<../../okf/modules/auth.md>)\n"+
		"- [code](../../src/refund.go)\n"+
		"- [defined][okf]\n\n"+
		"[okf]: ../../okf/modules/auth.md\n"+
		"[^1]: See the notes.\n\n"+
		"```markdown\n[sample]: ../features/venues/opening-hours.md\n```\n")
	move(t, root, "changes/old-fix", "changes/hours/old-fix")
	got := read(t, root, "changes/hours/old-fix.md")
	for _, want := range []string{
		"(../../../okf/modules/auth.md#login)",
		"(<../../../okf/modules/auth.md>)",
		"(../../../src/refund.go)",
		"[okf]: ../../../okf/modules/auth.md",
		"[^1]: See the notes.",
		"[sample]: ../features/venues/opening-hours.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("after moving one level deeper, want %q in:\n%s", want, got)
		}
	}
}
