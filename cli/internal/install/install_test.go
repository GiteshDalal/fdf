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

func TestInstallUnknownHarness(t *testing.T) {
	var out bytes.Buffer
	if code := Run("emacs", t.TempDir(), &out); code != 2 {
		t.Fatalf("unknown harness should be usage error, got %d", code)
	}
}
