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

// Version is stamped by the CLI (main.version) at dispatch time; this default
// only shows when the package is driven directly, and must track main.version.
var Version = "0.7.0"

// defaultRoot is the bundle root the skill texts are written against; a
// different install root rewrites every occurrence in the skill bodies.
const defaultRoot = "docs/features"

var skillNames = []string{"fdf-help", "fdf-init", "fdf-adopt", "fdf-brainstorm", "fdf-plan", "fdf-execute", "fdf-change", "fdf-debug", "fdf-checkpoint", "fdf-validate"}

// legacyCommands are the Claude Code slash commands shipped before the
// surface became skills-only. They wrapped skills the model can now reach
// directly, so an install removes them — by exact name, never the whole
// commands directory, which also holds the user's own commands.
var legacyCommands = []string{"fdf-init.md", "fdf-new.md", "fdf-validate.md"}

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
	hadCommands      bool     // shipped slash commands before the skills-only surface
}

var harnesses = map[string]harness{
	"claude-code": {
		skillsDir:        []string{".claude", "skills"},
		instrFile:        []string{".claude", "CLAUDE.md"},
		projectSkillsDir: []string{".claude", "skills"},
		projectInstrFile: []string{"CLAUDE.md"}, // repo-root memory file, not .claude/CLAUDE.md
		hadCommands:      true,
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

	// Commands sat beside skills under .claude/ in both scopes.
	removed := 0
	if h.hadCommands {
		removed = removeLegacyCommands(filepath.Join(filepath.Dir(skillsDir), "commands"))
	}

	instrPath := filepath.Join(append([]string{base}, instrSeg...)...)
	instrVerb, code := ensurePrimer(instrPath, root, out)
	if code != 0 {
		return code
	}

	if upToDate && removed == 0 {
		if instrVerb == "unchanged" {
			fmt.Fprintf(out, "fdf skills for %s are up to date (v%s)\n", harnessName, Version)
		} else {
			fmt.Fprintf(out, "fdf skills for %s are up to date (v%s); %s primer %s in %s\n", harnessName, Version, primerHeading, instrVerb, instrPath)
		}
		return 0
	}
	if upToDate {
		fmt.Fprintf(out, "fdf skills for %s are up to date (v%s)", harnessName, Version)
	} else {
		verb := "installed"
		if hadAny {
			verb = "upgraded"
		}
		fmt.Fprintf(out, "%s fdf skills for %s (v%s, root %s): %s", verb, harnessName, Version, root, strings.Join(skillNames, ", "))
	}
	if removed > 0 {
		fmt.Fprintf(out, "; removed %d superseded slash command(s)", removed)
	}
	fmt.Fprintf(out, "; %s primer %s in %s\n", primerHeading, instrVerb, instrPath)
	return 0
}

// removeLegacyCommands deletes the superseded fdf slash commands from dir and
// reports how many were removed. It touches only the names fdf shipped, and
// prunes dir only when fdf's removals left it empty.
func removeLegacyCommands(dir string) int {
	n := 0
	for _, name := range legacyCommands {
		if err := os.Remove(filepath.Join(dir, name)); err == nil {
			n++
		}
	}
	if n > 0 {
		if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
			os.Remove(dir)
		}
	}
	return n
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
status (draft → specified → planned → implementing → done → retired, or
adopted → retired for a capability documented from code that predates the
bundle) that must always reflect reality; the ` + "`fdf validate`" + ` CLI gates
consistency and must exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Five bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + `,
` + "`DOMAIN.md`" + ` — are the project's current stack, architecture, surface
(interface) principles for all surfaces, build/deployment infrastructure, and
**domain language**. They are **critical**: filled once by the fdf-init
interview, then changed only with explicit human approval and a logged reason.
Accurate context here is what makes this agentic engineering rather than vibe
coding — read them before designing, and keep them true. Their upkeep is the
human's responsibility.

` + "`DOMAIN.md`" + ` is the project's vocabulary: one canonical name per concept
and the words banned in its place. It governs the project's **internal**
language — the bundle's documents and the identifiers in the code — where
calling one thing ` + "`Item`" + ` here and ` + "`Product`" + ` there is the drift
it exists to stop. It does not govern what a person reads on a surface: UI
labels, locale and translation files, help text and other user-facing copy
may say "store" for a Venue on purpose. That is a surface decision, not
drift — never rewrite such copy to match the lexicon. F12 reports a banned
word in every document except ` + "`slug.test.md`" + ` and ` + "`slug.surface.md`" + `
(which quote what a surface shows), and in document names; ` + "`fdf lexicon`" + `
lists each one, and ` + "`fdf lexicon --term <Term> --fix`" + ` sweeps a term once
its other senses are triaged into ` + "`except:`" + `.

**Practice documents** under ` + "`" + root + "/practices/`" + ` (` + "`type: Practice`" + `)
are the project's binding answers to *how do we do X* for recurring
mechanisms — authorization, permission checks, payment capture, database
access. Before writing code in a path a practice's ` + "`applies-to`" + ` covers,
read it and follow its ` + "`# Rules`" + `; a deliberate divergence is an approved
` + "`# Exceptions`" + ` entry, never silence. Practices are living and binding:
propose, get explicit approval, then write.

**Debt documents** under ` + "`" + root + "/debts/`" + ` (` + "`type: Debt`" + `) record
known gaps between what the project says and what the code does — work left
undone, and rules the code does not follow everywhere yet. ` + "`fdf debt`" + `
reads the register and ` + "`fdf debt --open`" + ` shows what is outstanding;
check it before diagnosing something, because a filed gap is not a discovery.
When work knowingly leaves something behind, file it rather than rounding it
off: ` + "`fdf debt [<group>/]<slug>`" + `.

**Bug documents** under ` + "`" + root + "/bugs/`" + ` (` + "`type: Bug`" + `) record
known defects — the software doing something observably wrong — that have
not been repaired yet. ` + "`fdf bug --open`" + ` shows them; check both registers
before diagnosing something. A bug is never resolved in place: its repair is a
` + "`Fix`" + ` or ` + "`Change`" + ` (` + "`fdf fix --from bugs/<id>`" + `) that names it in
` + "`resolves`" + `.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, practices, debts, bugs,
the Context docs) and are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode. Three **maintenance edits**
reach every document, frozen ones included, because each only puts one name in
place of another: a lexicon fix (a banned word replaced by its term), a
reference repair after a move (` + "`fdf mv`" + ` does it whole), and a path repair
after code moves. Each is logged; none needs a change document.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the five
  Context docs. Feature work is blocked (rule F9) while they're unfilled. On a
  codebase that predates the bundle, the fdf-adopt skill then maps what it
  already does (` + "`fdf adopt`" + `).
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/adopted/retired → fdf-change;
  a capability that exists in code but has no feature → fdf-adopt.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new — or a bug on the
  register when it is not repaired now.
- Scaffold with ` + "`fdf new`" + `, ` + "`fdf adopt`" + `, ` + "`fdf practice`" + `,
  ` + "`fdf debt`" + ` and ` + "`fdf bug`" + `; rename or move a document with
  ` + "`fdf mv`" + `, never by hand; validate with ` + "`fdf validate`" + `.
- Record what happened with ` + "`fdf log <id> \"<entry>\"`" + `: an entry goes in
  the log of the document it is about (a feature's ` + "`slug.log.md`" + `,
  created on first use), and only what concerns the whole bundle goes in the
  root ` + "`LOG.md`" + `. A feature with an interface — a screen, an endpoint, a
  command, an event — describes it in ` + "`slug.surface.md`" + `, kept current
  like its Gherkin; one without says ` + "`surface: none`" + ` in its frontmatter.
  Each test case is a ` + "`## <scenario name>`" + ` heading, matched exactly.
- Once a feature is **done** or **adopted**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, review the project-level documents: propose any
  needed Context-doc update, and ask whether the work established a mechanism a
  second feature has now repeated (a new practice), diverged from an existing
  one (an ` + "`# Exceptions`" + ` entry), or knowingly left something undone (a
  debt) or broken (a bug). Apply only on approval, logging the change.
- Run the fdf-checkpoint skill regularly — before a release, and after
  dependency, tooling or infrastructure work no feature recorded — to keep the
  Context docs, ` + "`SPEC.md`" + ` and this file current, free of repetition, and
  consistent with the code and each other. Never hand-edit this section:
  ` + "`fdf install`" + ` owns it, and stops refreshing it once it is edited.
`
}

