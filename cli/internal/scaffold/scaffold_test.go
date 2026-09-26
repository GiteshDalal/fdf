package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
)

// fillContext simulates the fdf-init interview: replace each Context stub with
// real content so the stub sentinel is gone and F9 is satisfied.
func fillContext(t *testing.T, root string) {
	t.Helper()
	// Driven by contextDocs so a newly added Context document is filled here
	// too, instead of failing F9 in every test that scaffolds a feature.
	for _, c := range contextDocs {
		content := "Real content.\n"
		if c.file == "DOMAIN.md" {
			content = "# Terms\n\n## Venue\nA physical location where a merchant sells.\n- instead-of: shopfront\n"
		}
		body := "---\ntype: Context\ntitle: " + c.file + "\ndescription: filled.\n" +
			"timestamp: 2026-07-06T00:00:00Z\n---\n\n" + content
		if err := os.WriteFile(filepath.Join(root, c.file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInitWritesSurfacesStub(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "docs", "fdf")
	var buf bytes.Buffer
	if code := Init(root, &buf); code != 0 {
		t.Fatalf("init exit %d: %s", code, buf.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "SURFACES.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "type: Context") || !strings.Contains(s, "<!-- fdf:stub -->") {
		t.Fatalf("SURFACES.md stub incomplete:\n%s", s)
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+currentVersion+`"`) {
		t.Fatalf("pin: %s", idx)
	}
}

func TestInitScaffoldsConformingBundle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	if code := Init(root, &out); code != 0 {
		t.Fatalf("init: %d\n%s", code, out.String())
	}
	for _, f := range []string{"INDEX.md", "LOG.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md",
		"features/INDEX.md", "changes/INDEX.md", "practices/INDEX.md", "debts/INDEX.md", "bugs/INDEX.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(f))); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "releases")); err == nil {
		t.Error("releases/ is written with the first release, not by init")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	for _, want := range []string{`fdf_version: "` + currentVersion + `"`, "/SPEC.md", "* [Features](/features/INDEX.md) - what the software does.\n",
		"* [Bugs](/bugs/INDEX.md) - known defects not repaired yet.\n", "[Domain](/DOMAIN.md) - the project's context.\n"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("INDEX.md should hold %q:\n%s", want, raw)
		}
	}
	// Zero features: unfilled stubs are warnings only, so init is conformant.
	var vout bytes.Buffer
	if exit := bundle.Validate(root, bundle.Options{Out: &vout}); exit != 0 {
		t.Fatalf("scaffold not conformant:\n%s", vout.String())
	}
}

func TestInitIdempotentAndMigrateHint(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(root, &out)
	out.Reset()
	if code := Init(root, &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-init: code %d out %q", code, out.String())
	}
	// The pin is read as every command reads it, quoted either way or not
	// at all, and a bundle the commands cannot work on is refused as they
	// refuse it.
	idx := filepath.Join(root, "INDEX.md")
	raw, _ := os.ReadFile(idx)
	for _, tc := range []struct {
		pin, says string
		code      int
	}{
		{"'" + currentVersion + "'", "up to date (fdf_version " + currentVersion + ")", 0},
		{currentVersion, "up to date (fdf_version " + currentVersion + ")", 0},
		{`"0.1"`, "pins fdf_version 0.1; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first", 1},
		{`"1.3"`, "pins fdf_version 1.3, newer than any spec this fdf knows (1.0) — upgrade fdf", 1},
	} {
		os.WriteFile(idx, bytes.Replace(raw, []byte(`"`+currentVersion+`"`), []byte(tc.pin), 1), 0o644)
		out.Reset()
		if code := Init(root, &out); code != tc.code || !strings.Contains(out.String(), tc.says) {
			t.Errorf("pin %s: exit %d, want %d saying %q:\n%s", tc.pin, code, tc.code, tc.says, out.String())
		}
	}
}

func TestNewScaffoldsDraftFeature(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(root, &out)
	fillContext(t, root) // F9: a feature-bearing bundle needs filled Context docs
	if code := New(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("new: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "features", "payments", "instant-refunds.md"))
	for _, want := range []string{"type: Feature", "status: draft", "Feature: Instant refunds", "Scenario:"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("feature missing %q:\n%s", want, raw)
		}
	}
	gidx, _ := os.ReadFile(filepath.Join(root, "features", "payments", "INDEX.md"))
	if !strings.Contains(string(gidx), "* [Instant refunds](/features/payments/instant-refunds.md) - TODO.\n") {
		t.Fatalf("group index not linking feature:\n%s", gidx)
	}
	if !strings.Contains(out.String(), "done: feature features/payments/instant-refunds is a draft") {
		t.Errorf("the feature is named by its full ID:\n%s", out.String())
	}
	var vout bytes.Buffer
	if exit := bundle.Validate(root, bundle.Options{Out: &vout}); exit != 0 {
		t.Fatalf("bundle with new feature not conformant:\n%s", vout.String())
	}
	if code := New(root, "payments/instant-refunds", &out); code != 1 {
		t.Fatal("re-new same id must fail")
	}
	if code := New(root, "features/payments/instant-refunds", &out); code != 1 {
		t.Fatal("the full ID names the same feature, which exists")
	}
	if code := New(root, "Payments/Bad", &out); code != 1 {
		t.Fatal("uppercase id must fail")
	}
}

// Features are filed flat or in groups nested to any depth (spec 1.0). Each
// new group is listed once, in its parent's index; the root INDEX.md lists
// the registers, and no group.
func TestNewListsEveryNewGroupInItsParent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(root, &out)
	fillContext(t, root)
	before, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	out.Reset()
	for _, id := range []string{"onboarding", "payments/instant-refunds", "payments/refund-status", "platform/payouts/weekly-payouts"} {
		if code := New(root, id, &out); code != 0 {
			t.Fatalf("fdf new %s: exit %d\n%s", id, code, out.String())
		}
	}
	if code := Adopt(root, "", "platform/payouts/card-payouts", []string{"main.go"}, &out); code != 0 {
		t.Fatalf("fdf adopt: exit %d\n%s", code, out.String())
	}
	for rel, want := range map[string]string{
		"features/INDEX.md": "* [Format reference](/SPEC.md) - how features are structured.\n" +
			"* [Onboarding](/features/onboarding.md) - TODO.\n" +
			"* [Payments](/features/payments/INDEX.md) - features in payments.\n" +
			"* [Platform](/features/platform/INDEX.md) - features in platform.\n",
		"features/platform/INDEX.md":         "# Platform\n\n* [Payouts](/features/platform/payouts/INDEX.md) - features in payouts.\n",
		"features/platform/payouts/INDEX.md": "# Payouts\n\n* [Weekly payouts](/features/platform/payouts/weekly-payouts.md) - TODO.\n* [Card payouts](/features/platform/payouts/card-payouts.md) - TODO.\n",
	} {
		if raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))); !strings.Contains(string(raw), want) {
			t.Errorf("%s should hold:\n%s\ngot:\n%s", rel, want, raw)
		}
	}
	if n := strings.Count(out.String(), `updated features/INDEX.md (now lists "Payments")`); n != 1 {
		t.Errorf("a group is listed once, got %d:\n%s", n, out.String())
	}
	if after, _ := os.ReadFile(filepath.Join(root, "INDEX.md")); string(after) != string(before) {
		t.Errorf("the root INDEX.md lists registers, not groups:\n%s", after)
	}
	var vout bytes.Buffer
	if exit := bundle.Validate(root, bundle.Options{Out: &vout}); exit != 0 {
		t.Fatalf("bundle not conformant:\n%s", vout.String())
	}
}

