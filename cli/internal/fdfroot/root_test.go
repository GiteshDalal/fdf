package fdfroot

import (
	"os"
	"path/filepath"
	"strings"
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

// A linked worktree (`git worktree add`) often sits inside the main
// repository's directory. Its .git file points at an admin directory holding
// `commondir`; the worktree is a checkout of its own, so the walk stops there
// instead of reaching the main checkout, whose files belong to another branch.
func TestProjectRootLinkedWorktreeStopsAtItsRoot(t *testing.T) {
	tmp := t.TempDir()
	main := mk(t, tmp, "repo")
	admin := mk(t, main, ".git", "worktrees", "wt")
	if err := os.WriteFile(filepath.Join(admin, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, gitdir := range []string{admin, filepath.Join("..", "..", "..", ".git", "worktrees", "wt")} {
		wt := mk(t, main, ".claude", "worktrees", "wt")
		if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+gitdir+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		deep := mk(t, wt, "docs", "features")
		root, standalone := ProjectRoot(deep)
		if root != wt || standalone {
			t.Fatalf("gitdir %q: want the worktree %q, got %q standalone=%v", gitdir, wt, root, standalone)
		}
		if bundle := Resolve("docs/features", wt).Root; bundle != filepath.Join(wt, "docs", "features") {
			t.Fatalf("gitdir %q: relative --root should resolve inside the worktree, got %q", gitdir, bundle)
		}
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
	if got := Resolve("", tmp).Root; got != filepath.Join(tmp, "documents", "features") {
		t.Fatalf("env: got %q", got)
	}
	if got := Resolve("custom/loc", tmp).Root; got != filepath.Join(tmp, "custom", "loc") {
		t.Fatalf("flag beats env: got %q", got)
	}
	abs := filepath.Join(tmp, "elsewhere")
	if got := Resolve(abs, tmp).Root; got != abs {
		t.Fatalf("absolute: got %q", got)
	}
	t.Setenv("FDF_ROOT_DIR", "")
	if got := Resolve("", tmp).Root; got != filepath.Join(tmp, "docs", "fdf") {
		t.Fatalf("default: got %q", got)
	}
}

// With neither --root nor FDF_ROOT_DIR, fdf takes the first of docs/fdf and
// docs/features that holds a bundle, so an upgraded fdf still finds one from
// before 1.0, and docs/fdf when neither does. When both hold one, docs/fdf
// wins and the other is named, so the header can warn. docs/features holds a
// bundle only when its INDEX.md pins a version: a documentation site's
// section page is not one, even on a disk that ignores case.
func TestDefaultFindsABundleAtEitherLocation(t *testing.T) {
	const pinned, page = "---\nfdf_version: \"0.7\"\n---\n", "# Features\n\nWhat the product does.\n"
	for _, tc := range []struct {
		name               string
		files              map[string]string
		root, source, seen string
	}{
		{"neither", nil, "docs/fdf", "default docs/fdf", ""},
		{"docs/fdf", map[string]string{"docs/fdf/INDEX.md": pinned}, "docs/fdf", "default docs/fdf", ""},
		{"docs/features", map[string]string{"docs/features/INDEX.md": pinned}, "docs/features", "pre-1.0 default docs/features", ""},
		{"both", map[string]string{"docs/fdf/INDEX.md": pinned, "docs/features/INDEX.md": pinned}, "docs/fdf", "default docs/fdf", "docs/features"},
		{"a site's index.md", map[string]string{"docs/features/index.md": pinned}, "docs/fdf", "default docs/fdf", ""},
		{"an INDEX.md with no pin", map[string]string{"docs/features/INDEX.md": page}, "docs/fdf", "default docs/fdf", ""},
		{"docs/fdf, and a site", map[string]string{"docs/fdf/INDEX.md": pinned, "docs/features/INDEX.md": page}, "docs/fdf", "default docs/fdf", ""},
	} {
		tmp := t.TempDir()
		for rel, text := range tc.files {
			p := filepath.Join(tmp, filepath.FromSlash(rel))
			if err := os.WriteFile(filepath.Join(mk(t, filepath.Dir(p)), filepath.Base(p)), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		r := Default(tmp)
		seen := ""
		if r.Shadowed != "" {
			seen = filepath.ToSlash(strings.TrimPrefix(r.Shadowed, tmp+string(filepath.Separator)))
		}
		if r.Root != filepath.Join(tmp, filepath.FromSlash(tc.root)) || r.Source != tc.source || seen != tc.seen {
			t.Errorf("%s: got %+v; want root %s (%s), shadowing %q", tc.name, r, tc.root, tc.source, tc.seen)
		}
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

func TestResolveNamesTheChooser(t *testing.T) {
	tmp := t.TempDir()
	for _, tc := range []struct{ flag, env, want string }{
		{"custom", "", "--root"},
		{"", "envdir", "FDF_ROOT_DIR"},
		{"", "", "default docs/fdf"},
		{"flagwins", "envdir", "--root"},
	} {
		t.Setenv("FDF_ROOT_DIR", tc.env)
		if source := Resolve(tc.flag, tmp).Source; source != tc.want {
			t.Errorf("flag=%q env=%q: source %q, want %q", tc.flag, tc.env, source, tc.want)
		}
	}
}

func TestCheckBundleWantsAnIndex(t *testing.T) {
	tmp := t.TempDir()
	err := CheckBundle(filepath.Join(tmp, "nope"))
	if err == nil || err.Error() != "no bundle at "+filepath.Join(tmp, "nope")+" (no INDEX.md) — run `fdf init` first, or point --root at the bundle" {
		t.Fatalf("a missing bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "INDEX.md"), []byte("---\nfdf_version: \"0.7\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckBundle(tmp); err != nil {
		t.Fatalf("a bundle with its INDEX.md: %v", err)
	}
}

// A register's or a group's INDEX.md pins nothing: a root there is inside
// the bundle whose INDEX.md pins its version, the nearest one above it, and
// every command says so in the same words. An INDEX.md that pins nothing,
// such as a documentation site's, is no bundle to be inside.
func TestBundleAboveFindsTheNearestPinnedIndex(t *testing.T) {
	tmp := t.TempDir()
	bundle := mk(t, tmp, "docs", "fdf")
	group := mk(t, bundle, "features", "payments")
	for p, text := range map[string]string{
		filepath.Join(tmp, "docs", "INDEX.md"):        "# Docs\n",
		filepath.Join(bundle, "INDEX.md"):             "---\nfdf_version: \"1.0\"\n---\n",
		filepath.Join(bundle, "features", "INDEX.md"): "# Features\n",
		filepath.Join(group, "INDEX.md"):              "# Payments\n",
	} {
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, root := range []string{filepath.Join(bundle, "features"), group, filepath.Join(group, "refunds")} {
		if got := BundleAbove(root); got != bundle {
			t.Errorf("BundleAbove(%s) = %q; want %s", root, got, bundle)
		}
	}
	for _, root := range []string{bundle, mk(t, tmp, "docs", "site")} {
		if got := BundleAbove(root); got != "" {
			t.Errorf("BundleAbove(%s) = %q; want none: no INDEX.md above it pins a version", root, got)
		}
	}
	root := filepath.Join(bundle, "features")
	want := root + " is inside the bundle at " + bundle + ", not a bundle of its own — pass --root " + bundle + ", or leave --root out"
	if err := InsideBundle(root, bundle); err == nil || err.Error() != want {
		t.Errorf("InsideBundle: %v\nwant: %s", err, want)
	}
}