// primerV063 is the primer shipped by the v0.6.3 release (the lexicon fix
// as the one edit every document takes), kept so an upgrade can recognize an
// untouched managed section and refresh it in place.
func primerV063(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Five bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + `,
` + "`DOMAIN.md`" + ` — are the project's current stack, architecture, surface
(interface) principles for all surfaces, build/deployment infrastructure, and
**domain language**. They are **critical**: filled once by the fdf-init
interview, then changed only with explicit human approval and a logged reason.
Accurate context here is what makes this agentic engineering rather than vibe
coding — read them before designing, and keep them true. Their upkeep is the
human's responsibility.

` + "`DOMAIN.md`" + ` is the project's vocabulary: one canonical name per concept
and the words banned in its place. It governs the project's **internal**
language — the bundle's documents and the identifiers in the code — where
calling one thing ` + "`Item`" + ` here and ` + "`Product`" + ` there is the drift
it exists to stop. It does not govern what a person reads on a surface: UI
labels, locale and translation files, help text and other user-facing copy
may say "store" for a Venue on purpose. That is a surface decision, not
drift — never rewrite such copy to match the lexicon. F12 reports a banned
word in a feature's Gherkin.

**Practice documents** under ` + "`" + root + "/practices/`" + ` (` + "`type: Practice`" + `)
are the project's binding answers to *how do we do X* for recurring
mechanisms — authorization, permission checks, payment capture, database
access. Before writing code in a path a practice's ` + "`applies-to`" + ` covers,
read it and follow its ` + "`# Rules`" + `; a deliberate divergence is an approved
` + "`# Exceptions`" + ` entry, never silence. Practices are living and binding:
propose, get explicit approval, then write.

**Debt documents** under ` + "`" + root + "/debts/`" + ` (` + "`type: Debt`" + `) record
known gaps between what the project says and what the code does — work left
undone, and rules the code does not follow everywhere yet. ` + "`fdf debt`" + `
reads the register and ` + "`fdf debt --open`" + ` shows what is outstanding;
check it before diagnosing something, because a filed gap is not a discovery.
When work knowingly leaves something behind, file it rather than rounding it
off: ` + "`fdf debt [<group>/]<slug>`" + `.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, practices, debts, the
Context docs) and are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode. The one edit every document
takes, frozen ones included, is a **lexicon fix**: a word ` + "`DOMAIN.md`" + ` bans
replaced by its term. It changes no behavior, so it needs no change document.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the five
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new.
- Scaffold with ` + "`fdf new <group>/<slug>`" + ` and
  ` + "`fdf practice [<group>/]<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, review the project-level documents: propose any
  needed Context-doc update, and ask whether the work established a mechanism a
  second feature has now repeated (a new practice), diverged from an existing
  one (an ` + "`# Exceptions`" + ` entry), or knowingly left something undone (a
  debt). Apply only on approval, logging the change.
- Run the fdf-checkpoint skill regularly — before a release, and after
  dependency, tooling or infrastructure work no feature recorded — to keep the
  Context docs, ` + "`SPEC.md`" + ` and this file current, free of repetition, and
  consistent with the code and each other. Never hand-edit this section:
  ` + "`fdf install`" + ` owns it, and stops refreshing it once it is edited.
`
}

// primerV062 is the primer shipped by the v0.6.2 release (episodic documents
// never rewritten, with no lexicon fix), kept so an upgrade can recognize an
// untouched managed section written by it and refresh it.
func primerV062(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Five bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + `,
` + "`DOMAIN.md`" + ` — are the project's current stack, architecture, surface
(interface) principles for all surfaces, build/deployment infrastructure, and
**domain language**. They are **critical**: filled once by the fdf-init
interview, then changed only with explicit human approval and a logged reason.
Accurate context here is what makes this agentic engineering rather than vibe
coding — read them before designing, and keep them true. Their upkeep is the
human's responsibility.

` + "`DOMAIN.md`" + ` is the project's vocabulary: one canonical name per concept
and the words banned in its place. It governs the project's **internal**
language — the bundle's documents and the identifiers in the code — where
calling one thing ` + "`Item`" + ` here and ` + "`Product`" + ` there is the drift
it exists to stop. It does not govern what a person reads on a surface: UI
labels, locale and translation files, help text and other user-facing copy
may say "store" for a Venue on purpose. That is a surface decision, not
drift — never rewrite such copy to match the lexicon. F12 reports a banned
word in a feature's Gherkin.

**Practice documents** under ` + "`" + root + "/practices/`" + ` (` + "`type: Practice`" + `)
are the project's binding answers to *how do we do X* for recurring
mechanisms — authorization, permission checks, payment capture, database
access. Before writing code in a path a practice's ` + "`applies-to`" + ` covers,
read it and follow its ` + "`# Rules`" + `; a deliberate divergence is an approved
` + "`# Exceptions`" + ` entry, never silence. Practices are living and binding:
propose, get explicit approval, then write.

**Debt documents** under ` + "`" + root + "/debts/`" + ` (` + "`type: Debt`" + `) record
known gaps between what the project says and what the code does — work left
undone, and rules the code does not follow everywhere yet. ` + "`fdf debt`" + `
reads the register and ` + "`fdf debt --open`" + ` shows what is outstanding;
check it before diagnosing something, because a filed gap is not a discovery.
When work knowingly leaves something behind, file it rather than rounding it
off: ` + "`fdf debt [<group>/]<slug>`" + `.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, practices, debts, the
Context docs) and are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the five
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new.
- Scaffold with ` + "`fdf new <group>/<slug>`" + ` and
  ` + "`fdf practice [<group>/]<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, review the project-level documents: propose any
  needed Context-doc update, and ask whether the work established a mechanism a
  second feature has now repeated (a new practice), diverged from an existing
  one (an ` + "`# Exceptions`" + ` entry), or knowingly left something undone (a
  debt). Apply only on approval, logging the change.
