# FDF — Feature Document Format

**Documentation-as-a-directory for software features.** Each feature is a
Markdown + Gherkin document; its design spec, plan, acceptance tests, and
tasks live in a paired directory beside it; an opinionated CLI validates the
whole bundle so it can never silently drift.

```
docs/features/
├── INDEX.md                      # bundle root (pins fdf_version)
├── LOG.md
├── SPEC.md                       # the format spec, shipped in the bundle
├── STACK.md                      # Context: technology stack
├── ARCHITECTURE.md               # Context: architecture & principles
├── INFRA.md                      # Context: build & deployment infra
└── payments/
    ├── INDEX.md
    ├── instant-refunds.md        # Feature: Gherkin scenarios + status
    └── instant-refunds/
        ├── SPEC.md               # approved design
        ├── PLAN.md               # links every task
        ├── TEST.md               # scenario -> concrete proof
        ├── LOG.md                # feature decisions & interactions
        ├── 01-refund-api.md      # Task (status, depends-on, resource)
        └── 02-refund-ui.md
```

The three **Context documents** (`STACK.md`, `ARCHITECTURE.md`, `INFRA.md`)
are the project's living stack / architecture / infrastructure snapshot.
`fdf init` scaffolds them as stubs; the `fdf-init` skill interview fills them.
They are critical and change only with explicit human approval — accurate
context is what makes this agentic engineering, not vibe coding.

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
                             #   then run the fdf-init skill to fill STACK/ARCHITECTURE/INFRA
fdf new payments/instant-refunds
fdf validate                 # F1-F9 + R1; exit 1 on any violation
fdf serve                    # browse the bundle (bun x mdts)
fdf install claude-code      # agent skills (fdf-help + brainstorm -> plan -> execute)
fdf install codex            #   + a "## Feature Document Format" primer in the
fdf install opencode         #   harness's CLAUDE.md/AGENTS.md (skipped if present)
fdf migrate                  # mechanical upgrade between spec versions
```

`fdf install --root <dir>` (or `FDF_ROOT_DIR`) bakes a non-default bundle
location into the installed skills, which otherwise reference
`docs/features/`. Agents need no prior FDF knowledge: the spec copy at the
bundle root is the reference the skills and primer point them to.

Works the same everywhere: the bundle may be a plain directory or a git
submodule mounted at the same path — `resource:` paths always verify against
the **project** root.

## Spec

FDF is defined by versioned specs under [`spec/`](spec/) — current
[v0.3](spec/0.3.md), prior [v0.2](spec/0.2.md); [SPEC.md](SPEC.md) indexes
them. Each is normative for the bundles pinning its version, and every bundle
vendors its pinned version's spec at `docs/features/SPEC.md`. `testdata/`
fixtures are the executable conformance contract. MIT licensed.
