package tags

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// newBundle is a fresh bundle, as fdf init makes it: pinned to the current
// version, with its empty TAGS.md.
func newBundle(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "fdf")
	var out bytes.Buffer
	if code := scaffold.Init(root, &out); code != 0 {
		t.Fatalf("init: exit %d\n%s", code, out.String())
	}
	return root
}

func put(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// doc is a debt carrying the frontmatter lines tags.
func doc(tags string) string {
	return "---\ntype: Debt\nstatus: open\ntitle: X\ndescription: d.\n" + tags + "timestamp: 2026-10-01T00:00:00Z\n---\n\n# Gap\n\nA gap.\n"
}

// A merge keeps the form `tags` was written in, lists the kept tag once, and
// changes no other line, the timestamp included: a tag fix is maintenance.
func TestRetagKeepsTheFormItWasWrittenIn(t *testing.T) {
	merge := func(ts []string) []string { return replaced(ts, "testing", "test") }
	for _, tc := range []struct{ name, in, want string }{
		{"inline", "tags: [api, testing]\n", "tags: [api, test]\n"},
		{"inline, quoted", "tags: [\"testing\", 'api']\n", "tags: [test, api]\n"},
		{"inline, both", "tags: [test, testing]\n", "tags: [test]\n"},
		{"one value", "tags: testing\n", "tags: [test]\n"},
		{"block", "tags:\n  - api\n  - testing\n", "tags:\n  - api\n  - test\n"},
		{"block, both", "tags:\n- testing\n- test\n", "tags:\n- test\n"},
		{"block, with comments", "tags:\n  # layers\n  - testing\n  # areas\n  - api\n", "tags:\n  # layers\n  - test\n  # areas\n  - api\n"},
		{"block, comments and both", "tags:\n  # one\n  - testing\n  - test\n", "tags:\n  # one\n  - test\n"},
	} {
		got, ok := retag(doc(tc.in), merge)
		if !ok || got != doc(tc.want) {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, doc(tc.want))
		}
	}
	// A document with no tags is given them before its timestamp.
	got, ok := retag(doc(""), func([]string) []string { return []string{"gtm", "api"} })
	if !ok || got != doc("tags: [gtm, api]\n") {
		t.Errorf("stamp: got\n%s", got)
	}
	if _, ok := retag("# No frontmatter\n", merge); ok {
		t.Errorf("a text with no frontmatter cannot be retagged")
	}
}

// fdf init writes an empty dictionary that validates, lists it in the root
// INDEX.md, and holds a comment whose sample heading is no entry.
func TestInitWritesAnEmptyDictionary(t *testing.T) {
	root := newBundle(t)
	text := get(t, root, "TAGS.md")
	d, exists := bundle.LoadTags(root)
	if !exists || d == nil || len(d.Tags) != 0 {
		t.Fatalf("TAGS.md defines no tag yet:\n%s", text)
	}
	if !strings.Contains(get(t, root, "INDEX.md"), scaffold.TagsListing) {
		t.Errorf("the root INDEX.md lists TAGS.md:\n%s", get(t, root, "INDEX.md"))
	}
	var out bytes.Buffer
	if code := bundle.Validate(root, bundle.Options{Out: &out}); code != 0 || strings.Contains(out.String(), "TAGS.md") {
		t.Errorf("a new bundle's TAGS.md validates: exit %d\n%s", code, out.String())
	}
}

