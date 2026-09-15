# FDF — Feature Document Format

**Documentation-as-a-directory for software features.** Each feature is a
Markdown + Gherkin document; its design spec, plan, acceptance tests, and
optional surface/log trail live as **stem-qualified siblings** beside it;
tasks live only under a paired `slug/` directory; post-delivery change
requests and bug fixes live under `changes/`; an opinionated CLI validates
the whole bundle so it can never silently drift.

```
docs/features/
├── INDEX.md                      # bundle root (pins fdf_version)
├── LOG.md
├── SPEC.md                       # the format spec, shipped in the bundle
├── STACK.md                      # Context: technology stack
├── ARCHITECTURE.md               # Context: architecture & principles
├── SURFACES.md                   # Context: interface principles (all surfaces)
├── INFRA.md                      # Context: build & deployment infra
├── changes/                      # post-delivery work (flat or grouped)
│   ├── INDEX.md
│   └── refund-rounding.md        # type: Fix — the code drifted from the doc
└── payments/
    ├── INDEX.md
    ├── instant-refunds.md        # Feature: Gherkin scenarios + status
    ├── instant-refunds.spec.md   # approved design (type: Spec)
    ├── instant-refunds.plan.md   # links every task (type: Plan)
    ├── instant-refunds.test.md   # scenario -> concrete proof (type: Test)
    ├── instant-refunds.surface.md  # optional (type: Surface)
    ├── instant-refunds.log.md    # optional feature decisions
    └── instant-refunds/          # task directory ONLY
        ├── 01-refund-api.md
        └── 02-refund-ui.md
```

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system *today* — the feature's
Gherkin, `slug.test.md`, `slug.surface.md`, the Context docs — and are amended
in place. Episodic documents are frozen records of one piece of work —
`slug.spec.md`, `slug.plan.md`, tasks, changes, log entries — and are never
rewritten. So a delivered feature that changes keeps **one** feature document,
whose Gherkin is edited, and gains a **new** episode under `changes/`. Its
original spec and plan stay untouched: they record how it was first built,
which is the context you need to judge the change.

A `Change` alters what a delivered feature does; a `Fix` restores behavior the
feature document already describes. Both name the features they touch in
`affects:` — so one bug spanning several features, or one request spanning
them, is a single document. Both declare the effects they will have, and rule
**F10** refuses to let one reach `done` while the features it claims to alter
still describe the old behavior. That is the drift FDF exists to prevent,
enforced rather than hoped for.

The four **Context documents** (`STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`) are the project's living stack / architecture /
surface-principles / infrastructure snapshot. **SURFACES.md is always
defined** at the bundle root (interface principles for APIs, UIs, CLIs,
events, and inputs — not “UI only”); feature-level `slug.surface.md` is
optional when a feature needs extra surface detail. `fdf init` scaffolds
all four as stubs; the `fdf-init` skill interview fills them. They are
critical and change only with explicit human approval — accurate context is
what makes this agentic engineering, not vibe coding.

## Why

Agents made writing code cheap. What broke is the record of *why* the code
exists: the design lived in a chat scrollback that is gone, the plan was
never written down, and the tests prove whatever the code happens to do.
That is vibe coding, and it compounds — every future change starts from
archaeology.

FDF is the opposite bet: the documentation is the interface between humans
and agents. Intent, design, plan, and proof live in the repo beside the
code; a feature's status must say what is actually true of it; and
`fdf validate` turns drift into a failing build instead of a discovery six
months later. Humans approve specs and context; agents do the mechanical
work in between. Built for **agentic engineering, not vibe coding**.

## Install