- Run the fdf-checkpoint skill regularly — before a release, and after
  dependency, tooling or infrastructure work no feature recorded — to keep the
  Context docs, ` + "`SPEC.md`" + ` and this file current, free of repetition, and
  consistent with the code and each other. Never hand-edit this section:
  ` + "`fdf install`" + ` owns it, and stops refreshing it once it is edited.
`
}

// primerV061 is the primer shipped by the v0.6.1 release (DOMAIN.md scoped to
// internal language; no fdf-checkpoint), kept so an upgrade can recognize an
// untouched managed section written by it and refresh it.
func primerV061(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Five bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + `,
` + "`DOMAIN.md`" + ` — are the project's current stack, architecture, surface
(interface) principles for all surfaces, build/deployment infrastructure, and
**domain language**. They are **critical**: filled once by the fdf-init
interview, then changed only with explicit human approval and a logged reason.
Accurate context here is what makes this agentic engineering rather than vibe
coding — read them before designing, and keep them true. Their upkeep is the
human's responsibility.

` + "`DOMAIN.md`" + ` is the project's vocabulary: one canonical name per concept
and the words banned in its place. It governs the project's **internal**
language — the bundle's documents and the identifiers in the code — where
calling one thing ` + "`Item`" + ` here and ` + "`Product`" + ` there is the drift
it exists to stop. It does not govern what a person reads on a surface: UI
labels, locale and translation files, help text and other user-facing copy
may say "store" for a Venue on purpose. That is a surface decision, not
drift — never rewrite such copy to match the lexicon. F12 reports a banned
word in a feature's Gherkin.

