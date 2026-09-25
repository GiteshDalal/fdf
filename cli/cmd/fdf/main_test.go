package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/install"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// writeMinimalBundle creates the valid-minimal bundle files under dir.
func writeMinimalBundle(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"INDEX.md": "---\nfdf_version: \"0.2\"\n---\n\n# B\n\n* [spec](https://github.com/GiteshDalal/fdf/blob/main/SPEC.md) - pin.\n",
		"LOG.md":   "# Bundle Update Log\n\n## 2026-07-06\n* **Initialization**: created.\n",
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallProjectOutsideGitExits2(t *testing.T) {
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	// Temp dirs have no .git ancestor; --project must refuse.
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if exit := runInstall([]string{"--project", "claude-code"}, &out); exit != 2 {
		t.Fatalf("expected exit 2 outside git project, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "git") {
		t.Fatalf("error should mention git project requirement:\n%s", out.String())
	}
}

func TestInstallProjectClaudeCodeCLI(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if exit := runInstall([]string{"--project", "claude-code"}, &out); exit != 0 {
		t.Fatalf("project install: exit %d\n%s", exit, out.String())
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude", "skills", "fdf-help", "SKILL.md")); err != nil {
		t.Fatalf("project skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "CLAUDE.md")); err != nil {
		t.Fatalf("repo-root CLAUDE.md missing: %v", err)
	}
}

func TestValidateHonorsEnvAndFlagRoots(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeMinimalBundle(t, filepath.Join(tmp, "documents", "features"))
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tmp)

	t.Setenv("FDF_ROOT_DIR", "documents/features")
	var out bytes.Buffer
	if exit := runValidate(nil, &out); exit != 0 {
		t.Fatalf("env root: exit %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "conformant") {
		t.Fatalf("env root output:\n%s", out.String())
	}

	t.Setenv("FDF_ROOT_DIR", "does/not/exist")
	out.Reset()
	if exit := runValidate([]string{"--root", "documents/features"}, &out); exit != 0 {
		t.Fatalf("--root must beat env: exit %d\n%s", exit, out.String())
	}
}

// With neither --root nor FDF_ROOT_DIR, a command finds a bundle from before
// 1.0 at docs/features and labels it so. Once docs/fdf holds a bundle too,
// docs/fdf wins, and the header warns about the other.
func TestDefaultRootIsLabelledAndAShadowedBundleWarned(t *testing.T) {
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tmp)
	t.Setenv("FDF_ROOT_DIR", "")
	features, fdf := filepath.Join(tmp, "docs", "features"), filepath.Join(tmp, "docs", "fdf")

	writeMinimalBundle(t, features)
	var out bytes.Buffer
	runValidate(nil, &out)
	if want := " · validate · root: " + features + " (pre-1.0 default docs/features)\n\n"; !strings.Contains(out.String(), want) {
		t.Errorf("a bundle at docs/features is found and labelled:\n%s", out.String())
	}

	writeMinimalBundle(t, fdf)
	out.Reset()
	runValidate(nil, &out)
	want := " · validate · root: " + fdf + " (default docs/fdf)\n" +
		"warning: " + features + " holds a bundle too; docs/fdf comes first, so pass --root to work on the other\n\n"
	if !strings.Contains(out.String(), want) {
		t.Errorf("docs/fdf wins, and the header warns about docs/features:\n%s", out.String())
	}
}

func TestSpecPrintsCurrentVersionByDefault(t *testing.T) {
	var out bytes.Buffer
	if exit := runSpec(nil, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	if !strings.HasPrefix(out.String(), "# Feature Document Format (FDF) — v"+scaffold.CurrentVersion()) {
		t.Fatalf("expected the current spec text, got:\n%.200s", out.String())
	}
}

func TestSpecVFlagPrintsOlderVersion(t *testing.T) {
	var out bytes.Buffer
	if exit := runSpec([]string{"-v=0.3"}, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	if !strings.HasPrefix(out.String(), "# Feature Document Format (FDF) — v0.3") {
		t.Fatalf("expected the v0.3 spec text, got:\n%.200s", out.String())
	}
}

func TestSpecUnknownVersionExits2(t *testing.T) {
	var out bytes.Buffer
	if exit := runSpec([]string{"-v", "9.9"}, &out); exit != 2 {
		t.Fatalf("expected exit 2 for an unembedded version, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "available:") {
		t.Fatalf("error should list the embedded versions:\n%s", out.String())
	}
}

func TestSpecListMarksCurrent(t *testing.T) {
	var out bytes.Buffer
	if exit := runSpec([]string{"--list"}, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), scaffold.CurrentVersion()+"  (current)") {
		t.Fatalf("--list should mark the current version:\n%s", out.String())
	}
}

// A bare path is the natural thing to type. Before these guards, commands
// discarded it and ran against the DEFAULT root, so `fdf migrate docs/features`
// reported an error about a path the user never typed — it looked like the
// command did nothing.
func TestCommandsRejectPositionalRootWithSuggestion(t *testing.T) {
	for _, tc := range []struct {
		cmd  string
		run  func([]string, *bytes.Buffer) int
		flag string
		arg  string
	}{
		{"migrate", func(a []string, o *bytes.Buffer) int { return runMigrate(a, o) }, "--root", "mydocs"},
		{"validate", func(a []string, o *bytes.Buffer) int { return runValidate(a, o) }, "--root", "mydocs"},
		{"init", func(a []string, o *bytes.Buffer) int { return runInit(a, o) }, "--root", "mydocs"},
		{"serve", func(a []string, o *bytes.Buffer) int { return runServe(a, o) }, "--root", "mydocs"},
		{"spec", func(a []string, o *bytes.Buffer) int { return runSpec(a, o) }, "-v", "0.3"},
	} {
		var out bytes.Buffer
		if exit := tc.run([]string{tc.arg}, &out); exit != 2 {
			t.Errorf("%s: expected exit 2 for a stray positional, got %d\n%s", tc.cmd, exit, out.String())
		}
		want := "fdf " + tc.cmd + " " + tc.flag + " " + tc.arg
		if !strings.Contains(out.String(), want) {
			t.Errorf("%s: error should suggest %q, got:\n%s", tc.cmd, want, out.String())
		}
	}
}

func TestHelpListsEveryCommand(t *testing.T) {
	var out bytes.Buffer
	if exit := runHelp(nil, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	for name := range commands {
		if !strings.Contains(out.String(), "fdf "+name) {
			t.Errorf("fdf help omits the %q command", name)
		}
	}
	for _, section := range []string{"BUNDLE ROOT", "TYPICAL FLOW", "EXIT CODES", "COMMANDS", "$ fdf validate"} {
		if !strings.Contains(out.String(), section) {
			t.Errorf("fdf help is missing the %q section", section)
		}
	}
}

// TestBannersNameCurrentSpecVersion locks both product headers to the spec
// version the binary actually scaffolds — `fdf help` once shipped a stale
// v0.4 banner while the short usage already said v0.5.
func TestBannersNameCurrentSpecVersion(t *testing.T) {
	want := "(SPEC v" + scaffold.CurrentVersion() + ")"

	var out bytes.Buffer
	if exit := runHelp(nil, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), want) {
		t.Errorf("fdf help banner does not name %s:\n%s", want, firstLine(out.String()))
	}
	if !strings.Contains(overview(), want) {
		t.Errorf("overview banner does not name %s:\n%s", want, firstLine(overview()))
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TestDevVersionDefaultsAgree keeps install's fallback in step with the CLI
// version goreleaser stamps. install.Version is overwritten at dispatch, so a
// drifted default is invisible until the package is driven directly — which is
// how both defaults sat at 0.4.0-dev for a whole release.
func TestDevVersionDefaultsAgree(t *testing.T) {
	if install.Version != version {
		t.Errorf("install.Version default %q does not match main.version %q", install.Version, version)
	}
}

func TestHelpForOneCommandIsScoped(t *testing.T) {
	var out bytes.Buffer
	if exit := runHelp([]string{"migrate"}, &out); exit != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "fdf migrate [--root <dir>]") {
		t.Fatalf("expected the migrate entry:\n%s", out.String())
	}
	// migrate's body legitimately mentions `fdf install`; assert no OTHER
	// command's usage line is rendered.
	if strings.Contains(out.String(), "fdf install [--project]") {
		t.Fatalf("fdf help migrate should print only that entry:\n%s", out.String())
	}
}

func TestHelpUnknownCommandExits2(t *testing.T) {
	var out bytes.Buffer
	if exit := runHelp([]string{"nope"}, &out); exit != 2 {
		t.Fatalf("expected exit 2, got %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "commands:") {
		t.Fatalf("should list the known commands:\n%s", out.String())
	}
}

// Every bundle command states which binary ran and which root it chose, and
// why. Those are the two facts that explain a command that appears to do
// nothing (a stale shim-pinned fdf, or a root resolved somewhere else).
func TestBundleCommandsAnnounceVersionAndRoot(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("FDF_ROOT_DIR", tmp)
	for _, tc := range []struct {
		cmd  string
		run  func([]string, *bytes.Buffer) int
		args []string
	}{
		{"validate", func(a []string, o *bytes.Buffer) int { return runValidate(a, o) }, nil},
		{"init", func(a []string, o *bytes.Buffer) int { return runInit(a, o) }, nil},
		{"new", func(a []string, o *bytes.Buffer) int { return runNew(a, o) }, []string{"g/s"}},
		{"migrate", func(a []string, o *bytes.Buffer) int { return runMigrate(a, o) }, nil},
	} {
		var out bytes.Buffer
		tc.run(tc.args, &out)
		first := strings.SplitN(out.String(), "\n", 2)[0]
		for _, want := range []string{"fdf " + version, tc.cmd, tmp, "FDF_ROOT_DIR"} {
			if !strings.Contains(first, want) {
				t.Errorf("%s: banner %q missing %q", tc.cmd, first, want)
			}
		}
	}
}

// A docs repository cloned on its own is a bundle at the top of its own
// repository: there is no project around it to check `resource` paths
// against, so R1 is skipped with a warning rather than failing every path —
// and the repository's own hidden directories are not bundle directories.
func TestValidateBundleThatIsItsOwnRepository(t *testing.T) {
	src := filepath.Join("..", "..", "..", "testdata", "valid-adopted-v07", "repo", "docs", "features")
	root := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{".git", ".obsidian"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, ".obsidian", "Workspace.md"), []byte("no frontmatter\n"), 0o644)

	var out bytes.Buffer
	if exit := runValidate([]string{"--root", root}, &out); exit != 0 {
		t.Fatalf("a bundle that is its own repository must validate: exit %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "R1 skipped") {
		t.Errorf("R1 should be skipped, with a warning:\n%s", out.String())
	}
	if strings.Contains(out.String(), ".obsidian") || strings.Contains(out.String(), ".git") {
		t.Errorf("hidden directories are not bundle directories:\n%s", out.String())
	}
}
