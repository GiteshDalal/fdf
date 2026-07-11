package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallClaudeCodePlacesSkillsPrimerAndUpgrades(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("claude-code", home, "", &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	for _, skill := range []string{"fdf-help", "fdf-init", "fdf-brainstorm", "fdf-plan", "fdf-execute"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", skill, "SKILL.md")); err != nil {
			t.Fatalf("missing skill %s", skill)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "commands", "fdf-validate.md")); err != nil {
		t.Fatal("missing slash command")
	}
	claudeMd, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil || !strings.Contains(string(claudeMd), "## Feature Document Format") {
		t.Fatalf("CLAUDE.md primer missing: %v\n%s", err, claudeMd)
	}
	if !strings.Contains(string(claudeMd), "docs/features/SPEC.md") {
		t.Fatalf("primer should point at the bundle spec copy:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "SURFACES.md") {
		t.Fatalf("primer should mention SURFACES.md Context doc:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "slug.spec.md") {
		t.Fatalf("primer should describe stem-qualified trail siblings:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "fill the four") {
		t.Fatalf("primer should say fill the four Context docs:\n%s", claudeMd)
	}
	out.Reset()
	if code := Run("claude-code", home, "", &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-install should be up to date: %d %q", code, out.String())
	}
	// Simulate an older install.
	marker := filepath.Join(home, ".claude", "skills", "fdf-brainstorm", ".fdf-version")
	os.WriteFile(marker, []byte("0.1.0 root=docs/features"), 0o644)
	out.Reset()
	if code := Run("claude-code", home, "", &out); code != 0 || !strings.Contains(out.String(), "upgraded") {
		t.Fatalf("should auto-upgrade: %d %q", code, out.String())
	}
}

func TestInstallCodexAndOpencodePlaceSkills(t *testing.T) {
	for harnessName, dir := range map[string][]string{
		"codex":    {".codex", "skills"},
		"opencode": {".config", "opencode", "skills"},
	} {
		home := t.TempDir()
		var out bytes.Buffer
		if code := Run(harnessName, home, "", &out); code != 0 {
			t.Fatalf("%s install: %d\n%s", harnessName, code, out.String())
		}
		p := filepath.Join(append(append([]string{home}, dir...), "fdf-help", "SKILL.md")...)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s: missing %s", harnessName, p)
		}
		agents := filepath.Join(filepath.Dir(filepath.Join(append([]string{home}, dir...)...)), "AGENTS.md")
		raw, err := os.ReadFile(agents)
		if err != nil || !strings.Contains(string(raw), "## Feature Document Format") {
			t.Fatalf("%s: AGENTS.md primer missing at %s: %v", harnessName, agents, err)
		}
	}
}

func TestInstallCustomRootRewritesSkillsAndPrimer(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("codex", home, "wiki/fdf", &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	skill, _ := os.ReadFile(filepath.Join(home, ".codex", "skills", "fdf-help", "SKILL.md"))
	if strings.Contains(string(skill), "docs/features") {
		t.Fatalf("default root leaked into rewritten skill:\n%s", skill)
	}
	if !strings.Contains(string(skill), "wiki/fdf") {
		t.Fatalf("custom root missing from skill:\n%s", skill)
	}
	agents, _ := os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if !strings.Contains(string(agents), "wiki/fdf/SPEC.md") {
		t.Fatalf("primer should reference the custom root spec path:\n%s", agents)
	}
	// Re-installing with a different root is an upgrade, not "up to date".
	out.Reset()
	if code := Run("codex", home, "", &out); code != 0 || strings.Contains(out.String(), "up to date") {
		t.Fatalf("root change should reinstall: %d %q", code, out.String())
	}
	skill, _ = os.ReadFile(filepath.Join(home, ".codex", "skills", "fdf-help", "SKILL.md"))
	if !strings.Contains(string(skill), "docs/features") {
		t.Fatalf("reinstall with default root should restore default path:\n%s", skill)
	}
}

func TestPrimerSkippedWhenHeadingPresent(t *testing.T) {
	home := t.TempDir()
	agents := filepath.Join(home, ".codex", "AGENTS.md")
	os.MkdirAll(filepath.Dir(agents), 0o755)
	custom := "# Mine\n\n## Feature Document Format\n\nMy own hand-written FDF notes.\n"
	os.WriteFile(agents, []byte(custom), 0o644)
	var out bytes.Buffer
	if code := Run("codex", home, "", &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(agents)
	if string(got) != custom {
		t.Fatalf("existing primer heading must be left untouched:\n%s", got)
	}
}

func TestUpgradeRemovesLegacyManagedBlock(t *testing.T) {
	home := t.TempDir()
	agents := filepath.Join(home, ".codex", "AGENTS.md")
	os.MkdirAll(filepath.Dir(agents), 0o755)
	legacy := "# Keep me\n\n<!-- fdf:begin v0.2.2 (managed by `fdf install` — do not edit) -->\nold inlined skills\n<!-- fdf:end -->\n\n# Keep me too\n"
	os.WriteFile(agents, []byte(legacy), 0o644)
	var out bytes.Buffer
	if code := Run("codex", home, "", &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(agents)
	s := string(got)
	if strings.Contains(s, "fdf:begin") {
		t.Fatalf("legacy block not removed:\n%s", s)
	}
	if !strings.Contains(s, "# Keep me") || !strings.Contains(s, "# Keep me too") {
		t.Fatalf("surrounding content lost:\n%s", s)
	}
	if !strings.Contains(s, "## Feature Document Format") {
		t.Fatalf("primer not added after legacy cleanup:\n%s", s)
	}
}

func TestInstallUnknownHarness(t *testing.T) {
	var out bytes.Buffer
	if code := Run("emacs", t.TempDir(), "", &out); code != 2 {
		t.Fatalf("unknown harness should be usage error, got %d", code)
	}
}