**Practice documents** under ` + "`" + root + "/practices/`" + ` (` + "`type: Practice`" + `)
are the project's binding answers to *how do we do X* for recurring
mechanisms — authorization, permission checks, payment capture, database
access. Before writing code in a path a practice's ` + "`applies-to`" + ` covers,
read it and follow its ` + "`# Rules`" + `; a deliberate divergence is an approved
` + "`# Exceptions`" + ` entry, never silence. Practices are living and binding:
propose, get explicit approval, then write.

**Debt documents** under ` + "`" + root + "/debts/`" + ` (` + "`type: Debt`" + `) record
known gaps between what the project says and what the code does — work left
undone, and rules the code does not follow everywhere yet. ` + "`fdf debt`" + `
reads the register and ` + "`fdf debt --open`" + ` shows what is outstanding;
check it before diagnosing something, because a filed gap is not a discovery.
When work knowingly leaves something behind, file it rather than rounding it
off: ` + "`fdf debt [<group>/]<slug>`" + `.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, practices, debts, the
Context docs) and are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the five
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new.
- Scaffold with ` + "`fdf new <group>/<slug>`" + ` and
  ` + "`fdf practice [<group>/]<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, review the project-level documents: propose any
  needed Context-doc update, and ask whether the work established a mechanism a
  second feature has now repeated (a new practice), diverged from an existing
  one (an ` + "`# Exceptions`" + ` entry), or knowingly left something undone (a
  debt). Apply only on approval, logging the change.
