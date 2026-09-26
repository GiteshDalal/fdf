package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

// fdfRun runs one invocation the way main does and returns its exit code,
// stdout and stderr.
func fdfRun(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// initBundle scaffolds a bundle in a temp dir and points FDF_ROOT_DIR at it.
func initBundle(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "features")
	var out bytes.Buffer
	if code := scaffold.Init(root, &out); code != 0 {
		t.Fatalf("init: %d\n%s", code, out.String())
	}
	t.Setenv("FDF_ROOT_DIR", root)
	return root
}

// Go's flag package stops at the first argument, so a flag written after one
// used to be read as an argument: a wrong error, a hint that dropped the
// flag, or none. Every such command is now refused with the command as it
// should have been typed.
func TestFlagAfterAnArgumentIsRefusedWithTheCorrectedCommand(t *testing.T) {
	for _, tc := range []struct {
		args []string
		flag string
		want string
	}{
		{[]string{"debt", "venues/x", "--resource", "a,b"}, "--resource", "fdf debt --resource a,b venues/x"},
		{[]string{"bug", "x", "--affects", "payments/y"}, "--affects", "fdf bug --affects payments/y x"},
		{[]string{"new", "payments/x", "--root", "docs"}, "--root", "fdf new --root docs payments/x"},
		{[]string{"practice", "x", "--root=docs"}, "--root=docs", "fdf practice --root=docs x"},
		{[]string{"mv", "payments/x", "--dry-run"}, "--dry-run", "fdf mv --dry-run payments/x"},
		{[]string{"log", "entry text", "--root", "docs"}, "--root", "fdf log --root docs 'entry text'"},
		{[]string{"install", "codex", "--project"}, "--project", "fdf install --project codex"},
		{[]string{"change", "x", "--affects", "p/q"}, "--affects", "fdf change --affects p/q x"},
		// A command that takes no arguments gives the stray one to the flag
		// it meant, and keeps every flag that was typed.
		{[]string{"lexicon", "Venue", "--fix"}, "--fix", "fdf lexicon --fix --term Venue"},
		{[]string{"validate", "docs/features", "--strict-domain"}, "--strict-domain", "fdf validate --strict-domain --root docs/features"},
	} {
		code, out, _ := fdfRun(tc.args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2\n%s", tc.args, code, out)
		}
		want := "error: " + tc.flag + " comes after an argument, so it was not read as a flag. Flags go first:\n  " + tc.want + "\n"
		if out != want {
			t.Errorf("%v:\n got: %q\nwant: %q", tc.args, out, want)
		}
	}
}

// Only a token naming one of the command's flags counts, and "--" ends the
// flags: a log entry may start with a dash.
func TestDashedArgumentsThatAreNotFlagsPass(t *testing.T) {
	root := initBundle(t)
	for _, args := range [][]string{
		{"log", "LOG", "- a bullet of its own"},
		{"log", "--", "- a bullet of its own"},
		{"log", "LOG", "--", "--root is not a flag here"},
		{"log", "--", "--root is not a flag here either"},
	} {
		code, out, _ := fdfRun(args...)
		if code != 0 || strings.Contains(out, "comes after an argument") {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(root, "LOG.md"))
	for _, want := range []string{"- a bullet of its own\n", "* --root is not a flag here\n", "* --root is not a flag here either\n"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("LOG.md should hold %q:\n%s", want, raw)
		}
	}
	// Before any argument, a dashed one is read as a flag; the error says to
	// put -- before it, and shows how.
	code, out, _ := fdfRun("log", "- a bullet of its own")
	if want := "error: '- a bullet of its own' starts with a dash, so it was read as a flag. Put -- before it:\n  fdf log -- '- a bullet of its own'\n"; code != 2 || out != want {
		t.Errorf("exit %d\n got: %q\nwant: %q", code, out, want)
	}
}

