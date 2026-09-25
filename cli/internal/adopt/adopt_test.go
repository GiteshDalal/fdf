package adopt

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A project with two capabilities in code: one mapped (and backfilled), one
// not. The map must count the first and list the second as unclaimed.
func TestMapCountsFeaturesAndListsUnclaimedCode(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := t.TempDir()
	write(t, repo, "src/cards/charge.go", "package cards\n")
	write(t, repo, "src/cards/settle.go", "package cards\n")
	write(t, repo, "src/wallet/topup.go", "package wallet\n")
	write(t, repo, "src/wallet/README.md", "not code\n")
	write(t, repo, "go.mod", "module x\n")
	for _, f := range []string{"a.go", "b.go", "c.go", "d.go"} {
		write(t, repo, "src/api/"+f, "package api\n")
	}
	root := filepath.Join(repo, "docs", "fdf")
	write(t, root, "INDEX.md", "---\nfdf_version: \"1.0\"\n---\n\n# Bundle\n\n* [Features](/features/INDEX.md) - what the software does.\n")
	write(t, root, "features/payments/card-payments.md", "---\ntype: Feature\nstatus: adopted\nresource: [src/cards]\n---\n\n```gherkin\nFeature: Card payments\n```\n\n```gherkin\nScenario: A settled payment is marked settled\n  Given x\n```\n")
	write(t, root, "features/payments/card-payments.test.md", "---\ntype: Test\n---\n\n# Test Cases\n\n## A settled payment is marked settled\n\n`go test`\n")
	write(t, root, "features/csv-export.md", "---\ntype: Feature\nstatus: adopted\nresource: src/export\n---\n\n```gherkin\nFeature: Export\n```\n")
	// A task whose feature is gone sits where a feature goes, and is none.
	write(t, root, "features/platform/refunds/01-api.md", "---\ntype: Task\nstatus: pending\n---\n\n# API\n")
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repo
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	add := exec.Command("git", "add", "-A")
	add.Dir = repo
	if err := add.Run(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if code := Map(root, repo, 2, &out); code != 0 {
		t.Fatalf("map: %d\n%s", code, out.String())
	}
	s := out.String()
	for _, want := range []string{
		"adopted       features/payments/card-payments          1       1",
		"adopted       features/csv-export                      0       0",
		"2 feature(s): 0 built, 0 in flight, 0 retired, 2 adopted (1 with scenarios, 1 map entry).",
		"src/wallet/",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("map missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "src/cards/") {
		t.Fatalf("claimed code is not listed as unclaimed:\n%s", s)
	}
	if strings.Contains(s, "01-api") {
		t.Fatalf("an orphan task is not a feature:\n%s", s)
	}
	// A row of three or fewer unclaimed files names them; a bigger one only
	// counts them.
	rows := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		if f := strings.Fields(line); len(f) >= 3 {
			rows[f[0]] = line
		}
	}
	if !strings.HasSuffix(rows["src/wallet/"], "  topup.go") || !strings.HasSuffix(rows["."], "  go.mod") {
		t.Fatalf("a small row names its unclaimed files:\n%s", s)
	}
	if !strings.HasSuffix(rows["src/api/"], "4  4") {
		t.Fatalf("a row of four unclaimed files only counts them:\n%s", s)
	}
	if strings.Contains(s, "docs/fdf") {
		t.Fatalf("the bundle itself is not code to claim:\n%s", s)
	}
}

// The map works on spec 1.0 bundles: a 0.x bundle is upgraded with
// `fdf migrate` first.
func TestMapPointsA0xBundleAtMigrate(t *testing.T) {
	root := t.TempDir()
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.7\"\n---\n\n# Bundle\n")
	var out bytes.Buffer
	if code := Map(root, "", 2, &out); code != 1 ||
		out.String() != "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — run `fdf migrate` to upgrade it first\n" {
		t.Fatalf("exit %d\n%s", code, out.String())
	}
}