`
}

// primerV06 is the primer shipped by the v0.6.0 release (five Context
// documents, practices, debts; DOMAIN.md described without its surface
// boundary), kept so an upgrade can recognize an untouched managed section
// written by it and refresh it.
func primerV06(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Five bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + `,
` + "`DOMAIN.md`" + ` — are the project's current stack, architecture, surface
(interface) principles for all surfaces, build/deployment infrastructure, and
**domain language**. They are **critical**: filled once by the fdf-init
interview, then changed only with explicit human approval and a logged reason.
Accurate context here is what makes this agentic engineering rather than vibe
coding — read them before designing, and keep them true. Their upkeep is the
human's responsibility.

` + "`DOMAIN.md`" + ` is the project's vocabulary: one canonical name per concept
and the words banned in its place. Use those names in Gherkin, in specs, and
in the identifiers you write — calling one thing ` + "`Item`" + ` here and
` + "`Product`" + ` there is the drift it exists to stop. F12 reports a banned
word in a feature's Gherkin.

**Practice documents** under ` + "`" + root + "/practices/`" + ` (` + "`type: Practice`" + `)
are the project's binding answers to *how do we do X* for recurring
mechanisms — authorization, permission checks, payment capture, database
access. Before writing code in a path a practice's ` + "`applies-to`" + ` covers,
read it and follow its ` + "`# Rules`" + `; a deliberate divergence is an approved
` + "`# Exceptions`" + ` entry, never silence. Practices are living and binding:
propose, get explicit approval, then write.

**Debt documents** under ` + "`" + root + "/debts/`" + ` (` + "`type: Debt`" + `) record
known gaps between what the project says and what the code does — work left
undone, and rules the code does not follow everywhere yet. ` + "`fdf debt`" + `
reads the register and ` + "`fdf debt --open`" + ` shows what is outstanding;
check it before diagnosing something, because a filed gap is not a discovery.
When work knowingly leaves something behind, file it rather than rounding it
off: ` + "`fdf debt [<group>/]<slug>`" + `.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, practices, debts, the
Context docs) and are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the five
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new.
- Scaffold with ` + "`fdf new <group>/<slug>`" + ` and
  ` + "`fdf practice [<group>/]<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, review the project-level documents: propose any
  needed Context-doc update, and ask whether the work established a mechanism a
  second feature has now repeated (a new practice), diverged from an existing
  one (an ` + "`# Exceptions`" + ` entry), or knowingly left something undone (a
  debt). Apply only on approval, logging the change.
