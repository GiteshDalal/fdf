package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fdf tag takes one mode at a time, its arguments as each mode needs them,
// and --dry-run and --definition only where they apply.
func TestTagUsage(t *testing.T) {
	initBundle(t)
	for _, tc := range []struct {
		args []string
		says string
	}{
		{[]string{"tag", "--sync", "--merge", "a", "b"}, "usage: fdf tag takes one of --show, --sync and --merge"},
		{[]string{"tag", "--dry-run"}, "usage: --dry-run applies to `fdf tag --sync` and `fdf tag --merge`"},
		{[]string{"tag", "--merge", "a"}, "usage: fdf tag --merge [--dry-run] <from> <into>"},
		{[]string{"tag", "--sync", "a"}, "usage: fdf tag"},
		{[]string{"tag", "--definition", "x", "a", "b"}, "usage: --definition gives one tag its definition"},
	} {
		if code, out, _ := fdfRun(tc.args...); code != 2 || !strings.Contains(out, tc.says) {
			t.Errorf("%v: exit %d\n%s", tc.args, code, out)
		}
	}
}

// A command that creates a document gives it --tags, and adds a tag the
// dictionary lacks with a TODO definition; a Fix started from a bug carries
// the bug's tags.
func TestCreatingCommandsTakeTags(t *testing.T) {
	root := initBundle(t)
	if code, out, _ := fdfRun("bug", "--cost", "7", "--tags", "refunds,payments", "split-capture"); code != 0 || !strings.Contains(out, "tagged it refunds, payments") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "bugs", "split-capture.md"))
	if !strings.Contains(string(raw), "\ncost: 7\n") || !strings.Contains(string(raw), "\ntags: [refunds, payments]\ntimestamp: ") {
		t.Errorf("bugs/split-capture.md:\n%s", raw)
	}
	tags, _ := os.ReadFile(filepath.Join(root, "TAGS.md"))
	if !strings.Contains(string(tags), "\n## refunds\nTODO — ") || !strings.Contains(string(tags), "\n## payments\nTODO — ") {
		t.Errorf("TAGS.md:\n%s", tags)
	}
	if code, out, _ := fdfRun("new", "--tags", "Not A Tag", "payouts"); code != 1 || !strings.Contains(out, `error: --tags: "Not A Tag" is not a tag`) {
		t.Errorf("a tag that is no tag is refused before anything is written: exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "features", "payouts.md")); err == nil {
		t.Errorf("nothing is written when --tags is refused")
	}
	if code, out, _ := fdfRun("debt", "--tag", "x", "slug"); code != 2 || !strings.Contains(out, "drop the listing flags") {
		t.Errorf("a listing flag with a slug: exit %d\n%s", code, out)
	}
	if code, out, _ := fdfRun("debt", "--sort", "priority"); code != 2 || !strings.Contains(out, "usage: --sort takes cost or value") {
		t.Errorf("--sort priority: exit %d\n%s", code, out)
	}
}
