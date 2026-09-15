package changes

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bundle(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("payments/instant-refunds.md", "---\ntype: Feature\ntitle: Instant refunds\nstatus: done\n---\n\n# Feature\n")
	return root
}

func TestNewFixScaffoldsRegressionSectionAndIndex(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "refund-rounding", "Fix", []string{"payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "refund-rounding.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"type: Fix", "status: draft", "affects: payments/instant-refunds",
		"# Regression cases", "## payments/instant-refunds"} {
		if !strings.Contains(body, want) {
			t.Errorf("scaffolded Fix missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# Scenario changes") {
		t.Error("a Fix must not carry a Change's declaration section")
	}
	idx, _ := os.ReadFile(filepath.Join(root, "changes", "INDEX.md"))
	if !strings.Contains(string(idx), "refund-rounding.md") {
		t.Errorf("changes/INDEX.md does not list the new fix:\n%s", idx)
	}
}

func TestNewChangeIsGroupedAndCarriesScenarioChanges(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := New(root, "payments/refund-window", "Change", []string{"payments/instant-refunds"}, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "changes", "payments", "refund-window.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"type: Change", "# Scenario changes", "- add:", "- modify:", "- remove:"} {
		if !strings.Contains(body, want) {
			t.Errorf("scaffolded Change missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# Regression cases") {
		t.Error("a Change must not carry a Fix's declaration section")
	}
	if _, err := os.Stat(filepath.Join(root, "changes", "payments", "INDEX.md")); err != nil {
		t.Error("a changes/ group must get its own INDEX.md")
	}
}

// affects is the whole link between a change and the features it touches, so
// a typo there would silently produce an orphan document.
func TestNewRejectsUnknownOrMissingAffects(t *testing.T) {
	root := bundle(t)
	for _, tc := range []struct {
		name    string
		affects []string
		want    string
	}{
		{"none", nil, "--affects is required"},
		{"malformed", []string{"instant-refunds"}, "feature IDs of the form"},
		{"unknown", []string{"payments/nope"}, "not a feature in this bundle"},
	} {
		var out bytes.Buffer
		if code := New(root, "c-"+tc.name, "Fix", tc.affects, &out); code == 0 {
			t.Errorf("%s: expected refusal", tc.name)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("%s: want %q, got %q", tc.name, tc.want, out.String())
		}
	}
}

func TestHistoryFindsChangesByAffects(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	New(root, "refund-rounding", "Fix", []string{"payments/instant-refunds"}, &out)
	New(root, "payments/refund-window", "Change", []string{"payments/instant-refunds"}, &out)

	out.Reset()
	if code := History(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	got := out.String()
	for _, want := range []string{"2 post-delivery document(s)", "changes/refund-rounding", "changes/payments/refund-window", "Fix", "Change"} {
		if !strings.Contains(got, want) {
			t.Errorf("history missing %q:\n%s", want, got)
		}
	}
}

func TestHistoryOnUntouchedFeatureSaysSo(t *testing.T) {
	root := bundle(t)
	var out bytes.Buffer
	if code := History(root, "payments/instant-refunds", &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "no changes or fixes since delivery") {
		t.Errorf("unexpected: %s", out.String())
	}
}