// -h and --help print the command's own help topic, exit 0, wherever they
// stand before "--" — never Go's raw flag list ("Usage of debt:").
func TestHelpFlagsPrintTheCommandsTopic(t *testing.T) {
	for _, args := range [][]string{
		{"debt", "--help"},
		{"debt", "-h"},
		{"debt", "-help"},
		{"debt", "venues/x", "--help"},
		{"help", "debt"},
	} {
		code, out, _ := fdfRun(args...)
		if code != 0 {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
		if !strings.Contains(out, "  fdf debt [--root <dir>]") || !strings.Contains(out, "$ fdf debt --open") || strings.Contains(out, "Usage of") {
			t.Errorf("%v: want the debt topic:\n%s", args, out)
		}
	}
	for _, name := range []string{"change", "validate", "bug"} {
		_, out, _ := fdfRun(name, "-h")
		// Back-quoted words in Go's flag text were read as value names.
		for _, bad := range []string{"-from resolves", "-strict-domain strict", "-no-log"} {
			if strings.Contains(out, bad) {
				t.Errorf("fdf %s -h shows Go's flag list (%q):\n%s", name, bad, out)
			}
		}
	}
}

func TestTopLevelHelpAndVersionFlags(t *testing.T) {
	for _, flag := range []string{"-h", "-help", "--help"} {
		code, out, errOut := fdfRun(flag)
		if code != 0 || !strings.Contains(out, "Usage: fdf <command> [flags] [arguments]") || errOut != "" {
			t.Errorf("fdf %s: exit %d\n%s%s", flag, code, out, errOut)
		}
	}
	for _, args := range [][]string{{"--version"}, {"-version"}, {"version"}} {
		code, out, _ := fdfRun(args...)
		if code != 0 || out != "fdf "+version+"\n" {
			t.Errorf("fdf %v: exit %d, %q", args, code, out)
		}
	}
	if code, _, errOut := fdfRun(); code != 2 || !strings.Contains(errOut, "Usage: fdf") {
		t.Errorf("fdf alone: exit %d\n%s", code, errOut)
	}
}

// A parse error names the problem, then the command's usage line from its
// topic and where the rest is.
func TestFlagErrorsPrintTheTopicUsage(t *testing.T) {
	code, out, _ := fdfRun("debt", "--bogus")
	want := "error: flag provided but not defined: -bogus\n" + wrapUsage("usage: ", mustTopic(t, "debt").usage) + "\nRun 'fdf help debt' for its flags and examples.\n"
	if code != 2 || out != want {
		t.Errorf("exit %d\n got: %q\nwant: %q", code, out, want)
	}
}

// Every generic usage error prints the command's one usage line: the one in
// its help topic, with every flag it has.
func TestUsageErrorsQuoteTheTopic(t *testing.T) {
	for _, args := range [][]string{
		{"new"}, {"practice"}, {"debt", "a", "b"}, {"bug", "a", "b"}, {"adopt", "a/b", "c/d"},
		{"change"}, {"fix"}, {"history"}, {"log"}, {"mv", "a"}, {"release"}, {"install"},
		{"help", "a", "b"}, {"version", "x"},
	} {
		code, out, _ := fdfRun(args...)
		if want := wrapUsage("usage: ", mustTopic(t, args[0]).usage); code != 2 || !strings.Contains(out, want) {
			t.Errorf("%v: exit %d, want %q in:\n%s", args, code, out, want)
		}
	}
}

// A missing required flag is a usage error, exit 2 (EXIT CODES).
func TestMissingRequiredFlagsExit2(t *testing.T) {
	initBundle(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"adopt", "payments/x"}, "--resource is required"},
		{[]string{"change", "refund-window"}, "--affects is required"},
		{[]string{"fix", "refund-rounding"}, "--affects is required"},
		{[]string{"lexicon", "--fix"}, "--fix sweeps one term at a time"},
	} {
		code, out, _ := fdfRun(tc.args...)
		if code != 2 || !strings.Contains(out, tc.want) {
			t.Errorf("%v: exit %d, want 2 and %q:\n%s", tc.args, code, tc.want, out)
		}
	}
}

// A register flag that only files an entry says which register it files in.
func TestFilingFlagsWithoutASlug(t *testing.T) {
	for _, tc := range []struct{ args []string }{
		{[]string{"debt", "--resource", "a"}},
		{[]string{"bug", "--affects", "p/q"}},
	} {
		code, out, _ := fdfRun(tc.args...)
		if code != 2 {
			t.Errorf("%v: exit %d", tc.args, code)
		}
		if tc.args[0] == "debt" && out != "usage: --resource applies when filing a debt: fdf debt [--resource <paths>] [<group>/…]<slug>\n" {
			t.Errorf("debt: %q", out)
		}
		if tc.args[0] == "bug" && out != "usage: --affects and --resource apply when filing a bug: fdf bug [--affects <ids>] [--resource <paths>] [<group>/…]<slug>\n" {
			t.Errorf("bug: %q", out)
		}
	}
	if code, out, _ := fdfRun("lexicon", "--term", "Venue", "--dry-run"); code != 2 || out != "usage: --dry-run goes with --fix: fdf lexicon --term Venue --fix --dry-run\n" {
		t.Errorf("lexicon --dry-run: exit %d %q", code, out)
	}
}

