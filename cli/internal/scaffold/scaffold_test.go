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
	root := filepath.Join(dir, "docs", "features")
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
	root := filepath.Join(t.TempDir(), "docs", "features")
	var out bytes.Buffer
	if code := Init(root, &out); code != 0 {
		t.Fatalf("init: %d\n%s", code, out.String())
	}
	for _, f := range []string{"INDEX.md", "LOG.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(raw), `fdf_version: "`+currentVersion+`"`) ||
		!strings.Contains(string(raw), "/SPEC.md") {
		t.Fatalf("INDEX.md missing pin or spec link:\n%s", raw)
	}
	// Zero features: unfilled stubs are warnings only, so init is conformant.
	var vout bytes.Buffer
	if exit := bundle.Validate(root, bundle.Options{Out: &vout}); exit != 0 {
		t.Fatalf("scaffold not conformant:\n%s", vout.String())
	}
}

func TestInitIdempotentAndMigrateHint(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	var out bytes.Buffer
	Init(root, &out)
	out.Reset()
	if code := Init(root, &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-init: code %d out %q", code, out.String())
	}
	// Simulate an older bundle: rewrite the pin.
	idx := filepath.Join(root, "INDEX.md")
	raw, _ := os.ReadFile(idx)
	os.WriteFile(idx, bytes.Replace(raw, []byte(`"`+currentVersion+`"`), []byte(`"0.1"`), 1), 0o644)
	out.Reset()
	if code := Init(root, &out); code != 1 || !strings.Contains(out.String(), "fdf migrate") {
		t.Fatalf("older pin: code %d out %q", code, out.String())
	}
}

func TestNewScaffoldsDraftFeature(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	var out bytes.Buffer
	Init(root, &out)
	fillContext(t, root) // F9: a feature-bearing bundle needs filled Context docs
	if code := New(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("new: %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "payments", "instant-refunds.md"))
	for _, want := range []string{"type: Feature", "status: draft", "Feature: Instant refunds", "Scenario:"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("feature missing %q:\n%s", want, raw)
		}
	}
	gidx, _ := os.ReadFile(filepath.Join(root, "payments", "INDEX.md"))
	if !strings.Contains(string(gidx), "/payments/instant-refunds.md") {
		t.Fatalf("group index not linking feature:\n%s", gidx)
	}
	var vout bytes.Buffer
	if exit := bundle.Validate(root, bundle.Options{Out: &vout}); exit != 0 {
		t.Fatalf("bundle with new feature not conformant:\n%s", vout.String())
	}
	if code := New(root, "payments/instant-refunds", &out); code != 1 {
		t.Fatal("re-new same id must fail")
	}
	if code := New(root, "Payments/Bad", &out); code != 1 {
		t.Fatal("uppercase id must fail")
	}
}

// A new practice is listed in practices/INDEX.md, or in its group's index,
// which is created and listed on first use.
func TestPracticeIsListed(t *testing.T) {
	root := t.TempDir()
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
	root := t.TempDir()
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

// A reserved directory holds one kind of document; a feature there would be
// filed where validation and every other command look for something else.
func TestFeaturesRefuseAReservedGroup(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct{ id, names string }{
		{"debts/x", "fdf debt <slug>"},
		{"bugs/x", "fdf bug <slug>"},
		{"practices/x", "fdf practice <slug>"},
		{"changes/x", "fdf change or fdf fix"},
		{"releases/x", "fdf release <version>"},
	} {
		var out bytes.Buffer
		if code := New(root, tc.id, &out); code != 1 {
			t.Errorf("fdf new %s: exit %d, want 1\n%s", tc.id, code, out.String())
		}
		if code := Adopt(root, "", tc.id, []string{"main.go"}, &out); code != 1 {
			t.Errorf("fdf adopt %s: exit %d, want 1\n%s", tc.id, code, out.String())
		}
		if !strings.Contains(out.String(), tc.names) || !strings.Contains(out.String(), "a feature's group is any other name") {
			t.Errorf("%s: the refusal names %q:\n%s", tc.id, tc.names, out.String())
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tc.id)+".md")); err == nil {
			t.Errorf("%s.md must not be written", tc.id)
		}
	}
}

// Re-running init on a current bundle adds back what is missing — the
// reserved directories' indexes included — and overwrites nothing.
func TestInitBackfillsMissingIndexes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	var out bytes.Buffer
	Init(root, &out)
	if !strings.Contains(out.String(), "`fdf validate` warns about them now, and fails F9 once a feature exists") {
		t.Errorf("init should say when F9 fails:\n%s", out.String())
	}
	os.Remove(filepath.Join(root, "bugs", "INDEX.md"))
	out.Reset()
	if code := Init(root, &out); code != 0 || !strings.Contains(out.String(), "wrote bugs/INDEX.md") {
		t.Fatalf("re-init should restore bugs/INDEX.md: exit %d\n%s", code, out.String())
	}
}
