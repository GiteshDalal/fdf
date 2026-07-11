// Package install places the FDF skills into an AI harness's configuration.
// Claude Code, Codex, and opencode all support agent skills as directories of
// SKILL.md files, so every harness gets real skills — loaded on demand, not
// inlined into instruction files. The instruction file (CLAUDE.md/AGENTS.md)
// only gets a short "## Feature Document Format" primer, and only when that
// heading is absent, so user edits to it are never clobbered.
//
// Installs are idempotent: an existing install at the current version and
// bundle root is reported "up to date"; anything else is upgraded in place.
// User-level and project-level installs coexist: each destination carries its
// own .fdf-version markers and upgrades independently.
package install

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	fdf "github.com/GiteshDalal/fdf"
)

// Version is stamped by the CLI (main.version) at dispatch time.
var Version = "0.4.0-dev"

// defaultRoot is the bundle root the skill texts are written against; a
// different install root rewrites every occurrence in the skill bodies.
const defaultRoot = "docs/features"

var skillNames = []string{"fdf-help", "fdf-init", "fdf-brainstorm", "fdf-plan", "fdf-execute"}

// legacyBlockRe matches the pre-0.3 managed block that inlined full skills
// into AGENTS.md; upgrades remove it in favor of real skills + the primer.
var legacyBlockRe = regexp.MustCompile(`(?s)<!-- fdf:begin v[^>]*-->.*?<!-- fdf:end -->\n?`)

const primerHeading = "## Feature Document Format"

// harness describes per-scope destination path segments under a base directory
// (user home for user-level installs, project root for --project).
type harness struct {
	skillsDir        []string // user-level skills path under home
	instrFile        []string // user-level instruction file under home
	projectSkillsDir []string // project-level skills path under project root
	projectInstrFile []string // project-level instruction file under project root
	commands         bool     // claude-code slash commands
}

var harnesses = map[string]harness{
	"claude-code": {
		skillsDir:        []string{".claude", "skills"},
		instrFile:        []string{".claude", "CLAUDE.md"},
		projectSkillsDir: []string{".claude", "skills"},
		projectInstrFile: []string{"CLAUDE.md"}, // repo-root memory file, not .claude/CLAUDE.md
		commands:         true,
	},
	"codex": {
		skillsDir:        []string{".codex", "skills"},
		instrFile:        []string{".codex", "AGENTS.md"},
		projectSkillsDir: []string{".codex", "skills"},
		projectInstrFile: []string{"AGENTS.md"},
	},
	"opencode": {
		skillsDir:        []string{".config", "opencode", "skills"},
		instrFile:        []string{".config", "opencode", "AGENTS.md"},
		projectSkillsDir: []string{".opencode", "skills"},
		projectInstrFile: []string{"AGENTS.md"},
	},
}