// Every command that works on a bundle says the same thing when there is none.
func TestCommandsSayTheSameWhenThereIsNoBundle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nowhere")
	t.Setenv("FDF_ROOT_DIR", root)
	want := "error: no bundle at " + root + " (no INDEX.md) — run `fdf init` first, or point --root at the bundle\n"
	for _, args := range [][]string{
		{"validate"}, {"migrate"}, {"debt"}, {"bug", "--cleanup"}, {"adopt"}, {"new", "payments/x"},
		{"practice", "x"}, {"history", "payments/x"}, {"change", "--affects", "p/q", "x"},
		{"mv", "a/b", "c/d"}, {"lexicon"}, {"release", "1.0.0"}, {"log", "an entry"},
	} {
		code, out, _ := fdfRun(args...)
		if code != 1 || !strings.Contains(out, want) {
			t.Errorf("%v: exit %d, want 1 and %q:\n%s", args, code, want, out)
		}
	}
	if _, err := os.Stat(root); err == nil {
		t.Errorf("no command may create %s", root)
	}
}

// Every command that works on a bundle's documents points a 0.x bundle at
// `fdf migrate`, in the same words, and writes nothing in it.
func TestCommandsPointA0xBundleAtMigrate(t *testing.T) {
	root := t.TempDir()
	index := "---\nfdf_version: \"0.7\"\n---\n\n# Bundle\n"
	if err := os.WriteFile(filepath.Join(root, "INDEX.md"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FDF_ROOT_DIR", root)
	want := "error: this bundle pins fdf_version 0.7; fdf's commands work on spec 1.0 bundles — upgrading the bundle is the user's decision, since `fdf migrate` moves its documents and rewrites references to them across the project: `fdf migrate --dry-run` shows the plan\n"
	for _, args := range [][]string{
		{"new", "payments/x"}, {"adopt", "--resource", "main.go", "payments/x"}, {"adopt"}, {"practice", "x"},
		{"debt"}, {"debt", "x"}, {"bug", "--cleanup"}, {"change", "--affects", "features/p/q", "x"},
		{"fix", "--affects", "features/p/q", "x"}, {"history", "features/p/q"},
		{"mv", "features/a", "features/b"}, {"lexicon"}, {"log", "an entry"}, {"release", "1.0.0"},
	} {
		code, out, _ := fdfRun(args...)
		if code != 1 || !strings.Contains(out, want) {
			t.Errorf("%v: exit %d, want 1 and %q:\n%s", args, code, want, out)
		}
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("no command may write in a 0.x bundle; it holds %d entries", len(entries))
	}
}

// A register's INDEX.md pins nothing, so a root pointed at one used to be
// sent to `fdf migrate`, which then built a second bundle inside the first.
// Every command, init and migrate among them, names the bundle instead, in
// the same words, and writes nothing.
func TestCommandsSendARootInsideABundleToIt(t *testing.T) {
	bundle := initBundle(t)
	root := filepath.Join(bundle, "features")
	t.Setenv("FDF_ROOT_DIR", root)
	files := func() string {
		var b strings.Builder
		filepath.WalkDir(bundle, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				raw, _ := os.ReadFile(p)
				b.WriteString("== " + p + "\n" + string(raw))
			}
			return nil
		})
		return b.String()
	}
	before := files()
	want := "error: " + root + " is inside the bundle at " + bundle + ", not a bundle of its own — pass --root " + bundle + ", or leave --root out\n"
	for _, args := range [][]string{
		{"init"}, {"migrate"},
		{"new", "payments/x"}, {"adopt", "--resource", "main.go", "payments/x"}, {"adopt"}, {"practice", "x"},
		{"debt"}, {"debt", "x"}, {"bug", "--cleanup"}, {"change", "--affects", "features/p/q", "x"},
		{"fix", "--affects", "features/p/q", "x"}, {"history", "features/p/q"},
		{"mv", "features/a", "features/b"}, {"lexicon"}, {"log", "an entry"}, {"release", "1.0.0"},
	} {
		code, out, _ := fdfRun(args...)
		if code != 1 || !strings.Contains(out, want) {
			t.Errorf("%v: exit %d, want 1 and %q:\n%s", args, code, want, out)
		}
	}
	if after := files(); after != before {
		t.Errorf("no command may write inside the bundle:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