// A group goes with the groups an index already lists, whatever the index
// looks like; an index that lists it already is left as it is.
func TestWithGroupListingPlacesTheGroup(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"after the last group",
			"# Features\n\n* [Payments](/features/payments/INDEX.md) - payments.\n* [Onboarding](/features/onboarding.md) - feature.\n",
			"# Features\n\n* [Payments](/features/payments/INDEX.md) - payments.\n* [Venues](/features/venues/INDEX.md) - features in venues.\n* [Onboarding](/features/onboarding.md) - feature.\n"},
		{"a subgroup is not a sibling",
			"# Features\n\n* [Deep](/features/a/deep/INDEX.md) - a subgroup's.\n",
			"# Features\n\n* [Deep](/features/a/deep/INDEX.md) - a subgroup's.\n* [Venues](/features/venues/INDEX.md) - features in venues.\n"},
		{"at the end",
			"# Features\n\nNo list yet.\n",
			"# Features\n\nNo list yet.\n\n* [Venues](/features/venues/INDEX.md) - features in venues.\n"},
		{"listed by its directory",
			"# Features\n\n* [Venues](venues/) - venues.\n",
			"# Features\n\n* [Venues](venues/) - venues.\n"},
	} {
		got, _ := WithGroupListing(tc.text, "features", "venues")
		if got != tc.want {
			t.Errorf("%s:\n got: %q\nwant: %q", tc.name, got, tc.want)
		}
	}
	got, added := WithGroupListing("# Debt\n\n* [Format reference](/SPEC.md) - how debts are structured.\n* [Platform](/debts/platform/INDEX.md) - debts in platform.\n* [Gap](/debts/gap.md) - debt.\n", "debts", "venues")
	if want := "* [Platform](/debts/platform/INDEX.md) - debts in platform.\n* [Venues](/debts/venues/INDEX.md) - debts in venues.\n* [Gap]"; !added || !strings.Contains(got, want) {
		t.Errorf("a register's group goes with its groups:\n%s", got)
	}
}