- One-liner (macOS/Linux, no Go needed):
```bash
curl -fsSL https://raw.githubusercontent.com/GiteshDalal/fdf/main/install.sh | bash
```
- Binaries on the [releases page](https://github.com/GiteshDalal/fdf/releases)
- With [mise](https://mise.jdx.dev)
```
mise use ubi:GiteshDalal/fdf
```
- With [Go](https://go.dev/)
```
go install github.com/GiteshDalal/fdf/cli/cmd/fdf@latest
```

## Use

```bash
fdf init                     # scaffold docs/features/ + SPEC.md + context stubs (or FDF_ROOT_DIR / --root)
                             #   then run the fdf-init skill to fill STACK/ARCHITECTURE/SURFACES/INFRA
fdf new payments/instant-refunds
fdf validate                 # F1-F10 + R1; exit 1 on any violation
fdf spec                     # print the format spec (-v 0.4 for an older one)
fdf help                     # every command with examples
fdf serve                    # browse the bundle (bun x mdts)

# after a feature ships
fdf change --affects payments/instant-refunds refund-window   # behavior should differ
fdf fix    --affects payments/instant-refunds refund-rounding # code drifted from the doc
fdf history payments/instant-refunds                          # what happened since it shipped
fdf release 1.2.0            # derive the release doc from `version:` fields; --ship to close it
fdf install claude-code      # user-level skills + "## Feature Document Format" primer
fdf install codex            #   (primer skipped if the heading is already present)
fdf install opencode
fdf install --project claude-code   # project-level: skills under .claude/, primer in ./CLAUDE.md
fdf migrate                  # mechanical upgrade to the current spec version
```

`fdf install` defaults to **user-level** (under your home directory). Prefer
`fdf install --project <harness>` for repos that actually carry an FDF bundle:
skills and the primer stay out of unrelated projects, and a stale primer after
`fdf migrate` is a per-repo refresh instead of a machine-global one. User-level
and project-level installs coexist and upgrade independently (each destination
has its own `.fdf-version` markers). A machine with both will carry two primers
— harnesses merge memory files, so that is fine.

| Scope | Harness | Skills | Instruction file | Commands |
|---|---|---|---|---|
| user | claude-code | `~/.claude/skills/` | `~/.claude/CLAUDE.md` | `~/.claude/commands/` |
| user | codex | `~/.codex/skills/` | `~/.codex/AGENTS.md` | — |
| user | opencode | `~/.config/opencode/skills/` | `~/.config/opencode/AGENTS.md` | — |
| project | claude-code | `<proj>/.claude/skills/` | `<proj>/CLAUDE.md` | `<proj>/.claude/commands/` |
| project | codex | `<proj>/.codex/skills/` | `<proj>/AGENTS.md` | — |
| project | opencode | `<proj>/.opencode/skills/` | `<proj>/AGENTS.md` | — |

`--project` outside a git working tree is a usage error (exit 2).
`fdf install --root <dir>` (or `FDF_ROOT_DIR`) bakes a non-default bundle
location into the installed skills, which otherwise reference
`docs/features/`; it composes with `--project` unchanged. Agents need no prior
FDF knowledge: the spec copy at the bundle root is the reference the skills
and primer point them to.

Works the same everywhere: the bundle may be a plain directory or a git
submodule mounted at the same path — `resource:` paths always verify against
the **project** root.

### Upgrading to v0.5

1. Run `fdf migrate` on the bundle. From **v0.4** this is purely additive —
   the pin moves, `SPEC.md` is re-vendored and `changes/INDEX.md` appears;
   nothing else moves or changes shape, and a bundle that never records a
   change validates identically. From **v0.2/v0.3** it also lifts nested
   trail files to stem-qualified siblings and scaffolds `SURFACES.md`.
2. **Re-run `fdf install`** for each harness you use. Skills and primers
   written under an older version do not know about `changes/`, the `Change`
   and `Fix` types, or `retired`, and stay stale in CLAUDE.md/AGENTS.md until
   refreshed. `fdf install` upgrades skills automatically when the version
   marker changes; if your primer heading already exists with old wording,
   edit or replace that section.

## Spec

FDF is defined by versioned specs under [`spec/`](spec/) — current
[v0.5](spec/0.5.md), prior [v0.4](spec/0.4.md) / [v0.3](spec/0.3.md) /
[v0.2](spec/0.2.md);
[SPEC.md](SPEC.md) indexes them. Each is normative for the bundles pinning
its version, and every bundle vendors its pinned version's spec at
`docs/features/SPEC.md`. `testdata/` fixtures are the executable conformance
contract. MIT licensed.
