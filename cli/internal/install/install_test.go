package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallClaudeCodePlacesSkillsAndUpgrades(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("claude-code", home, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	for _, skill := range []string{"fdf-brainstorm", "fdf-plan", "fdf-execute"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", skill, "SKILL.md")); err != nil {
			t.Fatalf("missing skill %s", skill)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "commands", "fdf-validate.md")); err != nil {
		t.Fatal("missing slash command")
	}
	out.Reset()
	if code := Run("claude-code", home, &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-install should be up to date: %d %q", code, out.String())
	}
	// Simulate an older install.
	marker := filepath.Join(home, ".claude", "skills", "fdf-brainstorm", ".fdf-version")
	os.WriteFile(marker, []byte("0.1.0"), 0o644)
	out.Reset()
	if code := Run("claude-code", home, &out); code != 0 || !strings.Contains(out.String(), "upgraded") {
		t.Fatalf("should auto-upgrade: %d %q", code, out.String())
	}
}

func TestInstallCodexManagedBlockIdempotent(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("codex", home, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	p := filepath.Join(home, ".codex", "AGENTS.md")
	first, _ := os.ReadFile(p)
	if !strings.Contains(string(first), "<!-- fdf:begin v") {
		t.Fatalf("no managed block:\n%s", first)
	}
	Run("codex", home, &out)
	second, _ := os.ReadFile(p)
	if strings.Count(string(second), "<!-- fdf:begin") != 1 {
		t.Fatalf("managed block duplicated:\n%s", second)
	}
}

func TestInstallCodexPreservesSurroundingContentOrder(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, ".codex", "AGENTS.md")
	var out bytes.Buffer
	if code := Run("codex", home, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	// Wrap the freshly installed block in user content before and after it.
	installed, _ := os.ReadFile(p)
	before := "# My own preamble\n\nkeep me first\n\n"
	after := "\n# My own appendix\n\nkeep me last\n"
	if err := os.WriteFile(p, []byte(before+string(installed)+after), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := Run("codex", home, &out); code != 0 {
		t.Fatalf("re-install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(p)
	s := string(got)
	if strings.Count(s, "<!-- fdf:begin") != 1 {
		t.Fatalf("expected exactly one managed block:\n%s", s)
	}
	iBefore := strings.Index(s, "keep me first")
	iBlock := strings.Index(s, "<!-- fdf:begin")
	iAfter := strings.Index(s, "keep me last")
	if iBefore < 0 || iBlock < 0 || iAfter < 0 {
		t.Fatalf("content lost (before=%d block=%d after=%d):\n%s", iBefore, iBlock, iAfter, s)
	}
	if !(iBefore < iBlock && iBlock < iAfter) {
		t.Fatalf("order not preserved (before=%d block=%d after=%d):\n%s", iBefore, iBlock, iAfter, s)
	}
	if !strings.Contains(out.String(), "upgraded") {
		t.Fatalf("replacing an existing block should report upgraded: %q", out.String())
	}
}

func TestInstallUnknownHarness(t *testing.T) {
	var out bytes.Buffer
	if code := Run("emacs", t.TempDir(), &out); code != 2 {
		t.Fatalf("unknown harness should be usage error, got %d", code)
	}
}