`
}

// primerV051 is the primer shipped by the v0.5.1 release (four Context
// documents, no practices), kept so an upgrade can recognize an untouched
// managed section written by it and refresh it.
func primerV051(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Four bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + ` — are the project's
current stack, architecture, surface (interface) principles for all surfaces,
and build/deployment infrastructure. They are **critical**: filled once by
the fdf-init interview, then changed only with explicit human approval and a
logged reason. Accurate context here is what makes this agentic engineering
rather than vibe coding — read them before designing, and keep them true.
Their upkeep is the human's responsibility.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, the Context docs) and
are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the four
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- When something is broken, start with the fdf-debug skill: find the root
  cause before any fix, and it routes the repair — a ` + "`Fix`" + ` when the code
  drifted from the document, a ` + "`Change`" + ` when the document itself has to
  change, a new feature when the behavior is genuinely new.
- Scaffold with ` + "`fdf new <group>/<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, propose any needed Context-doc update and
  apply it only on approval, logging the change.
`
}

// primerV05 is the primer shipped by the v0.5.0 release, kept so an upgrade
// can recognize an untouched managed section written by it and refresh it.
func primerV05(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, and decision log live as
stem-qualified trail siblings (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `,
` + "`slug.test.md`" + `, optional ` + "`slug.surface.md`" + `/` + "`slug.log.md`" + `);
tasks live only under a ` + "`slug/`" + ` directory. Feature frontmatter carries a
status (draft → specified → planned → implementing → done → retired) that must
always reflect reality; the ` + "`fdf validate`" + ` CLI gates consistency and must
exit 0 after any bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` and are also printed by ` + "`fdf spec`" + ` — read
either when you need exact frontmatter fields, casing, or validation
semantics. ` + "`fdf help`" + ` documents every command with examples.

Four bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`SURFACES.md`" + `, ` + "`INFRA.md`" + ` — are the project's
current stack, architecture, surface (interface) principles for all surfaces,
and build/deployment infrastructure. They are **critical**: filled once by
the fdf-init interview, then changed only with explicit human approval and a
logged reason. Accurate context here is what makes this agentic engineering
rather than vibe coding — read them before designing, and keep them true.
Their upkeep is the human's responsibility.

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system today (the feature's
Gherkin, ` + "`slug.test.md`" + `, ` + "`slug.surface.md`" + `, the Context docs) and
are amended in place. Episodic documents are frozen records of one piece of
work (` + "`slug.spec.md`" + `, ` + "`slug.plan.md`" + `, tasks, changes, logs) and are
never rewritten — new work gets a new episode.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the four
  Context docs. Feature work is blocked (rule F9) while they're unfilled.
- Before writing code, route by feature status using the fdf-help skill:
  no feature/draft → fdf-brainstorm, specified → fdf-plan,
  planned/implementing → fdf-execute, done/retired → fdf-change.
- Scaffold with ` + "`fdf new <group>/<slug>`" + `; validate with ` + "`fdf validate`" + `.
- Once a feature is **done**, never edit its behavior in place and never fork
  a second feature document for the same capability. Post-delivery work is a
  document under ` + "`" + root + "/changes/`" + `: ` + "`fdf change`" + ` when the
  feature's Gherkin must change, ` + "`fdf fix`" + ` when the code merely drifted
  from what the document already says. Rule F10 will not let one reach
  ` + "`done`" + ` until the features it claims to alter actually say so.
- Code that changes behavior without touching the bundle makes the bundle
  lie — record the feature first, then implement.
- After a feature or a change, propose any needed Context-doc update and
  apply it only on approval, logging the change.
