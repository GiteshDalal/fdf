package release

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bundle(t *testing.T, featureStatus, changeStatus string) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("payments/instant-refunds.md", "---\ntype: Feature\ntitle: Instant refunds\nstatus: "+featureStatus+"\nversion: \"1.1.0\"\n---\n\n# Feature\n")
	mk("changes/refund-rounding.md", "---\ntype: Fix\ntitle: Refund rounding\nstatus: "+changeStatus+"\naffects: payments/instant-refunds\nversion: \"1.1.0\"\n---\n\n# Symptom\n")
	mk("payments/other.md", "---\ntype: Feature\ntitle: Other\nstatus: done\n---\n\n# Feature\n")
	return root
}

func TestSyncDerivesBothListsAndSkipsUnversioned(t *testing.T) {
	root := bundle(t, "done", "done")
	var out bytes.Buffer
	if code := Sync(root, "1.1.0", "", false, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "releases", "1.1.0.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{"status: planned", "# Features", "payments/instant-refunds.md", "# Changes", "changes/refund-rounding.md"} {
		if !strings.Contains(body, want) {
			t.Errorf("release doc missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "payments/other.md") {
		t.Error("a document without a matching `version` must not be listed")
	}
}

// `# Notes` is human prose; regenerating the derived lists must not eat it.
func TestSyncPreservesNotes(t *testing.T) {
	root := bundle(t, "done", "done")
	var out bytes.Buffer
	Sync(root, "1.1.0", "", false, &out)
	path := filepath.Join(root, "releases", "1.1.0.md")
	raw, _ := os.ReadFile(path)
	os.WriteFile(path, []byte(strings.Replace(string(raw), "# Notes\n\nTODO — optional.", "# Notes\n\nShipped behind a flag for two weeks.", 1)), 0o644)

	if code := Sync(root, "1.1.0", "", false, &out); code != 0 {
		t.Fatalf("second sync failed:\n%s", out.String())
	}
	after, _ := os.ReadFile(path)
	if !strings.Contains(string(after), "Shipped behind a flag for two weeks.") {
		t.Errorf("hand-written notes were lost:\n%s", after)
	}
}

func TestShipRefusesWhileAnythingIsOpen(t *testing.T) {
	root := bundle(t, "done", "draft")
	var out bytes.Buffer
	if code := Sync(root, "1.1.0", "", true, &out); code == 0 {
		t.Fatal("expected a refusal while a listed document is open")
	}
	if !strings.Contains(out.String(), "cannot ship") || !strings.Contains(out.String(), "refund-rounding") {
		t.Errorf("refusal should name what is open:\n%s", out.String())
	}
}

func TestShipStampsStatusAndIndex(t *testing.T) {
	root := bundle(t, "done", "done")
	var out bytes.Buffer
	if code := Sync(root, "1.1.0", "2026-10-01", true, &out); code != 0 {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "releases", "1.1.0.md"))
	if !strings.Contains(string(raw), "status: shipped") || !strings.Contains(string(raw), "date: 2026-10-01") {
		t.Errorf("ship should stamp status and date:\n%s", raw)
	}
	idx, err := os.ReadFile(filepath.Join(root, "releases", "INDEX.md"))
	if err != nil || !strings.Contains(string(idx), "1.1.0.md") {
		t.Errorf("releases/INDEX.md should list the release: %v\n%s", err, idx)
	}
}

// Membership is a human decision; with nothing tagged there is nothing to derive.
func TestSyncRefusesWhenNothingCarriesTheVersion(t *testing.T) {
	root := bundle(t, "done", "done")
	var out bytes.Buffer
	if code := Sync(root, "9.9.9", "", false, &out); code == 0 {
		t.Fatal("expected a refusal with no members")
	}
	if !strings.Contains(out.String(), "membership is a human decision") {
		t.Errorf("should explain why:\n%s", out.String())
	}
}
