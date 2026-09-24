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

func TestMoveRenamesAFeatureWithItsTrailAndRepairsEveryReference(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	// A release listing the feature, and a sample link in a code fence that
	// must stay a sample.
	write(t, root, "venues/opening-hours.spec.md", read(t, root, "venues/opening-hours.spec.md")+
		"\nSee [the plan](opening-hours.plan.md).\n\n```markdown\n[x](/venues/opening-hours.md)\n```\n")
	out := move(t, root, "venues/opening-hours", "venues/trading-hours")

	for _, rel := range []string{"venues/opening-hours.md", "venues/opening-hours.spec.md", "venues/opening-hours/01-build.md"} {
		gone(t, root, rel)
	}
	for _, rel := range []string{"venues/trading-hours.md", "venues/trading-hours.spec.md", "venues/trading-hours.plan.md", "venues/trading-hours.test.md", "venues/trading-hours/01-build.md"} {
		read(t, root, rel)
	}
	if s := read(t, root, "venues/trading-hours.plan.md"); !strings.Contains(s, "(trading-hours/01-build.md)") {
		t.Fatalf("the plan's task link must follow its task directory:\n%s", s)
	}
	if s := read(t, root, "venues/trading-hours.spec.md"); !strings.Contains(s, "(trading-hours.plan.md)") || !strings.Contains(s, "[x](/venues/opening-hours.md)") {
		t.Fatalf("a real link is repaired, a sample in a code fence is not:\n%s", s)
	}
	fix := read(t, root, "changes/closed-hours-fix.md")
	if !strings.Contains(fix, "affects: venues/trading-hours") || !strings.Contains(fix, "## venues/trading-hours") {
		t.Fatalf("a Fix's affects and declaration heading follow the move:\n%s", fix)
	}
	bug := read(t, root, "bugs/hours-off-by-one.md")
	if !strings.Contains(bug, "affects: venues/trading-hours") || !strings.Contains(bug, "## venues/trading-hours") {
		t.Fatalf("a bug's affects and # Violates heading follow the move:\n%s", bug)
	}
	if s := read(t, root, "venues/INDEX.md"); !strings.Contains(s, "(trading-hours.md)") {
		t.Fatalf("the group index link follows the move:\n%s", s)
	}
	if s := read(t, root, "LOG.md"); !strings.Contains(s, "**Moved**: `venues/opening-hours` → `venues/trading-hours`") {
		t.Fatalf("the move is logged:\n%s", s)
	}
	if !strings.Contains(out, "reference(s) repaired") {
		t.Fatalf("the command reports what it repaired:\n%s", out)
	}
	validates(t, root)
}

func TestMoveToANewGroupMovesTheListing(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	move(t, root, "venues/opening-hours", "sites/opening-hours")
	if s := read(t, root, "venues/INDEX.md"); strings.Contains(s, "opening-hours") {
		t.Fatalf("the old group index no longer lists the feature:\n%s", s)
	}
	if s := read(t, root, "sites/INDEX.md"); !strings.Contains(s, "(opening-hours.md)") {
		t.Fatalf("the new group index lists it:\n%s", s)
	}
	validates(t, root)
}

func TestMoveRenamesAWholeGroup(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	move(t, root, "venues", "sites")
	gone(t, root, "venues")
	if s := read(t, root, "INDEX.md"); !strings.Contains(s, "(/sites/INDEX.md)") {
		t.Fatalf("the root index follows the group:\n%s", s)
	}
	if s := read(t, root, "changes/old-fix.md"); !strings.Contains(s, "affects: sites/opening-hours") {
		t.Fatalf("every edge into the group follows it:\n%s", s)
	}
	validates(t, root)
}

func TestMoveRefilesADebtAsABug(t *testing.T) {
	root := fixture(t, "valid-debt-v06")
	// valid-debt-v06 pins 0.6; re-filing is a 0.7 move, so pin the copy.
	write(t, root, "INDEX.md", strings.Replace(read(t, root, "INDEX.md"), `"0.6"`, `"0.7"`, 1))
	write(t, root, "debts/INDEX.md", read(t, root, "debts/INDEX.md")+"* [Deferred batch import](deferred-batch-import.md) - deferred.\n")
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
	root := fixture(t, "valid-bugs-v07")
	write(t, root, "venues/opening-hours/02-docs.md", "---\ntype: Task\nstatus: done\ntitle: Docs\ndepends-on: 01-build\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Objective\n\nDocs.\n")
	plan := read(t, root, "venues/opening-hours.plan.md")
	write(t, root, "venues/opening-hours.plan.md", plan+"2. [02-docs.md](opening-hours/02-docs.md)\n")
	move(t, root, "venues/opening-hours/01-build", "venues/opening-hours/01-build-hours")
	if s := read(t, root, "venues/opening-hours.plan.md"); !strings.Contains(s, "(opening-hours/01-build-hours.md)") {
		t.Fatalf("the plan links the renamed task:\n%s", s)
	}
	if s := read(t, root, "venues/opening-hours/02-docs.md"); !strings.Contains(s, "depends-on: 01-build-hours") {
		t.Fatalf("a sibling's depends-on follows the rename:\n%s", s)
	}
	validates(t, root)
}