// A new practice is listed in practices/INDEX.md, or in its group's index,
// which is created and listed on first use.
// Every placeholder is one whole line: deleting an optional section's TODO
// line leaves nothing of it behind.
func TestPracticePlaceholdersAreWholeLines(t *testing.T) {
	root := pinned(t, currentVersion)
	var out bytes.Buffer
	if code := Practice(root, "permission-checks", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "practices", "permission-checks.md"))
	lines := strings.Split(string(raw), "\n")
	for i := 1; i < len(lines); i++ {
		if strings.Contains(lines[i-1], "TODO —") && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(lines[i], "#") && !strings.Contains(lines[i], ":") {
			t.Errorf("%q continues a TODO line:\n%s", lines[i], raw)
		}
	}
}

func TestPracticeIsListed(t *testing.T) {
	root := pinned(t, currentVersion)
	var out bytes.Buffer
	for _, id := range []string{"permission-checks", "payments/capture"} {
		if code := Practice(root, id, &out); code != 0 {
			t.Fatalf("fdf practice %s: exit %d\n%s", id, code, out.String())
		}
	}
	top, _ := os.ReadFile(filepath.Join(root, "practices", "INDEX.md"))
	for _, want := range []string{
		"* [Permission checks](/practices/permission-checks.md) - practice.\n",
		"* [Payments](/practices/payments/INDEX.md) - practices in payments.\n",
	} {
		if !strings.Contains(string(top), want) {
			t.Errorf("practices/INDEX.md should contain %q:\n%s", want, top)
		}
	}
	group, _ := os.ReadFile(filepath.Join(root, "practices", "payments", "INDEX.md"))
	if want := "# Payments\n\n* [Capture](/practices/payments/capture.md) - practice.\n"; string(group) != want {
		t.Errorf("practices/payments/INDEX.md:\n%s\nwant:\n%s", group, want)
	}
	// Each listing is reported the way `fdf new` reports its own.
	for _, want := range []string{
		`updated practices/INDEX.md (now lists "Permission checks")`,
		`updated practices/INDEX.md (now lists "Payments")`,
		`updated practices/payments/INDEX.md (now lists "Capture")`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should say %q:\n%s", want, out.String())
		}
	}
}

// The full ID files a practice where it says, not under practices/practices/.
func TestPracticeTakesTheFullID(t *testing.T) {
	root := pinned(t, currentVersion)
	var out bytes.Buffer
	if code := Practice(root, "practices/permission-checks", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "practices", "permission-checks.md")); err != nil {
		t.Fatalf("practices/permission-checks.md was not written:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "practices", "practices")); err == nil {
		t.Fatalf("practices/practices/ must not exist:\n%s", out.String())
	}
}

