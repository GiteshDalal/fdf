package install

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	fdf "github.com/GiteshDalal/fdf"
)

func TestInstallClaudeCodePlacesSkillsPrimerAndUpgrades(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	for _, skill := range []string{"fdf-help", "fdf-init", "fdf-brainstorm", "fdf-plan", "fdf-execute", "fdf-change", "fdf-debug", "fdf-checkpoint", "fdf-validate"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", skill, "SKILL.md")); err != nil {
			t.Fatalf("missing skill %s", skill)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "commands")); !os.IsNotExist(err) {
		t.Fatalf("skills-only install must not create a commands dir: %v", err)
	}
	claudeMd, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil || !strings.Contains(string(claudeMd), "## Feature Document Format") {
		t.Fatalf("CLAUDE.md primer missing: %v\n%s", err, claudeMd)
	}
	if !strings.Contains(string(claudeMd), "docs/fdf/SPEC.md") {
		t.Fatalf("primer should point at the bundle spec copy:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "`features/payments/instant-refunds`") || !strings.Contains(string(claudeMd), "`fdf migrate`, then `fdf install`") {
		t.Fatalf("primer should teach 1.0's IDs and the upgrade from 0.x:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "SURFACES.md") {
		t.Fatalf("primer should mention SURFACES.md Context doc:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "slug.spec.md") {
		t.Fatalf("primer should describe stem-qualified trail siblings:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "fill the five") {
		t.Fatalf("primer should say fill the five Context docs:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "DOMAIN.md") {
		t.Fatalf("primer should mention the DOMAIN.md Context doc:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "practices/") || !strings.Contains(string(claudeMd), "type: Practice") {
		t.Fatalf("primer should teach practice documents:\n%s", claudeMd)
	}
	if !strings.Contains(string(claudeMd), "set its `timestamp` to\n  now, in UTC") || !strings.Contains(string(claudeMd), "leaves `timestamp` as it is") {
		t.Fatalf("primer should say a changed document's timestamp is now, in UTC, and a maintenance edit's is kept:\n%s", claudeMd)
	}
	if strings.Contains(string(claudeMd), "resolved in place") {
		t.Fatalf("a bug is never *repaired* in place; one that needs no repair is resolved:\n%s", claudeMd)
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

// Two builds of one version can ship different skills and primers — a
// development build and its release. The marker records what this build
// installs, so the later build upgrades the skills, and the earlier build's
// primer, untouched, is recognized as fdf's own and refreshed.
func TestInstallUpgradesAnEarlierBuildOfTheSameVersion(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	skills := filepath.Join(home, ".claude", "skills")
	marker := string(mustRead(t, filepath.Join(skills, "fdf-help", ".fdf-version")))
	if !strings.HasPrefix(marker, Version+" skills=") || !strings.HasSuffix(marker, " root="+defaultRoot) || recordedPrimer(marker) == "" {
		t.Fatalf("the marker keeps the version and records the build: %q", marker)
	}

	// An earlier build of this version: other skill text, another primer.
	variant := func(line string) string {
		return strings.Replace(primer(defaultRoot), primerHeading+"\n", primerHeading+"\n\n"+line+"\n", 1)
	}
	earlier := variant("An earlier build said this.")
	earlierMarker := versionMarker(defaultRoot, "0123456789ab", digest(strings.TrimRight(earlier, "\n")))
	for _, name := range skillNames {
		os.WriteFile(filepath.Join(skills, name, "SKILL.md"), []byte("an earlier build's skill\n"), 0o644)
		os.WriteFile(filepath.Join(skills, name, ".fdf-version"), []byte(earlierMarker), 0o644)
	}
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	os.WriteFile(claudeMd, []byte("# Mine\n\n"+earlier), 0o644)

	out.Reset()
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("reinstall: %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "upgraded fdf skills") || strings.Contains(out.String(), "up to date") {
		t.Errorf("an earlier build of the same version is upgraded:\n%s", out.String())
	}
	if s := string(mustRead(t, filepath.Join(skills, "fdf-help", "SKILL.md"))); strings.Contains(s, "an earlier build") {
		t.Errorf("the skills are this build's:\n%s", s)
	}
	if strings.Contains(out.String(), "differs from the shipped primer") {
		t.Errorf("the earlier build's primer is fdf's own, not an edit:\n%s", out.String())
	}
	if s := string(mustRead(t, claudeMd)); !strings.Contains(s, "# Mine\n\n"+strings.TrimRight(primer(defaultRoot), "\n")) {
		t.Errorf("the primer is this build's, the rest kept:\n%s", s)
	}

	// A primer edited after that build is still an edit.
	edited := variant("I wrote this line myself.")
	for _, name := range skillNames {
		os.WriteFile(filepath.Join(skills, name, ".fdf-version"), []byte(earlierMarker), 0o644)
	}
	os.WriteFile(claudeMd, []byte(edited), 0o644)
	out.Reset()
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("reinstall: %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "differs from the shipped primer") || string(mustRead(t, claudeMd)) != edited {
		t.Errorf("an edited primer is left as it is, with a note:\n%s", out.String())
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
	if strings.Contains(string(skill), defaultRoot) {
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
	if !strings.Contains(string(skill), defaultRoot) {
		t.Fatalf("reinstall with default root should restore default path:\n%s", skill)
	}
	// The primer the last install recorded is fdf's own under any root, so
	// it follows the root too, rather than reading as an edit.
	agents, _ = os.ReadFile(filepath.Join(home, ".codex", "AGENTS.md"))
	if !strings.Contains(string(agents), defaultRoot+"/SPEC.md") || strings.Contains(string(agents), "wiki/fdf") || strings.Contains(out.String(), "differs from the shipped primer") {
		t.Fatalf("the primer follows the root change:\n%s\n%s", agents, out.String())
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
	if _, err := os.Stat(filepath.Join(proj, ".claude", "commands")); !os.IsNotExist(err) {
		t.Fatalf("skills-only project install must not create a commands dir: %v", err)
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

func TestUpgradeRefreshesShippedV05Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.5.0 release wrote (untouched managed content).
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV05("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "fdf-debug skill") {
		t.Fatalf("refreshed primer must route broken things to fdf-debug:\n%s", got)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeRefreshesShippedV051Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.5.1 release wrote (untouched managed
	// content): four Context documents, no practices, no domain language.
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV051("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got := string(mustRead(t, path))
	if strings.Contains(got, "fill the four") {
		t.Fatalf("stale four-Context-doc text must be replaced:\n%s", got)
	}
	if !strings.Contains(got, "DOMAIN.md") || !strings.Contains(got, "type: Practice") {
		t.Fatalf("refreshed primer must teach DOMAIN.md and practices:\n%s", got)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeRefreshesShippedV06Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.6.0 release wrote (untouched managed
	// content): DOMAIN.md taught without its surface boundary.
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV06("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got := string(mustRead(t, path))
	if strings.Contains(got, "in the identifiers you write") {
		t.Fatalf("stale unscoped DOMAIN.md text must be replaced:\n%s", got)
	}
	if !strings.Contains(got, "does not govern what a person reads on a surface") {
		t.Fatalf("refreshed primer must scope the lexicon to internal language:\n%s", got)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeRefreshesShippedV061Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.6.1 release wrote (untouched managed
	// content): no fdf-checkpoint, no warning against hand edits.
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV061("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got := string(mustRead(t, path))
	if !strings.Contains(got, "fdf-checkpoint skill") {
		t.Fatalf("refreshed primer must point at the fdf-checkpoint skill:\n%s", got)
	}
	if !strings.Contains(got, "Never hand-edit this section") {
		t.Fatalf("refreshed primer must warn that hand edits freeze it:\n%s", got)
	}
	if strings.Count(got, primerHeading) != 1 {
		t.Fatalf("refresh must replace the section, not append a second one:\n%s", got)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeRefreshesShippedV062Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.6.2 release wrote (untouched managed
	// content): episodic documents never rewritten, no lexicon fix.
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV062("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got := string(mustRead(t, path))
	if !strings.Contains(got, "a lexicon fix (a banned word replaced by its term)") {
		t.Fatalf("refreshed primer must allow the lexicon fix on every document:\n%s", got)
	}
	if strings.Count(got, primerHeading) != 1 {
		t.Fatalf("refresh must replace the section, not append a second one:\n%s", got)
	}
	if !strings.Contains(out.String(), "updated") {
		t.Fatalf("report should say the primer was updated:\n%s", out.String())
	}
}

func TestUpgradeRefreshesShippedV063Primer(t *testing.T) {
	home := t.TempDir()
	// Seed the exact primer the v0.6.3 release wrote (untouched managed
	// content): no bug register, no adoption, the lexicon fix alone.
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(primerV063("docs/features")), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	got := string(mustRead(t, path))
	for _, want := range []string{"**Bug documents**", "fdf-adopt", "**maintenance edits**", "`fdf mv`", "fdf lexicon"} {
		if !strings.Contains(got, want) {
			t.Fatalf("refreshed primer must teach v0.7 (%q):\n%s", want, got)
		}
	}
	if strings.Count(got, primerHeading) != 1 {
		t.Fatalf("refresh must replace the section, not append a second one:\n%s", got)
	}
}

// The primer 0.7.0 shipped, untouched, is fdf's own: an upgrade replaces it
// with 1.0's, though it was written for docs/features, the root every
// install before 1.0 wrote when given none, and no marker says so.
func TestUpgradeRefreshesShippedV07Primer(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# Mine\n\n"+primerV07(legacyRoot)), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	if got := string(mustRead(t, path)); got != "# Mine\n\n"+primer(defaultRoot) || strings.Contains(out.String(), "differs from the shipped primer") {
		t.Fatalf("the 0.7 primer is replaced with 1.0's:\n%s\n%s", got, out.String())
	}
}

// A primer an earlier install wrote for another root, which its marker
// records, is fdf's own too: one written by 0.6.3, whose marker records a
// root and no primer, for docs/handbook, is replaced with the primer for
// the root this install bakes in.
func TestUpgradeRecognizesThePrimerOfTheRecordedRoot(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "CLAUDE.md")
	skills := filepath.Join(home, ".claude", "skills")
	for _, name := range skillNames {
		if err := os.MkdirAll(filepath.Join(skills, name), 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(skills, name, MarkerFile), []byte("0.6.3 root=docs/handbook"), 0o644)
	}
	os.WriteFile(path, []byte(primerV063("docs/handbook")), 0o644)
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	if got := string(mustRead(t, path)); got != primer(defaultRoot) || strings.Contains(out.String(), "differs from the shipped primer") {
		t.Fatalf("the primer written for the recorded root is replaced:\n%s\n%s", got, out.String())
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
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

// The pre-skills-only surface shipped three slash commands wrapping skills the
// model can now reach directly. An install must clear them so agents stop
// seeing stale wrappers, without touching commands the user wrote themselves.
func TestInstallRemovesSupersededSlashCommands(t *testing.T) {
	home := t.TempDir()
	cmds := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(cmds, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fdf-init.md", "fdf-new.md", "fdf-validate.md", "my-own.md"} {
		if err := os.WriteFile(filepath.Join(cmds, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	for _, name := range []string{"fdf-init.md", "fdf-new.md", "fdf-validate.md"} {
		if _, err := os.Stat(filepath.Join(cmds, name)); !os.IsNotExist(err) {
			t.Fatalf("superseded command %s still present: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(cmds, "my-own.md")); err != nil {
		t.Fatalf("user's own command must survive: %v", err)
	}
	if !strings.Contains(out.String(), "removed 3 superseded slash command(s)") {
		t.Fatalf("removal should be reported: %q", out.String())
	}

	// Second run: nothing left to remove, and the dir the user still owns stays.
	out.Reset()
	if code := Run("claude-code", home, "", false, &out); code != 0 || !strings.Contains(out.String(), "up to date") {
		t.Fatalf("re-install: %d %q", code, out.String())
	}
	if strings.Contains(out.String(), "superseded") {
		t.Fatalf("nothing left to remove, should not report: %q", out.String())
	}
}

// A commands dir that held only fdf's own commands is pruned, since leaving an
// empty directory behind is just litter.
func TestInstallPrunesCommandsDirItEmptied(t *testing.T) {
	home := t.TempDir()
	cmds := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(cmds, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmds, "fdf-new.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run("claude-code", home, "", false, &out); code != 0 {
		t.Fatalf("install: %d\n%s", code, out.String())
	}
	if _, err := os.Stat(cmds); !os.IsNotExist(err) {
		t.Fatalf("emptied commands dir should be pruned: %v", err)
	}
}

// The skills and the primer teach 1.0's names: a feature's path and its ID
// start with features/, and groups nest. These are 0.7's spellings, which
// teach the old layout wherever one comes back.
func TestSkillsAndPrimerTeachTheNamesOf10(t *testing.T) {
	stale := []struct{ re, why string }{
		{`\[<group>/\]<slug>`, "groups nest: [<group>/…]<slug>"},
		{`(?:fdf log|fdf history|--affects) <group>/`, "a command names a feature by its full ID, features/…"},
		{`(?:^|[^/])<group>/<slug>[./]`, "a feature's path starts with features/"},
	}
	for _, name := range append([]string{"primer"}, skillNames...) {
		text := primer(defaultRoot)
		if name != "primer" {
			raw, err := fs.ReadFile(fdf.Assets, "skills/"+name+"/SKILL.md")
			if err != nil {
				t.Fatal(err)
			}
			text = string(raw)
		}
		for _, s := range stale {
			if m := regexp.MustCompile(s.re).FindString(text); m != "" {
				t.Errorf("%s writes %q — %s", name, m, s.why)
			}
		}
	}
}