func TestMoveDryRunChangesNothing(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	before := read(t, root, "changes/closed-hours-fix.md")
	var out bytes.Buffer
	if code := Move(root, "", "venues/opening-hours", "venues/trading-hours", true, &out); code != 0 {
		t.Fatalf("dry run: %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "would move venues/opening-hours.md -> venues/trading-hours.md") || !strings.Contains(out.String(), "Nothing was changed") {
		t.Fatalf("dry run reports the plan:\n%s", out.String())
	}
	read(t, root, "venues/opening-hours.md")
	if read(t, root, "changes/closed-hours-fix.md") != before {
		t.Fatal("a dry run edits nothing")
	}
}

func TestLogsKeepTheirWordsButNotBrokenLinks(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	write(t, root, "bugs/hours-off-by-one.log.md", read(t, root, "bugs/hours-off-by-one.log.md")+
		"\nFirst seen in venues/opening-hours; see [it](/venues/opening-hours.md).\n")
	move(t, root, "venues/opening-hours", "venues/trading-hours")
	s := read(t, root, "bugs/hours-off-by-one.log.md")
	if !strings.Contains(s, "First seen in venues/opening-hours;") {
		t.Fatalf("a log keeps what things were called then:\n%s", s)
	}
	if !strings.Contains(s, "(/venues/trading-hours.md)") {
		t.Fatalf("but its links still resolve:\n%s", s)
	}
}

// bugsGroupV06 is a v0.6 bundle whose feature group is named bugs/: legal
// under v0.6, and what `fdf migrate` asks to be moved before it reaches v0.7.
func bugsGroupV06(t *testing.T) string {
	t.Helper()
	root := fixture(t, "valid-domain-v06")
	write(t, root, "INDEX.md", read(t, root, "INDEX.md")+"* [Bugs](/bugs/INDEX.md) - the crash tracker.\n")
	write(t, root, "bugs/INDEX.md", "# Bugs features\n\n* [Crash report](/bugs/crash-report.md) - crash reports.\n")
	write(t, root, "bugs/crash-report.md", strings.NewReplacer("title: Example", "title: Crash report", "Feature: Example", "Feature: Crash report").
		Replace(read(t, root, "wdise/example.md")))
	validates(t, root)
	return root
}

// Which directories are registers follows the pin: on a v0.6 bundle bugs/ is
// a feature group, so its documents and the group itself move like any other.
func TestMoveOnAV06BundleTreatsBugsAsAFeatureGroup(t *testing.T) {
	root := bugsGroupV06(t)
	move(t, root, "bugs/crash-report", "issues/crash-report")
	gone(t, root, "bugs/crash-report.md")
	if s := read(t, root, "issues/INDEX.md"); !strings.HasPrefix(s, "# Issues features\n") || !strings.Contains(s, "crash-report.md") {
		t.Fatalf("the feature is listed in its new group:\n%s", s)
	}
	// Under v0.6 a feature may move into bugs/ as well.
	move(t, root, "wdise/example", "bugs/example")
	validates(t, root)

	root = bugsGroupV06(t)
	move(t, root, "bugs", "issues")
	gone(t, root, "bugs")
	read(t, root, "issues/crash-report.md")
	if s := read(t, root, "INDEX.md"); !strings.Contains(s, "(/issues/INDEX.md)") {
		t.Fatalf("the root index follows the group:\n%s", s)
	}
	validates(t, root)

	// Without a bug register there is nothing to re-file a debt into.
	root = bugsGroupV06(t)
	write(t, root, "debts/INDEX.md", "# Debt\n\n* [Gap](/debts/gap.md) - a gap.\n")
	write(t, root, "debts/gap.md", "---\ntype: Debt\nstatus: open\ntitle: Gap\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Gap\n\nMissing.\n")
	var out bytes.Buffer
	if code := Move(root, "", "debts/gap", "bugs/gap", false, &out); code != 1 || !strings.Contains(out.String(), "a debt stays under debts/ (nothing changes register)") {
		t.Fatalf("a v0.6 debt cannot be re-filed as a bug: exit %d\n%s", code, out.String())
	}
}

func TestMoveRefusals(t *testing.T) {
	root := fixture(t, "valid-bugs-v07")
	cases := []struct{ from, to, want string }{
		{"venues/opening-hours", "venues/opening-hours", "already where it is"},
		{"venues/ghost", "venues/x", "is not a feature"},
		{"changes", "work", "reserved directory"},
		{"changes/old-fix", "practices/old-fix", "stays under changes/"},
		{"venues/opening-hours/01-build", "venues/other/01-build", "within its own task directory"},
		{"venues/opening-hours", "bugs/hours", "moves to <group>/<slug>"},
		{"venues/opening-hours", "Venues/Hours", "lowercase"},
	}
	for _, c := range cases {
		var out bytes.Buffer
		if code := Move(root, "", c.from, c.to, false, &out); code != 1 || !strings.Contains(out.String(), c.want) {
			t.Errorf("fdf mv %s %s: exit %d, want a refusal containing %q:\n%s", c.from, c.to, code, c.want, out.String())
		}
	}
	// A stray file where a sibling would land is never overwritten either.
	write(t, root, "venues/fresh.spec.md", "---\ntype: Spec\n---\n")
	var stray bytes.Buffer
	if code := Move(root, "", "venues/opening-hours", "venues/fresh", false, &stray); code != 1 || !strings.Contains(stray.String(), "would overwrite venues/fresh.spec.md") {
		t.Fatalf("a stray target sibling is refused:\n%s", stray.String())
	}
	// A target that exists is never overwritten.
	write(t, root, "venues/other.md", "---\ntype: Feature\n---\n")
	var out bytes.Buffer
	if code := Move(root, "", "venues/opening-hours", "venues/other", false, &out); code != 1 || !strings.Contains(out.String(), "already exists") {
		t.Fatalf("an existing target is refused:\n%s", out.String())
	}
}