// Run installs the skills and instruction-file primer for a harness. base is
// the install root: the user home for user-level installs ("" → UserHomeDir),
// or the project root when project is true. root is the bundle root to bake
// into the installed skills ("" means the default docs/features); pass it as
// the user wrote it (project-relative preferred).
func Run(harnessName, base, root string, project bool, out io.Writer) int {
	if base == "" {
		if project {
			fmt.Fprintln(out, "error: project install requires a project root base")
			return 1
		}
		base, _ = os.UserHomeDir()
	}
	if root == "" {
		root = defaultRoot
	}
	h, ok := harnesses[harnessName]
	if !ok {
		fmt.Fprintf(out, "unknown harness %q\n\nusage: fdf install [--project] [--root <dir>] <claude-code|codex|opencode>\n", harnessName)
		return 2
	}

	skillsSeg := h.skillsDir
	instrSeg := h.instrFile
	if project {
		skillsSeg = h.projectSkillsDir
		instrSeg = h.projectInstrFile
	}

	skillsDir := filepath.Join(append([]string{base}, skillsSeg...)...)
	marker := Version + " root=" + root

	upToDate := true
	for _, name := range skillNames {
		if v, err := os.ReadFile(filepath.Join(skillsDir, name, ".fdf-version")); err != nil || string(v) != marker {
			upToDate = false
		}
	}

	hadAny := false
	if !upToDate {
		for _, name := range skillNames {
			if _, err := os.Stat(filepath.Join(skillsDir, name)); err == nil {
				hadAny = true
			}
			raw, err := fs.ReadFile(fdf.Assets, "skills/"+name+"/SKILL.md")
			if err != nil {
				fmt.Fprintf(out, "error: embedded skills/%s/SKILL.md: %v\n", name, err)
				return 1
			}
			body := string(raw)
			if root != defaultRoot {
				body = strings.ReplaceAll(body, defaultRoot, root)
			}
			dir := filepath.Join(skillsDir, name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			// The marker is written only after SKILL.md landed, so a failed
			// install can never masquerade as "up to date" on the next run.
			if err := os.WriteFile(filepath.Join(dir, ".fdf-version"), []byte(marker), 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
		}
	}

	cmdCount := 0
	if h.commands && !upToDate {
		cmds, err := fs.ReadDir(fdf.Assets, "harness/claude-code/commands")
		if err != nil {
			fmt.Fprintf(out, "error: embedded harness/claude-code/commands: %v\n", err)
			return 1
		}
		// Commands sit beside skills under .claude/ for both user and project scopes.
		cmdDir := filepath.Join(filepath.Dir(skillsDir), "commands")
		if err := os.MkdirAll(cmdDir, 0o755); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		for _, e := range cmds {
			raw, err := fs.ReadFile(fdf.Assets, "harness/claude-code/commands/"+e.Name())
			if err != nil {
				fmt.Fprintf(out, "error: embedded command %s: %v\n", e.Name(), err)
				return 1
			}
			if err := os.WriteFile(filepath.Join(cmdDir, e.Name()), raw, 0o644); err != nil {
				fmt.Fprintln(out, "error:", err)
				return 1
			}
			cmdCount++
		}
	}

	instrPath := filepath.Join(append([]string{base}, instrSeg...)...)
	instrVerb, code := ensurePrimer(instrPath, root, out)
	if code != 0 {
		return code
	}

	if upToDate {
		if instrVerb == "unchanged" {
			fmt.Fprintf(out, "fdf skills for %s are up to date (v%s)\n", harnessName, Version)
		} else {
			fmt.Fprintf(out, "fdf skills for %s are up to date (v%s); %s primer %s in %s\n", harnessName, Version, primerHeading, instrVerb, instrPath)
		}
		return 0
	}
	verb := "installed"
	if hadAny {
		verb = "upgraded"
	}
	fmt.Fprintf(out, "%s fdf skills for %s (v%s, root %s): %s", verb, harnessName, Version, root, strings.Join(skillNames, ", "))
	if cmdCount > 0 {
		fmt.Fprintf(out, " + %d command(s)", cmdCount)
	}
	fmt.Fprintf(out, "; %s primer %s in %s\n", primerHeading, instrVerb, instrPath)
	return 0
}

// primer is the instruction-file section teaching an agent what FDF is and
// where the full rules live. It assumes no prior FDF knowledge.
func primer(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done) that must always
reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must exit 0
after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` — read that file when you need exact frontmatter
fields, casing, or validation semantics.

Four bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + ` — are the project's
current stack, architecture, surface (interface) principles for all surfaces,
and build/deployment infrastructure. They are **critical**: filled once by
the fdf-init interview, then changed only with explicit human approval and a
logged reason. Accurate context here is what makes this agentic engineering
rather than vibe coding — read them before designing, and keep them true.
Their upkeep is the human's responsibility.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the four
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute.
- Scaffold with ` + "`fdf new <group>/<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature, propose any needed Context-doc update and apply it only on
  approval, logging the change.
`
}

// ensurePrimer appends the primer to the instruction file unless its heading
// is already present (the section is user-editable after install). Legacy
// managed blocks from pre-0.3 installs are removed. Returns a verb for
// reporting: "added", "updated" (legacy block replaced), or "unchanged".
func ensurePrimer(path, root string, out io.Writer) (string, int) {
	existing, _ := os.ReadFile(path)
	content := string(existing)
	verb := "unchanged"

	if legacyBlockRe.MatchString(content) {
		content = legacyBlockRe.ReplaceAllString(content, "")
		verb = "updated"
	}
	if !regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(primerHeading) + `\s*$`).MatchString(content) {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += primer(root)
		if verb == "unchanged" {
			verb = "added"
		}
	}
	if verb == "unchanged" {
		return verb, 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return verb, 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return verb, 1
	}
	return verb, 0
}
