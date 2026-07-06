package bundle

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestSubmoduleMountResolvesResourcesAgainstSuperproject(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmp := t.TempDir()

	// Bundle repo: copy the valid-done fixture, give 01-do-thing.md a resource.
	bundleRepo := filepath.Join(tmp, "features-repo")
	copyTree(t, filepath.Join("..", "..", "..", "testdata", "valid-done", "bundle"), bundleRepo)
	task := filepath.Join(bundleRepo, "wdise", "example", "01-do-thing.md")
	raw, _ := os.ReadFile(task)
	patched := strings.Replace(string(raw), "status: done\n", "status: done\nresource: services/thing\n", 1)
	if err := os.WriteFile(task, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, bundleRepo, "init", "-q")
	git(t, bundleRepo, "add", "-A")
	git(t, bundleRepo, "commit", "-qm", "bundle")

	// Superproject with the code the resource points at.
	super := filepath.Join(tmp, "super")
	if err := os.MkdirAll(filepath.Join(super, "services", "thing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(super, "services", "thing", "main.go"), []byte("package thing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, super, "init", "-q")
	git(t, super, "add", "-A")
	git(t, super, "commit", "-qm", "code")
	git(t, super, "-c", "protocol.file.allow=always", "submodule", "add", "-q", bundleRepo, "docs/features")
	git(t, super, "commit", "-qm", "mount bundle")

	mounted := filepath.Join(super, "docs", "features")
	root, standalone := fdfroot.ProjectRoot(mounted)
	if standalone || root != super {
		t.Fatalf("ProjectRoot(%s) = %q standalone=%v; want superproject %q", mounted, root, standalone, super)
	}
	var out bytes.Buffer
	if exit := Validate(mounted, Options{RepoRoot: root, Out: &out}); exit != 0 {
		t.Fatalf("expected conformant, got exit %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "R1) verified") {
		t.Fatalf("expected R1 verified, got:\n%s", out.String())
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
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
}
