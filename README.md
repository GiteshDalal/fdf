# FDF — Feature Document Format

**Documentation-as-a-directory for software features.** Each feature is a
Markdown + Gherkin document; its design spec, plan, acceptance tests, and
optional surface/log trail live as **stem-qualified siblings** beside it;
tasks live only under a paired `slug/` directory; an opinionated CLI
validates the whole bundle so it can never silently drift.

```
docs/features/
├── INDEX.md                      # bundle root (pins fdf_version)
├── LOG.md
├── SPEC.md                       # the format spec, shipped in the bundle
├── STACK.md                      # Context: technology stack
├── ARCHITECTURE.md               # Context: architecture & principles
├── SURFACES.md                   # Context: interface principles (all surfaces)
├── INFRA.md                      # Context: build & deployment infra
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

The four **Context documents** (`STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`) are the project's living stack / architecture /
surface-principles / infrastructure snapshot. **SURFACES.md is always
defined** at the bundle root (interface principles for APIs, UIs, CLIs,
events, and inputs — not “UI only”); feature-level `slug.surface.md` is
optional when a feature needs extra surface detail. `fdf init` scaffolds
all four as stubs; the `fdf-init` skill interview fills them. They are
critical and change only with explicit human approval — accurate context is
what makes this agentic engineering, not vibe coding.

## Install

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
fdf validate                 # F1-F9 + R1; exit 1 on any violation
fdf serve                    # browse the bundle (bun x mdts)
fdf install claude-code      # agent skills (fdf-help + brainstorm -> plan -> execute)
fdf install codex            #   + a "## Feature Document Format" primer in the
fdf install opencode         #   harness's CLAUDE.md/AGENTS.md (skipped if present)
fdf migrate                  # mechanical upgrade between adjacent spec versions (e.g. 0.3 → 0.4)
```

`fdf install --root <dir>` (or `FDF_ROOT_DIR`) bakes a non-default bundle
location into the installed skills, which otherwise reference
`docs/features/`. Agents need no prior FDF knowledge: the spec copy at the
bundle root is the reference the skills and primer point them to.

Works the same everywhere: the bundle may be a plain directory or a git
submodule mounted at the same path — `resource:` paths always verify against
the **project** root.

### Upgrading from v0.3

1. Run `fdf migrate` on the bundle (rewrites the pin, moves nested trail
   files to stem-qualified siblings, scaffolds `SURFACES.md` if missing).
2. **Re-run `fdf install`** for each harness you use. Primers and skills
   written under 0.3 describe the paired-directory model (`slug/SPEC.md`,
   three Context docs) and stay stale in global CLAUDE.md/AGENTS.md until
   refreshed. `fdf install` upgrades skills automatically when the version
   marker changes; if your primer heading already exists with old wording,
   edit or replace that section so agents see stem trails and four Context
   docs.

## Spec

FDF is defined by versioned specs under [`spec/`](spec/) — current
[v0.4](spec/0.4.md), prior [v0.3](spec/0.3.md) / [v0.2](spec/0.2.md);
[SPEC.md](SPEC.md) indexes them. Each is normative for the bundles pinning
its version, and every bundle vendors its pinned version's spec at
`docs/features/SPEC.md`. `testdata/` fixtures are the executable conformance
contract. MIT licensed.