// pinned returns a temp bundle root whose INDEX.md pins pin ("" for none).
func pinned(t *testing.T, pin string) string {
	t.Helper()
	root := t.TempDir()
	index := "# Bundle\n\n* [Spec](/SPEC.md) - the format.\n"
	if pin != "" {
		index = "---\nfdf_version: \"" + pin + "\"\n---\n\n" + index
	}
	if err := os.WriteFile(filepath.Join(root, "INDEX.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// No name is reserved inside a register (spec 1.0): a feature group may be
// called bugs/ or changes/, and a new register never renames one again.
func TestNoNameIsReservedInsideFeatures(t *testing.T) {
	root := pinned(t, currentVersion)
	for _, id := range []string{"debts/crash-report", "bugs/triage", "practices/audit", "changes/review", "releases/notes", "features"} {
		var out bytes.Buffer
		if code := New(root, id, &out); code != 0 {
			t.Errorf("fdf new %s: exit %d\n%s", id, code, out.String())
		}
		if _, err := os.Stat(filepath.Join(root, "features", filepath.FromSlash(id)+".md")); err != nil {
			t.Errorf("fdf new %s writes features/%s.md: %v", id, id, err)
		}
	}
}

// fdf new refuses exactly what the validator would reject: each place it
// refuses, a feature written there by hand fails F3. The commands and the
// validator read positions from one place, layout.
func TestNewRefusesWhatTheValidatorRejects(t *testing.T) {
	feature := "---\ntype: Feature\nstatus: draft\ntitle: X\ndescription: d.\ntimestamp: 2026-09-25T00:00:00Z\n---\n\n" +
		"```gherkin\nFeature: X\n  As a user\n  I want x\n  So that y\n```\n\n```gherkin\nScenario: It works\n  Given x\n  When y\n  Then z\n```\n"
	for _, tc := range []struct{ name, refusal string }{
		{"onboarding/welcome", "features/onboarding/ is the task directory of features/onboarding"},
		{"payments", "features/payments/ is a group, and a feature named payments would make it its task directory (F3)"},
		{"Billing/refunds", `"Billing" in features/Billing/refunds is not a name`},
		{"billing/index", "index is not a slug: a disk that ignores case reads index.md as the INDEX.md beside it (F3)"},
		{"platform/log", "log is not a slug: a disk that ignores case reads log.md as the LOG.md beside it (F3)"},
	} {
		root := filepath.Join(t.TempDir(), "docs", "fdf")
		var out bytes.Buffer
		Init(root, &out)
		fillContext(t, root)
		for _, id := range []string{"onboarding", "payments/instant-refunds"} {
			if code := New(root, id, &out); code != 0 {
				t.Fatalf("fdf new %s: exit %d\n%s", id, code, out.String())
			}
		}
		out.Reset()
		if code := New(root, tc.name, &out); code != 1 || !strings.Contains(out.String(), tc.refusal) {
			t.Errorf("fdf new %s: exit %d, want a refusal saying %q:\n%s", tc.name, code, tc.refusal, out.String())
		}
		p := filepath.Join(root, "features", filepath.FromSlash(tc.name)+".md")
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(feature), 0o644)
		var vout bytes.Buffer
		if bundle.Validate(root, bundle.Options{Out: &vout}) == 0 || !strings.Contains(vout.String(), "(F3)") {
			t.Errorf("features/%s.md written by hand should fail F3:\n%s", tc.name, vout.String())
		}
	}
}

// A group directory spelled in capitals that holds no Markdown, one of
// images, is outside FDF; but on a disk that ignores case it is where fdf new
// would file features/payments/refunds.md, and the bundle would then fail
// F3. fdf new refuses it, naming the directory as it is spelled, and writes
// nothing.
func TestNewRefusesAGroupSpelledInAnotherCase(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(root, &out)
	fillContext(t, root)
	os.MkdirAll(filepath.Join(root, "features", "Payments"), 0o755)
	os.WriteFile(filepath.Join(root, "features", "Payments", "diagram.png"), []byte("PNG"), 0o644)
	before := tree(t, root)
	out.Reset()
	want := "error: features/Payments/ is already there: a disk that ignores case would file features/payments/refunds in it, and directory names are lowercase (F3); rename that directory, or choose another name\n"
	if code := New(root, "payments/refunds", &out); code != 1 || out.String() != want {
		t.Errorf("fdf new payments/refunds: exit %d\n got: %q\nwant: %q", code, out.String(), want)
	}
	if after := tree(t, root); after != before {
		t.Errorf("a refused feature writes nothing:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// A command never writes a new document over a file that is there. Place
// reads names exactly, and on a disk that ignores case a file whose name
// differs only in case is where the new one would go: WriteNew leaves the
// file system to refuse it.
func TestWriteNewNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "features", "refunds.md")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("kept\n"), 0o644)
	if err := WriteNew(root, "features/refunds", "new\n"); err == nil || err.Error() != "a file is already at features/refunds.md; on a disk that ignores case, its name may differ in case" {
		t.Errorf("WriteNew over a file: %v", err)
	}
	if raw, _ := os.ReadFile(p); string(raw) != "kept\n" {
		t.Errorf("the file there is untouched, got %q", raw)
	}
	if err := WriteNew(root, "features/checkout", "new\n"); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(filepath.Join(root, "features", "checkout.md")); string(raw) != "new\n" {
		t.Errorf("a new document is written, got %q", raw)
	}
}

// The commands write spec 1.x, so they refuse any other bundle before they
// write anything, and name the fix: `fdf migrate` for a 0.x pin, the pin or
// `fdf migrate` for none, a newer fdf for a newer pin, the pin itself when it
// is not a version, and the frontmatter when no `---` line closes it.
func TestScaffoldsPointA0xBundleAtMigrate(t *testing.T) {
	for _, tc := range []struct{ pin, says string }{
		{"0.7", "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n"},
		{"0.5", "error: this bundle pins fdf_version 0.5; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n"},
		{"", "error: this bundle's INDEX.md pins no fdf_version; fdf's commands work on spec 1.0 bundles — pin the version it was written for, as fdf_version: \"" + currentVersion + "\", or upgrade a bundle from before 1.0 with `fdf migrate` first\n"},
		{"1.3", "error: this bundle pins fdf_version 1.3, newer than any spec this fdf knows (1.0) — upgrade fdf\n"},
		// A pin that is almost 1.0 is no 0.x version to migrate.
		{"1.0.0", "error: this bundle pins fdf_version 1.0.0, which is not a MAJOR.MINOR version such as " + currentVersion + " — correct the pin in INDEX.md\n"},
		{"v1.0", "error: this bundle pins fdf_version v1.0, which is not a MAJOR.MINOR version such as " + currentVersion + " — correct the pin in INDEX.md\n"},
	} {
		for name, run := range map[string]func(root string, out *bytes.Buffer) int{
			"new": func(root string, out *bytes.Buffer) int { return New(root, "payments/refunds", out) },
			"adopt": func(root string, out *bytes.Buffer) int {
				return Adopt(root, "", "payments/cards", []string{"main.go"}, out)
			},
			"practice": func(root string, out *bytes.Buffer) int { return Practice(root, "permission-checks", out) },
		} {
			root := pinned(t, tc.pin)
			var out bytes.Buffer
			if code := run(root, &out); code != 1 || out.String() != tc.says {
				t.Errorf("fdf %s on pin %q: exit %d\n got: %q\nwant: %q", name, tc.pin, code, out.String(), tc.says)
			}
			if entries, _ := os.ReadDir(root); len(entries) != 1 {
				t.Errorf("fdf %s on pin %q writes nothing, but the bundle holds %d entries", name, tc.pin, len(entries))
			}
		}
	}
	// Its pin line may be there, so a missing pin would not say why.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "INDEX.md"), []byte("---\nfdf_version: \""+currentVersion+"\"\n\n# Bundle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := New(root, "payments/refunds", &out); code != 1 || out.String() != "error: this bundle's INDEX.md frontmatter has no closing `---` line, so it pins no fdf_version — end the block with one\n" {
		t.Errorf("fdf new on frontmatter that never closes: exit %d\n%s", code, out.String())
	}
}

// A register's or a group's INDEX.md pins nothing, so a root pointed at one,
// --root docs/fdf/features for docs/fdf, used to be sent to `fdf migrate`,
// which built a second bundle inside the first. The commands, fdf init among
// them, name the bundle instead, and write nothing. So does a directory in
// the bundle that holds no INDEX.md, or does not exist yet, where fdf init
// used to build a second bundle.
func TestScaffoldsSendARootInsideABundleToIt(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(bundle, &out)
	fillContext(t, bundle)
	if code := New(bundle, "payments/refunds", &out); code != 0 {
		t.Fatalf("fdf new: exit %d\n%s", code, out.String())
	}
	if err := os.MkdirAll(filepath.Join(bundle, "features", "payments", "diagrams"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := tree(t, bundle)
	for _, root := range []string{
		filepath.Join(bundle, "features"),
		filepath.Join(bundle, "features", "payments"),
		filepath.Join(bundle, "features", "payments", "diagrams"),
		filepath.Join(bundle, "features", "cards"),
	} {
		want := "error: " + root + " is inside the bundle at " + bundle + ", not a bundle of its own — pass --root " + bundle + ", or leave --root out\n"
		for name, run := range map[string]func(root string, out *bytes.Buffer) int{
			"init": func(root string, out *bytes.Buffer) int { return Init(root, out) },
			"new":  func(root string, out *bytes.Buffer) int { return New(root, "cards", out) },
			"adopt": func(root string, out *bytes.Buffer) int {
				return Adopt(root, "", "cards", []string{"main.go"}, out)
			},
			"practice": func(root string, out *bytes.Buffer) int { return Practice(root, "permission-checks", out) },
		} {
			out.Reset()
			if code := run(root, &out); code != 1 || out.String() != want {
				t.Errorf("fdf %s on %s: exit %d\n got: %q\nwant: %q", name, root, code, out.String(), want)
			}
		}
	}
	if after := tree(t, bundle); after != before {
		t.Errorf("nothing is written inside the bundle:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// tree lists every file under root with its content, in path order.
func tree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			raw, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(root, p)
			b.WriteString("== " + filepath.ToSlash(rel) + "\n" + string(raw))
		}
		return nil
	})
	return b.String()
}

// fdf init starts a bundle only where there is none. A directory that holds
// Markdown but no INDEX.md, such as a bundle from before 1.0 that never had
// one, keeps every file, its LOG.md among them, and is sent to fdf migrate.
// A README.md, which the root may hold, is no reason to refuse, nor is what
// is hidden.
func TestInitRefusesADirectoryThatHoldsMarkdown(t *testing.T) {
	root := t.TempDir()
	log := "# Log\n\n## 2026-01-01\n* Kept.\n"
	for rel, text := range map[string]string{"LOG.md": log, "wdise/example.md": "# Example\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := tree(t, root)
	var out bytes.Buffer
	if code := Init(root, &out); code != 1 || !strings.Contains(out.String(), "but no INDEX.md") || !strings.Contains(out.String(), "`fdf migrate --root "+root+"`") {
		t.Errorf("init refuses and names fdf migrate: exit %d\n%s", code, out.String())
	}
	if tree(t, root) != before {
		t.Error("a refused init writes nothing")
	}
	for _, rel := range []string{"README.md", ".notes/draft.md"} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte("# Docs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if code := Init(dir, &out); code != 0 {
			t.Errorf("init beside %s: exit %d\n%s", rel, code, out.String())
		}
	}
}

// A bundle's spec copy is the spec its pin names, which a later minor makes
// older than the current one: EnsureSpec writes the version it is given,
// and init gives it the pin, and writes nothing over a copy that is there.
func TestEnsureSpecWritesTheVersionItIsGiven(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if code := EnsureSpec(root, "0.7", &out); code != 0 {
		t.Fatalf("EnsureSpec: exit %d\n%s", code, out.String())
	}
	spec, err := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if err != nil || !strings.Contains(string(spec), "The FDF v0.7 specification this bundle conforms to.") || !strings.Contains(out.String(), "wrote SPEC.md (FDF v0.7 spec copy)") {
		t.Errorf("SPEC.md is the 0.7 spec: %v\n%s", err, out.String())
	}
	out.Reset()
	if code := EnsureSpec(root, CurrentVersion(), &out); code != 0 || out.String() != "" {
		t.Errorf("a copy that is there is kept: exit %d\n%s", code, out.String())
	}
}

// Re-running init on a current bundle adds back what is missing — the
// reserved directories' indexes included — and overwrites nothing.
func TestInitBackfillsMissingIndexes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "fdf")
	var out bytes.Buffer
	Init(root, &out)
	if !strings.Contains(out.String(), "`fdf validate` warns about them now, and fails F9 once a feature exists") {
		t.Errorf("init should say when F9 fails:\n%s", out.String())
	}
	os.Remove(filepath.Join(root, "bugs", "INDEX.md"))
	os.Remove(filepath.Join(root, "features", "INDEX.md"))
	out.Reset()
	if code := Init(root, &out); code != 0 || !strings.Contains(out.String(), "wrote bugs/INDEX.md") || !strings.Contains(out.String(), "wrote features/INDEX.md") {
		t.Fatalf("re-init should restore bugs/INDEX.md and features/INDEX.md: exit %d\n%s", code, out.String())
	}
}
