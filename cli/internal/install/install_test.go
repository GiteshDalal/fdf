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
	if code := Run("claude-code", home, "", false, &out); code != 0 {
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
	if code := Run("claude-code", home, "", false, &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-install should be up to date: %d %q", code, out.String())
	}
	// Simulate an older install.
	marker := filepath.Join(home, ".claude", "skills", "fdf-brainstorm", ".fdf-version")
	os.WriteFile(marker, []byte("0.1.0 root=docs/features"), 0o644)
	out.Reset()
	if code := Run("claude-code", home, "", false, &out); code != 0 || !strings.Contains(out.String(), "upgraded") {
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
		if code := Run(harnessName, home, "", false, &out); code != 0 {
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
	if code := Run("codex", home, "wiki/fdf", false, &out); code != 0 {
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
	if code := Run("codex", home, "", false, &out); code != 0 || strings.Contains(out.String(), "up to date") {
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
	if code := Run("codex", home, "", false, &out); code != 0 {
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
	if code := Run("codex", home, "", false, &out); code != 0 {
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
	if code := Run("emacs", t.TempDir(), "", false, &out); code != 2 {
		t.Fatalf("unknown harness should be usage error, got %d", code)
	}
}

func TestInstallProjectClaudeCode(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(proj, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed a user-level install so we can prove project scope does not touch it.
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("seed user install: %d\n%s", code, out.String())
	}
	userSkill := filepath.Join(home, ".claude", "skills", "fdf-help", "SKILL.md")
	userBefore, err := os.ReadFile(userSkill)
	if err != nil {
		t.Fatal(err)
	}
	// Unique marker so any rewrite is visible.
	if err := os.WriteFile(userSkill, append(userBefore, []byte("\n// user-canary\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	if code := Run("claude-code", proj, "", true, &out); code != 0 {
		t.Fatalf("project install: %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "fdf-help", "SKILL.md")); err != nil {
		t.Fatalf("project skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "commands", "fdf-validate.md")); err != nil {
		t.Fatal("project slash command missing")
	}
	claudeMd, err := os.ReadFile(filepath.Join(proj, "CLAUDE.md"))
	if err != nil || !strings.Contains(string(claudeMd), "## Feature Document Format") {
		t.Fatalf("project CLAUDE.md primer missing (must be repo-root, not .claude/CLAUDE.md): %v\n%s", err, claudeMd)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatalf("project scope must not write .claude/CLAUDE.md: %v", err)
	}

	userAfter, err := os.ReadFile(userSkill)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(userAfter), "user-canary") {
		t.Fatalf("user-level install was modified by project install:\n%s", userAfter)
	}
}

func TestInstallProjectCodexAndOpencode(t *testing.T) {
	for harnessName, skillsSeg := range map[string][]string{
		"codex":    {".codex", "skills"},
		"opencode": {".opencode", "skills"},
	} {
		proj := t.TempDir()
		var out bytes.Buffer
		if code := Run(harnessName, proj, "", true, &out); code != 0 {
			t.Fatalf("%s project install: %d\n%s", harnessName, code, out.String())
		}
		skill := filepath.Join(append(append([]string{proj}, skillsSeg...), "fdf-help", "SKILL.md")...)
		if _, err := os.Stat(skill); err != nil {
			t.Fatalf("%s: missing project skill %s: %v", harnessName, skill, err)
		}
		agents, err := os.ReadFile(filepath.Join(proj, "AGENTS.md"))
		if err != nil || !strings.Contains(string(agents), "## Feature Document Format") {
			t.Fatalf("%s: project AGENTS.md primer missing at repo root: %v\n%s", harnessName, err, agents)
		}
	}
}

func TestUpgradeRefreshesStaleShippedPrimer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact v0.3-era primer (untouched managed content).
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# My notes\n\n"+primerV03("docs/features")+"\n## Other section\n\nkeep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if strings.Contains(s, "paired directory beside it") {
		t.Fatalf("stale v0.3 primer text must be replaced:\n%s", s)
	}
	if !strings.Contains(s, "slug.spec.md") || !strings.Contains(s, "SURFACES.md") {
		t.Fatalf("refreshed primer must teach the v0.4 layout:\n%s", s)
	}
	if !strings.Contains(s, "# My notes") || !strings.Contains(s, "## Other section\n\nkeep me") {
		t.Fatalf("content around the managed section must survive:\n%s", s)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeLeavesUserEditedPrimerWithNote(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "## Feature Document Format\n\nMy own hand-tuned FDF notes.\n"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "My own hand-tuned FDF notes.") {
		t.Fatalf("user-edited primer must not be clobbered:\n%s", got)
	}
	if !strings.Contains(out.String(), "differs from the shipped primer") {
		t.Fatalf("should warn about the outdated user-edited primer:\n%s", out.String())
	}
}
