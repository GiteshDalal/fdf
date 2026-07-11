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
	for _, name := range []string{"STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md"} {
		body := "---\ntype: Context\ntitle: " + name + "\ndescription: filled.\n" +
			"timestamp: 2026-07-06T00:00:00Z\n---\n\n# " + name + "\n\nReal content.\n"
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
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
	if !strings.Contains(string(idx), `fdf_version: "0.4"`) {
		t.Fatalf("pin: %s", idx)
	}
}

func TestInitScaffoldsConformingBundle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	var out bytes.Buffer
	if code := Init(root, &out); code != 0 {
		t.Fatalf("init: %d\n%s", code, out.String())
	}
	for _, f := range []string{"INDEX.md", "LOG.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(raw), `fdf_version: "0.4"`) ||
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
	os.WriteFile(idx, bytes.Replace(raw, []byte(`"0.4"`), []byte(`"0.1"`), 1), 0o644)
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
