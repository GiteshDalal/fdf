package fdfroot

import (
	"os"
	"path/filepath"
	"testing"
)

func mk(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProjectRootPlainRepo(t *testing.T) {
	tmp := t.TempDir()
	mk(t, tmp, "proj", ".git")
	deep := mk(t, tmp, "proj", "a", "b")
	root, standalone := ProjectRoot(deep)
	if root != filepath.Join(tmp, "proj") || standalone {
		t.Fatalf("got %q standalone=%v", root, standalone)
	}
}

func TestProjectRootSubmoduleWalksToSuperproject(t *testing.T) {
	tmp := t.TempDir()
	mk(t, tmp, "super", ".git")
	sub := mk(t, tmp, "super", "docs", "features")
	// submodule boundary: .git FILE, not dir
	if err := os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: ../../.git/modules/features\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, standalone := ProjectRoot(sub)
	if root != filepath.Join(tmp, "super") || standalone {
		t.Fatalf("got %q standalone=%v", root, standalone)
	}
}

func TestProjectRootStandaloneBundleRepo(t *testing.T) {
	tmp := t.TempDir()
	// a bundle repo checked out alone: .git file with no enclosing repo
	bundle := mk(t, tmp, "features")
	if err := os.WriteFile(filepath.Join(bundle, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, standalone := ProjectRoot(bundle)
	if root != bundle || standalone {
		t.Fatalf("submodule-style checkout alone: root should be the bundle repo itself, not standalone; got %q %v", root, standalone)
	}
}

func TestProjectRootNoGitAnywhere(t *testing.T) {
	tmp := t.TempDir()
	d := mk(t, tmp, "x")
	root, standalone := ProjectRoot(d)
	if root != d || !standalone {
		t.Fatalf("got %q standalone=%v", root, standalone)
	}
}

func TestBundleRootPrecedence(t *testing.T) {
	tmp := t.TempDir()
	mk(t, tmp, ".git")
	t.Setenv("FDF_ROOT_DIR", "documents/features")
	got, err := BundleRoot("", tmp)
	if err != nil || got != filepath.Join(tmp, "documents", "features") {
		t.Fatalf("env: got %q err %v", got, err)
	}
	got, _ = BundleRoot("custom/loc", tmp)
	if got != filepath.Join(tmp, "custom", "loc") {
		t.Fatalf("flag beats env: got %q", got)
	}
	abs := filepath.Join(tmp, "elsewhere")
	got, _ = BundleRoot(abs, tmp)
	if got != abs {
		t.Fatalf("absolute: got %q", got)
	}
	t.Setenv("FDF_ROOT_DIR", "")
	got, _ = BundleRoot("", tmp)
	if got != filepath.Join(tmp, "docs", "features") {
		t.Fatalf("default: got %q", got)
	}
}

func TestNearestProjectRootPrefersInnerRepo(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "outer", "inner", "src")
	if err := os.MkdirAll(filepath.Join(dir, "outer", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "outer", "inner", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	got, standalone := NearestProjectRoot(inner)
	if standalone {
		t.Fatal("nested repo must not be standalone")
	}
	if want := filepath.Join(dir, "outer", "inner"); got != want {
		t.Fatalf("nearest root = %s, want inner repo %s", got, want)
	}
	// A .git FILE (worktree/submodule) also marks the nearest project.
	wt := filepath.Join(dir, "outer", "wt")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: ../.git/worktrees/wt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := NearestProjectRoot(wt); got != wt {
		t.Fatalf("worktree root = %s, want %s", got, wt)
	}
	if _, standalone := NearestProjectRoot(t.TempDir()); !standalone {
		t.Fatal("no .git above: must be standalone")
	}
}
