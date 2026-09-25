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
	mk("INDEX.md", "---\nfdf_version: \"1.0\"\n---\n\n# Bundle\n\n"+
		"* [Features](/features/INDEX.md) - what the software does.\n"+
		"* [Changes](/changes/INDEX.md) - work on delivered features.\n"+
		"* [Format reference](/SPEC.md) - the spec this bundle pins.\n")
	mk("features/payments/instant-refunds.md", "---\ntype: Feature\ntitle: Instant refunds\nstatus: "+featureStatus+"\nversion: \"1.1.0\"\n---\n\n# Feature\n")
	mk("features/payments/instant-refunds.plan.md", "---\ntype: Feature\ntitle: Plan\nversion: \"1.1.0\"\n---\n\n# Tasks\n")
	mk("changes/refund-rounding.md", "---\ntype: Fix\ntitle: Refund rounding\nstatus: "+changeStatus+"\naffects: features/payments/instant-refunds\nversion: \"1.1.0\"\n---\n\n# Symptom\n")
	mk("features/payments/other.md", "---\ntype: Feature\ntitle: Other\nstatus: done\n---\n\n# Feature\n")
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
	for _, want := range []string{"status: planned", "# Features\n\n* [Instant refunds](/features/payments/instant-refunds.md) - done.\n",
		"# Changes\n\n* [Refund rounding](/changes/refund-rounding.md) - fix (done).\n"} {
		if !strings.Contains(body, want) {
			t.Errorf("release doc missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "payments/other.md") {
		t.Error("a document without a matching `version` must not be listed")
	}
	if strings.Contains(body, "instant-refunds.plan") {
		t.Error("a trail document is not a member, whatever its frontmatter says")
	}
}

// A version names the release's file, so it must be a name the 1.0 layout
// gives a release: lowercase, flat in releases/, and not the register's own
// index or log.
func TestSyncRefusesAVersionNoReleaseFileCanTake(t *testing.T) {
	root := bundle(t, "done", "done")
	for version, says := range map[string]string{
		"2.0.0-RC1": "error: 2.0.0-RC1 cannot name a release — releases/2.0.0-RC1.md: filenames are lowercase",
		"2.0/beta":  "error: 2.0/beta cannot name a release — releases/2.0/: releases/ is flat",
		"index":     "error: index cannot name a release — releases/index.md: no document is named index.md",
		"INDEX":     "error: INDEX cannot name a release — releases/INDEX.md is the register's index, not a release (F3)\n",
		"LOG":       "error: LOG cannot name a release — releases/LOG.md is the register's log, not a release (F3)\n",
	} {
		var out bytes.Buffer
		if code := Sync(root, version, "", false, &out); code != 2 || !strings.Contains(out.String(), says) {
			t.Errorf("fdf release %s: exit %d, want 2 saying %q:\n%s", version, code, says, out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "releases")); err == nil {
		t.Error("a refused release writes nothing")
	}
}

// The first release creates releases/, and the root INDEX.md lists it with
// the other registers, once.
func TestTheFirstReleaseListsTheRegisterInTheRootIndex(t *testing.T) {
	root := bundle(t, "done", "done")
	var out bytes.Buffer
	for _, v := range []string{"1.1.0", "1.1.0"} {
		if code := Sync(root, v, "", false, &out); code != 0 {
			t.Fatalf("exit %d\n%s", code, out.String())
		}
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	want := "* [Changes](/changes/INDEX.md) - work on delivered features.\n" +
		"* [Releases](/releases/INDEX.md) - what shipped in each version.\n" +
		"* [Format reference](/SPEC.md) - the spec this bundle pins.\n"
	if !strings.Contains(string(idx), want) || strings.Count(out.String(), `updated INDEX.md (now lists "Releases")`) != 1 {
		t.Errorf("the root INDEX.md lists releases/ after the other registers, once:\n%s\n%s", idx, out.String())
	}
}

// fdf release works on spec 1.0 bundles: a 0.x bundle is upgraded with
// `fdf migrate` first.
func TestSyncPointsA0xBundleAtMigrate(t *testing.T) {
	root := bundle(t, "done", "done")
	os.WriteFile(filepath.Join(root, "INDEX.md"), []byte("---\nfdf_version: \"0.7\"\n---\n\n# Bundle\n"), 0o644)
	var out bytes.Buffer
	if code := Sync(root, "1.1.0", "", false, &out); code != 1 ||
		out.String() != "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n" {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "releases")); err == nil {
		t.Error("a refused release writes nothing")
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

// releases/INDEX.md says it is newest first, so a later release is listed
// above an earlier one rather than appended below it.
func TestIndexListsNewestFirst(t *testing.T) {
	root := bundle(t, "done", "done")
	os.WriteFile(filepath.Join(root, "features", "payments", "other.md"), []byte("---\ntype: Feature\ntitle: Other\nstatus: done\nversion: \"1.2.0\"\n---\n\n# Feature\n"), 0o644)
	var out bytes.Buffer
	for _, v := range []string{"1.1.0", "1.2.0"} {
		if code := Sync(root, v, "", false, &out); code != 0 {
			t.Fatalf("release %s: exit %d\n%s", v, code, out.String())
		}
	}
	idx, _ := os.ReadFile(filepath.Join(root, "releases", "INDEX.md"))
	want := "# Releases\n\nNewest first.\n\n* [1.2.0](/releases/1.2.0.md) - release.\n* [1.1.0](/releases/1.1.0.md) - release.\n"
	if string(idx) != want {
		t.Errorf("releases/INDEX.md:\n%s\nwant:\n%s", idx, want)
	}
	if !strings.Contains(out.String(), `updated releases/INDEX.md (now lists "1.2.0")`) {
		t.Errorf("the listing is reported:\n%s", out.String())
	}
}