`
}

// primerV04 is the v0.4 primer, kept so an upgrade can recognize an
// untouched managed section written by a 0.4.x release and refresh it.
func primerV04(root string) string {
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

// legacyPrimers reproduces the primer text earlier CLI versions wrote, so an
// upgrade can recognize an untouched managed section and refresh it in place.
// Every release that changes primer() must append the superseded text here —
// otherwise re-running `fdf install` upgrades the skills but leaves the
// instruction file teaching the old format.
var legacyPrimers = []func(root string) string{primerV03, primerV04, primerV05, primerV051, primerV06, primerV061, primerV062, primerV063}

// primerV03 is the primer shipped by the v0.3-era CLI (paired-directory
// layout, three Context docs). Kept verbatim for upgrade detection.
func primerV03(root string) string {
	return primerHeading + `

Projects on this machine may document software features with FDF (Feature
Document Format): the directory ` + "`" + root + "/`" + ` in a project is an FDF
**bundle** — every feature is a Markdown + Gherkin document, and its design
spec, implementation plan, acceptance tests, tasks, and decision log live in a
paired directory beside it. Feature frontmatter carries a status
(draft → specified → planned → implementing → done) that must always reflect
reality; the ` + "`fdf validate`" + ` CLI gates consistency and must exit 0 after any
bundle edit. The full format rules ship inside the bundle at
` + "`" + root + "/SPEC.md`" + ` — read that file when you need exact frontmatter
fields, casing, or validation semantics.

Three bundle-root **Context documents** — ` + "`" + root + "/STACK.md`" + `,
` + "`ARCHITECTURE.md`" + `, ` + "`INFRA.md`" + ` — are the project's current stack,
architecture, and build/deployment infrastructure. They are **critical**:
filled once by the fdf-init interview, then changed only with explicit human
approval and a logged reason. Accurate context here is what makes this agentic
engineering rather than vibe coding — read them before designing, and keep
them true. Their upkeep is the human's responsibility.

Working in an FDF project:

- First run: after ` + "`fdf init`" + `, use the fdf-init skill to fill the three
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

var primerHeadingRe = regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(primerHeading) + `\s*$`)
var nextH2Re = regexp.MustCompile(`(?m)^## `)

// primerSection locates the managed primer section: from the start of the
// heading line to the start of the next `## ` heading (or EOF).
func primerSection(content string) (start, end int, ok bool) {
	loc := primerHeadingRe.FindStringIndex(content)
	if loc == nil {
		return 0, 0, false
	}
	if next := nextH2Re.FindStringIndex(content[loc[1]:]); next != nil {
		return loc[0], loc[1] + next[0], true
	}
	return loc[0], len(content), true
}

// sectionMatchesLegacy reports whether an existing primer section is exactly
// a superseded shipped primer (rendered with this install's root).
func sectionMatchesLegacy(section, root string) bool {
	for _, legacy := range legacyPrimers {
		if section == strings.TrimRight(legacy(root), "\n") {
			return true
		}
	}
	return false
}

// ensurePrimer places or refreshes the instruction-file primer. A missing
// section is appended. An existing section that matches a primer some CLI
// version shipped (current → no-op; superseded → replaced in place) is
// managed content; anything else was user-edited and is left untouched with
// a warning. Legacy pre-0.3 managed blocks are removed. Returns a verb for
// reporting: "added", "updated", or "unchanged".
func ensurePrimer(path, root string, out io.Writer) (string, int) {
	existing, _ := os.ReadFile(path)
	content := string(existing)
	verb := "unchanged"

	if legacyBlockRe.MatchString(content) {
		content = legacyBlockRe.ReplaceAllString(content, "")
		verb = "updated"
	}
	if s, e, ok := primerSection(content); ok {
		section := strings.TrimRight(content[s:e], "\n")
		switch {
		case section == strings.TrimRight(primer(root), "\n"):
			// Current text; nothing to do.
		case sectionMatchesLegacy(section, root):
			repl := strings.TrimRight(primer(root), "\n") + "\n"
			if e < len(content) {
				repl += "\n"
			}
			content = content[:s] + repl + content[e:]
			verb = "updated"
		default:
			fmt.Fprintf(out, "note: %s section in %s differs from the shipped primer (user-edited?) — left as-is; delete the section and re-run `fdf install` to refresh it\n", primerHeading, path)
		}
	} else {
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
