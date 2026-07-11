# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

This is the **source repository for the `fdf` CLI and its skills** — not a project that
*uses* FDF. There is no `docs/features/` bundle here, so the global "Feature Document Format"
workflow instructions (route by feature status, `fdf new`, etc.) do **not** apply to work in
this repo. FDF bundles live under `testdata/` as conformance fixtures; the real bundles this
tool manages exist in *other* projects.

FDF (Feature Document Format) is "documentation-as-a-directory": each software feature is a
Markdown + Gherkin document whose design spec, plan, acceptance tests, and optional surface/log
trail live as **stem-qualified siblings** (`slug.spec.md`, `slug.plan.md`, `slug.test.md`,
optional `slug.surface.md` / `slug.log.md`); tasks live only under a `slug/` directory. Four
bundle-root Context docs (`STACK.md`, `ARCHITECTURE.md`, `SURFACES.md`, `INFRA.md`) hold
project context. This repo ships:

1. A Go CLI (`cli/cmd/fdf`) that scaffolds and **validates** those bundles.
2. Harness-neutral **skills** (`skills/`) and per-harness **adapters** (`harness/`) that teach
   AI agents the brainstorm → plan → execute workflow.
3. **Versioned specs** (`spec/`) that are normative for the bundles pinning each version.

Assets 2 and 3 are compiled into the binary via `//go:embed` (see `embed.go`) so `fdf install`
and `fdf init`/`fdf migrate` can write them out anywhere.

## Commands

```bash
go build ./cli/cmd/fdf         # build (committed binary ./fdf is a convenience copy)
go test ./...                  # full test suite
go vet ./...                   # CI runs this
gofmt -l .                     # CI fails if this prints ANY file — run gofmt -w before commit

# Run one conformance fixture (fixtures are subtests of TestConformanceFixtures):
go test ./cli/internal/bundle -run 'TestConformanceFixtures/valid-minimal' -v

# Run the CLI locally without installing:
go run ./cli/cmd/fdf validate
go run ./cli/cmd/fdf new payments/instant-refunds
```

CI (`.github/workflows/ci.yml`, ubuntu + macos, Go 1.26) runs `go vet`, `go test`, the
`gofmt -l` cleanliness check, and a shell lint that every `skills/*/SKILL.md` starts with `---`
and has `name:`/`description:` frontmatter. Match all four before pushing. Releases are built by
goreleaser (`.goreleaser.yaml`) on tag push.

## Architecture

The CLI is a flat command dispatcher (`cli/cmd/fdf/main.go`: a `map[string]func` over
`validate|init|new|install|serve|migrate|version`). Each command is a thin wrapper around one
`cli/internal/` package. Commands write usage/errors to stdout (not stderr) so tests capture
everything, and flags must precede positional args (`ContinueOnError` FlagSets).

- **`cli/internal/fdfroot`** — root resolution, used by every command. Bundle root precedence:
  `--root` flag > `FDF_ROOT_DIR` env > `docs/features`. Relative roots resolve against the
  **project root**, found by walking up to the topmost `.git`. A `.git` *file* (submodule)
  marks a boundary but the walk continues to the superproject — so `resource:` paths always
  verify against the real project root even when the bundle is a git submodule.

- **`cli/internal/bundle`** (`validate.go`) — the heart of the tool. `Validate()` is the
  enforcement engine for the spec. Rules are coded **F1–F9** (format conformance) and **R1**
  (repo integrity); every error message ends with its rule code, e.g. `(F4)`. Highlights:
  - `readPin()` reads `fdf_version` from the root `INDEX.md` *before* the directory walk,
    because version-gated rules (Context docs, stem trail layout) must be known for every
    file and `WalkDir` visits lexically (`ARCHITECTURE.md` sorts before `INDEX.md`).
  - Validates spec **v0.2, v0.3, and v0.4** (`supportedVersions`); a pin outside that set is
    an F1 error pointing at `fdf migrate`. v0.4 uses stem-trail siblings + four Context docs
    (including `SURFACES.md`); v0.2/v0.3 keep the nested paired-directory layout.
  - Feature statuses `draft → specified → planned → implementing → done` drive the F4/F8
    status↔artifact invariants (e.g. v0.4 `planned` requires `slug.test.md` with a case per
    Gherkin scenario; `done` requires all tasks done). `Options.FreshStubsAdvisory`
    downgrades F9 (unfilled Context stub) from error to warning — only `fdf migrate` sets it.

- **`cli/internal/scaffold`** (`init`, `new`), **`cli/internal/install`** (harness adapters +
  a `## Feature Document Format` primer, idempotent, never clobbers user edits), and
  **`cli/internal/migrate`** (mechanical upgrades between adjacent spec versions; 0.3→0.4
  rewrites nested trail files to stem siblings and scaffolds `SURFACES.md`).

### Layout the validator expects (v0.4)

```
docs/features/
├── STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md   # type: Context
└── group/
    ├── slug.md                    # Feature
    ├── slug.spec.md / .plan.md / .test.md
    ├── slug.surface.md / .log.md  # optional
    └── slug/                      # tasks only: NN-….md
```

Trail roles are only `spec`, `plan`, `test`, `surface`, `log`. Task dirs must not contain
nested SPEC/PLAN/TEST or LOG.md.

### The conformance contract

`testdata/*/` fixtures are the **executable spec**. Each fixture is a directory with a
`bundle/` (and optionally a `repo/` wrapper for R1 tests) plus an `expect.txt` of `exit:` and
`contains:` assertions. `TestConformanceFixtures` (`conformance_test.go`) runs `Validate` over
every fixture and checks the output. **When you change validation behavior, add or update a
fixture** — the fixtures, not the Go assertions, are where conformance is pinned. Fixture names
describe the case they lock in (e.g. `done-with-open-task`, `depends-on-cycle`,
`context-stub-blocks-feature`).

### Specs and versioning

`spec/<version>.md` files are normative. `spec/README.md` and the top-level `SPEC.md` index
them; the current version is **0.4**. `currentVersion` is defined in
`cli/internal/scaffold/scaffold.go`. A bundle vendors a copy of its pinned spec at its own
`docs/features/SPEC.md`, so bundles are self-describing. Bumping the spec means: add
`spec/<new>.md`, extend `supportedVersions` in `validate.go`, add a `migrate` path, update
`currentVersion`, add fixtures for the new rules, and refresh **skills + install primer +
README** so agents teach the new layout (re-run `fdf install` after users migrate).