// Adding a tag writes a TODO definition, which F15 reports until a person
// writes it; --definition writes it at once, or fills in a TODO, and a
// redefinition is logged.
func TestAddAndDefine(t *testing.T) {
	root := newBundle(t)
	var out bytes.Buffer
	if code := Add(root, []string{"gtm", "api"}, "", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	d, _ := bundle.LoadTags(root)
	if g := d.Lookup("gtm"); g == nil || g.Defined() || d.Lookup("api") == nil {
		t.Fatalf("gtm and api are added with a TODO:\n%s", get(t, root, "TAGS.md"))
	}
	out.Reset()
	if code := Add(root, []string{"gtm"}, "Work the public launch waits for.", &out); code != 0 || !strings.Contains(out.String(), "defined gtm in TAGS.md") {
		t.Fatalf("filling in a TODO: exit %d\n%s", code, out.String())
	}
	if strings.Contains(get(t, root, "LOG.md"), "redefined") {
		t.Errorf("filling in a TODO is not a redefinition")
	}
	out.Reset()
	if code := Add(root, []string{"gtm"}, "Work the launch waits for, and nothing after it.", &out); code != 0 || !strings.Contains(get(t, root, "LOG.md"), "* **Tags**: redefined `gtm`: Work the launch waits for, and nothing after it.") {
		t.Errorf("a redefinition is logged: exit %d\n%s\n%s", code, out.String(), get(t, root, "LOG.md"))
	}
	d, _ = bundle.LoadTags(root)
	if g := d.Lookup("gtm"); g.Definition != "Work the launch waits for, and nothing after it." {
		t.Errorf("gtm's definition: %q", g.Definition)
	}
	for _, bad := range []string{"Go To Market", "gtm_"} {
		out.Reset()
		if code := Add(root, []string{bad}, "", &out); code != 1 || !strings.Contains(out.String(), "is not a tag") {
			t.Errorf("%q is refused: exit %d\n%s", bad, code, out.String())
		}
	}
	out.Reset()
	if code := Add(root, []string{"security", "securities"}, "", &out); code != 0 || !strings.Contains(out.String(), "note: securities looks like security") {
		t.Errorf("a tag that looks like another is noted as it is added: exit %d\n%s", code, out.String())
	}
}

// A bundle pinned before 1.2 has no dictionary to write.
func TestWritingNeedsSpec12(t *testing.T) {
	root := newBundle(t)
	put(t, root, "INDEX.md", strings.Replace(get(t, root, "INDEX.md"), `fdf_version: "`+scaffold.CurrentVersion()+`"`, `fdf_version: "1.1"`, 1))
	var out bytes.Buffer
	if code := Add(root, []string{"gtm"}, "", &out); code != 1 || !strings.Contains(out.String(), "TAGS.md is a file of the bundle from spec 1.2, and this bundle pins 1.1") {
		t.Errorf("exit %d\n%s", code, out.String())
	}
}

// --sync starts a dictionary from the tags a bundle's documents carry, each
// with a TODO; a tag not written as one, or a listed spelling, is named with
// its repair, and two that look like one are named.
func TestSyncStartsADictionary(t *testing.T) {
	root := newBundle(t)
	if err := os.Remove(filepath.Join(root, "TAGS.md")); err != nil {
		t.Fatal(err)
	}
	put(t, root, "INDEX.md", strings.Replace(get(t, root, "INDEX.md"), scaffold.TagsListing+"\n", "", 1))
	put(t, root, "debts/a.md", doc("tags: [payments, Auth, session]\n"))
	put(t, root, "debts/b.md", doc("tags:\n  - sessions\n  - payments\n"))
	var out bytes.Buffer
	if code := Sync(root, true, &out); code != 0 || !strings.Contains(out.String(), "would add 3 tag(s) to TAGS.md, each with a TODO definition: payments, session, sessions") ||
		!strings.Contains(out.String(), "left out Auth — 1 document(s); not a tag name: `fdf tag --merge 'Auth' auth`") ||
		!strings.Contains(out.String(), "session and sessions look like one tag") {
		t.Fatalf("dry run: exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "TAGS.md")); err == nil {
		t.Fatalf("a dry run writes nothing")
	}
	out.Reset()
	if code := Sync(root, false, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	d, exists := bundle.LoadTags(root)
	if !exists || d.Lookup("payments") == nil || d.Lookup("sessions") == nil || d.Lookup("Auth") != nil {
		t.Fatalf("TAGS.md:\n%s", get(t, root, "TAGS.md"))
	}
	if !strings.Contains(get(t, root, "INDEX.md"), scaffold.TagsListing) {
		t.Errorf("starting the dictionary lists it in the root INDEX.md")
	}
	if log := get(t, root, "LOG.md"); !strings.Contains(log, "* **Tags**: started the tag dictionary, TAGS.md") || !strings.Contains(log, "added 3 tag(s) the documents carry to TAGS.md") {
		t.Errorf("LOG.md:\n%s", log)
	}
	out.Reset()
	if code := Sync(root, false, &out); code != 0 || !strings.Contains(out.String(), "TAGS.md defines every tag written as one that the documents carry.") {
		t.Errorf("a second sync adds nothing: exit %d\n%s", code, out.String())
	}
}

// --merge folds one tag into another: documents retagged without their
// timestamps moving, the dropped tag's entry gone and listed under the kept
// one, its definition kept when the kept tag had none, and the fix logged.
func TestMergeIsATagFix(t *testing.T) {
	root := newBundle(t)
	put(t, root, "debts/a.md", doc("tags: [testing, api]\n"))
	put(t, root, "debts/b.md", doc("tags: [tests]\n"))
	var out bytes.Buffer
	if code := Add(root, []string{"testing"}, "Proving the software does what its features say.", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if code := Add(root, []string{"api"}, "The interface other programs call.", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	put(t, root, "TAGS.md", strings.Replace(get(t, root, "TAGS.md"), "say.\n", "say.\n- instead-of: tests\n", 1))
	out.Reset()
	if code := Merge(root, "testing", "test", false, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if got := get(t, root, "debts/a.md"); got != doc("tags: [test, api]\n") {
		t.Errorf("debts/a.md:\n%s", got)
	}
	d, _ := bundle.LoadTags(root)
	if d.Lookup("testing") != nil {
		t.Errorf("testing's entry is gone:\n%s", get(t, root, "TAGS.md"))
	}
	if tt := d.Lookup("test"); tt == nil || tt.Definition != "Proving the software does what its features say." || strings.Join(tt.InsteadOf, ",") != "testing,tests" {
		t.Errorf("test takes testing's definition and its spellings:\n%s", get(t, root, "TAGS.md"))
	}
	if !strings.Contains(get(t, root, "LOG.md"), "* **Tag fix**: `testing` → `test` in 1 document(s)") {
		t.Errorf("LOG.md:\n%s", get(t, root, "LOG.md"))
	}
	// tests, a listed spelling, merges into the tag that lists it, and no other.
	out.Reset()
	if code := Merge(root, "tests", "api", false, &out); code != 1 || !strings.Contains(out.String(), "tests is listed under `instead-of` of test in TAGS.md — merge it into test") {
		t.Errorf("exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := Merge(root, "tests", "test", false, &out); code != 0 || get(t, root, "debts/b.md") != doc("tags: [test]\n") {
		t.Errorf("exit %d\n%s\n%s", code, out.String(), get(t, root, "debts/b.md"))
	}
	var v bytes.Buffer
	if code := bundle.Validate(root, bundle.Options{Out: &v}); strings.Contains(v.String(), "(F15)") {
		t.Errorf("after the merges F15 has nothing to say: exit %d\n%s", code, v.String())
	}
	out.Reset()
	if code := Merge(root, "ghost", "test", false, &out); code != 1 || !strings.Contains(out.String(), "no document carries ghost") {
		t.Errorf("a tag nobody carries or defines: exit %d\n%s", code, out.String())
	}
}

// Stamp gives a new document its tags, and adds those the dictionary lacks;
// Check refuses a spelling the dictionary lists under another tag first.
func TestStampAndCheck(t *testing.T) {
	root := newBundle(t)
	put(t, root, "debts/a.md", doc(""))
	var out bytes.Buffer
	if code := Stamp(root, "debts/a", []string{"gtm", "gtm", "api"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if got := get(t, root, "debts/a.md"); got != doc("tags: [gtm, api]\n") {
		t.Errorf("debts/a.md:\n%s", got)
	}
	if d, _ := bundle.LoadTags(root); d.Lookup("gtm") == nil || d.Lookup("api") == nil {
		t.Errorf("TAGS.md:\n%s", get(t, root, "TAGS.md"))
	}
	put(t, root, "TAGS.md", strings.Replace(get(t, root, "TAGS.md"), "## api\n", "## api\nThe API.\n- instead-of: apis\n", 1))
	out.Reset()
	if Check(root, []string{"apis"}, &out) || !strings.Contains(out.String(), "error: --tags: apis is listed under `instead-of` of api in TAGS.md — use api") {
		t.Errorf("a listed spelling is refused:\n%s", out.String())
	}
	out.Reset()
	if Check(root, []string{"Not A Tag"}, &out) {
		t.Errorf("a name that is no tag is refused")
	}
}

// A merge records the dropped spelling under the kept tag only where it can:
// with no TAGS.md there is nothing to record it in, and a spelling not
// written as a tag cannot be listed. The log says only what happened.
func TestMergeLogsOnlyWhatItRecorded(t *testing.T) {
	root := newBundle(t)
	if err := os.Remove(filepath.Join(root, "TAGS.md")); err != nil {
		t.Fatal(err)
	}
	put(t, root, "debts/a.md", doc("tags: [booking, Security]\n"))
	var out bytes.Buffer
	if code := Merge(root, "booking", "bookings", false, &out); code != 0 || !strings.Contains(out.String(), "note: the bundle has no TAGS.md, so nothing records booking as a spelling of bookings") {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if log := get(t, root, "LOG.md"); !strings.Contains(log, "* **Tag fix**: `booking` → `bookings` in 1 document(s) (fdf tag --merge).") || strings.Contains(log, "is listed under") {
		t.Errorf("LOG.md:\n%s", log)
	}
	out.Reset()
	if code := Sync(root, false, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := Merge(root, "Security", "security", false, &out); code != 0 || !strings.Contains(out.String(), "note: Security is not written as a tag is, so it is not listed under security's `instead-of`") {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if d, _ := bundle.LoadTags(root); d.Lookup("security") == nil || len(d.Lookup("security").InsteadOf) != 0 {
		t.Errorf("TAGS.md:\n%s", get(t, root, "TAGS.md"))
	}
	if log := get(t, root, "LOG.md"); strings.Contains(log, "`Security` is listed") {
		t.Errorf("LOG.md:\n%s", log)
	}
}

// Defining the last tag in TAGS.md keeps the file's final newline, and
// defining one in the middle keeps the blank line before the next.
func TestDefinitionKeepsTheLayout(t *testing.T) {
	root := newBundle(t)
	var out bytes.Buffer
	if code := Add(root, []string{"alpha", "beta"}, "", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	for _, n := range []string{"beta", "alpha"} {
		if code := Add(root, []string{n}, "Defined "+n+".", &out); code != 0 {
			t.Fatalf("exit %d\n%s", code, out.String())
		}
	}
	text := get(t, root, "TAGS.md")
	if !strings.HasSuffix(text, "## beta\nDefined beta.\n") || !strings.Contains(text, "## alpha\nDefined alpha.\n\n## beta\n") {
		t.Errorf("TAGS.md:\n%q", text)
	}
}

// Adding a tag never starts the dictionary of a bundle whose documents carry
// tags already: that would hold all of them to it at once. --sync does.
func TestAddStartsNoDictionaryOverExistingTags(t *testing.T) {
	root := newBundle(t)
	if err := os.Remove(filepath.Join(root, "TAGS.md")); err != nil {
		t.Fatal(err)
	}
	put(t, root, "debts/a.md", doc("tags: [payments]\n"))
	var out bytes.Buffer
	if code := Add(root, []string{"gtm"}, "Launch work.", &out); code != 1 || !strings.Contains(out.String(), "`fdf tag --sync` starts it from them all") {
		t.Errorf("exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "TAGS.md")); err == nil {
		t.Errorf("no TAGS.md is written")
	}
}

// TAGS.md keeps the line ends it was written with, a merge leaves its
// timestamp as it is (a tag fix is maintenance), two instead-of bullets fold
// into one, and a comment inside an entry survives a new definition.
func TestEditingKeepsWhatItDoesNotChange(t *testing.T) {
	root := newBundle(t)
	crlf := strings.ReplaceAll("---\ntype: Tags\ntitle: Tags\ndescription: d.\ntimestamp: 2026-01-01T00:00:00Z\n---\n\n# Tags\n\n## web\nThe browser app.\n<!-- keep me -->\n- instead-of: www\n- instead-of: site\n\n## api\nTODO — x.\n", "\n", "\r\n")
	put(t, root, "TAGS.md", crlf)
	put(t, root, "debts/a.md", doc("tags: [frontend, api]\n"))
	var out bytes.Buffer
	if code := Merge(root, "frontend", "web", false, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	text := get(t, root, "TAGS.md")
	if strings.Count(text, "\r\n") != strings.Count(text, "\n") {
		t.Errorf("TAGS.md keeps CRLF line ends:\n%q", text)
	}
	if !strings.Contains(text, "timestamp: 2026-01-01T00:00:00Z") {
		t.Errorf("a tag fix leaves the timestamp as it is:\n%s", text)
	}
	if !strings.Contains(text, "- instead-of: www, site, frontend\r\n") || strings.Contains(text, "- instead-of: site\r\n") {
		t.Errorf("one instead-of line:\n%q", text)
	}
	out.Reset()
	if code := Add(root, []string{"web"}, "What runs in the browser.", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if text := get(t, root, "TAGS.md"); !strings.Contains(text, "## web\r\nWhat runs in the browser.\r\n<!-- keep me -->\r\n- instead-of:") {
		t.Errorf("the comment survives a definition:\n%q", text)
	}
	out.Reset()
	if code := Merge(root, "api", "www", false, &out); code != 1 || !strings.Contains(out.String(), "merge into web") {
		t.Errorf("a listed spelling of another tag is no target: exit %d\n%s", code, out.String())
	}
}

// A tags.md that a disk ignoring case reads as TAGS.md is not the
// dictionary: fdf writes nothing into it and says to rename it.
func TestACaseTwinIsNoDictionary(t *testing.T) {
	root := newBundle(t)
	if err := os.Rename(filepath.Join(root, "TAGS.md"), filepath.Join(root, "tags.md")); err != nil {
		t.Fatal(err)
	}
	if _, exists := bundle.LoadTags(root); exists {
		t.Errorf("tags.md is not TAGS.md")
	}
	var out bytes.Buffer
	if code := Add(root, []string{"gtm"}, "Launch work.", &out); code != 1 || !strings.Contains(out.String(), "tags.md is at the bundle root") {
		t.Errorf("exit %d\n%s", code, out.String())
	}
}
