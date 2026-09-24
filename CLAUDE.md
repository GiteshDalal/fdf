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
optional `slug.surface.md` / `slug.log.md`); tasks live only under a `slug/` directory. Five
bundle-root Context docs (`STACK.md`, `ARCHITECTURE.md`, `SURFACES.md`, `INFRA.md`,
`DOMAIN.md`) hold project context. Post-delivery work lives under `changes/` as a `Change` (alters what a
delivered feature does) or a `Fix` (the code drifted from what the document already says).
Known gaps are `Debt` under `debts/`; known defects not repaired yet are `Bug` under
`bugs/` (v0.7). A capability that predates the bundle is an `adopted` feature (v0.7): no
build trail, its code named in `resource`.
This repo ships:

1. A Go CLI (`cli/cmd/fdf`) that scaffolds and **validates** those bundles.
2. Harness-neutral **skills** (`skills/`) that teach AI agents the brainstorm → plan →
   execute workflow and the post-delivery `fdf-change` workflow, plus `fdf-help`
   (routing), `fdf-adopt` (mapping a codebase that predates its bundle, in
   phases), `fdf-debug` (root-cause-first triage that routes a defect to a Fix,
   Change, feature, task or adoption — or files a Bug), `fdf-checkpoint` (the
   periodic audit of the Context docs, the vendored `SPEC.md` and the agent
   instruction files) and `fdf-validate` (the post-edit gate).
   Skills are the *only* agent-facing surface — there are deliberately no slash commands
   or per-harness adapters, since a command is user-typed and cannot be a reliable gate.
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
`validate|init|new|practice|debt|bug|adopt|mv|lexicon|log|install|serve|migrate|spec|help|change|fix|history|release|version`).
Each command is a thin wrapper around one
`cli/internal/` package. Commands write usage/errors to stdout (not stderr) so tests capture
everything, and flags must precede positional args (`ContinueOnError` FlagSets): every command
parses through `parseArgs` (main.go), which refuses a flag after an argument and prints the
command as it should have been typed. `helpTopics` in `help.go` is the single source for each
command's usage line, its group and summary in the overview, `fdf help <command>` and `-h`;
`help_test.go` checks that every dispatcher command has one topic and every flag it defines
is documented. A command that works on a bundle stops with `fdfroot.NoBundle` when the root
holds no `INDEX.md`.

- **`cli/internal/fdfroot`** — root resolution, used by every command. Bundle root precedence:
  `--root` flag > `FDF_ROOT_DIR` env > `docs/features`. Relative roots resolve against the
  **project root**, found by walking up to the topmost `.git`. A `.git` *file* (submodule)
  marks a boundary but the walk continues to the superproject — so `resource:` paths always
  verify against the real project root even when the bundle is a git submodule. A `.git`
  file whose gitdir holds `commondir` is a **linked worktree** and ends the walk there:
  the worktree is its own checkout, even inside the main repo's directory.

- **`cli/internal/specver`** — spec versions: `Parse` reads `MAJOR.MINOR` (a bundle's
  `fdf_version`, a `spec/<version>.md` name), and versions compare by number, so 1.0
  follows 0.7 and 1.10 follows 1.2. Every version gate (`pinAtLeast` in `bundle` and
  `scaffold`) and `scaffold.SpecVersions` go through it.

- **`cli/internal/links`** — the one link-repair engine. `Find` returns a Markdown text's
  link targets as CommonMark reads them (inline links and images, with any title or `<…>`
  destination, and reference definitions; a footnote is not a link, and a target in a
  fenced or indented code block or a code span is marked as a sample), `Code` returns the
  byte ranges `Find` reads as code (`[]Span`), `Resolve` reads one target as a path, and
  `Retarget` recomputes one after a `Move`: a relative link changes whenever its file or
  its target moves, wherever the target is, inside the bundle or outside it. A link
  written from the bundle root changes only when its target moves. `fdf mv` uses it, and
  so does the validator under a 1.0 pin (`sectionTargets` also skips a heading line that
  starts in `Code`).

- **`cli/internal/layout`** — the one source of 1.0 positions: the closed root (its own
  files, the five Context documents and the six `Registers`), groups nested to any depth in
  every register but `releases/`, and the directory rule (a directory beside a feature,
  Change or Fix is its task directory; beside a practice, debt or bug it is an error; any
  other is a group). `Bundle.File` gives a Markdown file's `Position` (`Kind`, `Register`,
  the owner's `ID`, a trail's `Role`), or a `Stray` with the path at fault (`Where`) and its
  `Problem`; `Bundle.Dir` does the same for a directory. It reads names through an `fs.FS`
  and matches them exactly, so a case-insensitive disk cannot change a result.

- **`cli/internal/bundle`** (`validate.go`) — the heart of the tool. `Validate()` is the
  enforcement engine for the spec. Rules are coded **F1–F14** (format conformance) and **R1**
  (repo integrity); every error message ends with its rule code, e.g. `(F4)`. Highlights:
  - `readPin()` reads `fdf_version` from the root `INDEX.md` *before* the directory walk,
    because version-gated rules (Context docs, stem trail layout) must be known for every
    file and `WalkDir` visits lexically (`ARCHITECTURE.md` sorts before `INDEX.md`).
  - Validates spec **v0.2 – v0.7** and **v1.0** (`supportedVersions`); a pin outside that
    set is an F1 error pointing at `fdf migrate`, or at a newer fdf when it is newer than
    every supported version (`unsupportedPin`). Gates are `pinAtLeast(pin, specver.Version)`:
    `specStem` the stem-trail layout (v0.4 onward); `specV5` `changes/`, the `Change`/`Fix`
    types, `retired`, feature `depends-on`, and F10; `specV6` `practices/`, the `Practice`
    and `Debt` types, `DOMAIN.md`, F11, F12 and F13; `specV7` `bugs/` and F14, `resolves`,
    the `adopted` status and feature `resource`, F12's full reach, and F1's check of a
    `timestamp`'s form (a date, or an RFC 3339 time with `Z` or an offset). Every gate means
    "this version **and later**", so a 1.0 pin passes them all: 1.0 keeps the rules 0.7
    has. v0.2/v0.3 keep the nested paired-directory layout.
  - **`v1`** (a 1.0 pin) swaps the walk. `validate.go`'s `visitV0` reads a position from
    the path's segments, as 0.x lays a bundle out; **`walk.go`**'s `walkV1` asks `layout`,
    and reports a path with no position once, as F3 with `layout`'s `Problem`. Both record
    each document through **`collect.go`**'s `collection`: `document` for the checks every
    document takes, then `feature`, `trail`, `task`, `change`, `practice`, `entry` (a debt
    or bug), `registerLog`, `release` or `root`. Under `v1` the validator also reads links
    through `links.Find` and `links.Resolve` (`resolveLink`, `sectionLinks`, and the
    broken-link check, which then reads reference definitions too), F10 and F14 suggest the
    full ID of a feature named the 0.7 way (`featureHint`), and F12 reads group names
    through `layout` (`groupName`: at every depth, never a register's own name).
  - **`practices.go`** holds F11 (practice body shape, `superseded-by` graph) and the
    `sectionText` helper that `domain.go`, `debts.go` and `bugs.go` also use.
    **`bugs.go`** holds F14 and the `resolves` half of F10: `# Symptom`/`# Expected`
    required, `# Violates` parsed with the regression-case grammar and checked verbatim
    while open, forbidden on `accepted`, and a resolved bug still citing it must be named
    by a done Fix/Change's `resolves`. A cleared bug's ID is read back from
    `bugs/LOG.md` (`* **bugs/<id>** — …`). **`testcases.go`** is v0.7's F8: a case is a
    `## <scenario name>` heading under `# Test Cases`, matched exactly (`TestCases`, also
    used by `fdf adopt`'s map); a case naming no scenario warns. It also holds the surface
    check: `surface: none` or a `slug.surface.md` from `specified` on (adopted: from the
    first scenario), a warning otherwise. Debts and bugs share one position branch in
    `validate.go`. **`adopted.go`** is F4/F8 for `adopted` features: no spec, plan or
    task directory, `resource` required, no `version`, `slug.test.md` from the first
    scenario (F5 lets a map entry have none).
    **`debts.go`** holds F13: `# Gap` required, no Gherkin, `# Rationale` on
    `accepted`, `# Resolution` on `resolved`. Debt and practice positions are
    near-identical (`<slug>.md` plus an optional `<slug>.log.md`, one level of
    groups, never a task directory) and share `logTrailRoleRe`. **`domain.go`** holds F12: parsing
    `DOMAIN.md`'s `# Terms` grammar (`readLexicon`: consistency, `except:` and `strict`
    checks) and the **v0.6** scan — a warning unless `Options.StrictDomain` (the
    `--strict-domain` flag) promotes it to an error. **`lexicon.go`** is the **v0.7**
    scanner, exported for `fdf lexicon` and `fdf migrate`: `LoadLexicon`, `ScanBundle`,
    `ScanDocument`, `DomainScanned`. It masks quoting text byte for byte (so every
    `Occurrence` keeps file:line:col), scans names, matches through a `phraseSet`
    (candidate words looked up by leading token rather than one large alternation over
    every byte), and prints one line per document, capped at `domainReportCap`.
    `strict: true` in DOMAIN.md also promotes. Validate hands the scanner the texts it
    already read. The scan covers every feature's Gherkin and the declared
    scenario names (`add:`/`modify:`/`remove:`/regression cases) of every Change and Fix, at
    any status — the 0.6 erratum lets a *lexicon fix* (banned word → term) edit any document,
    frozen ones included, so every finding is clearable. It never scans a `## <feature-id>`
    heading or a verification: they quote an ID, a path or surface wording as it stands.
  - **`changes.go`** holds the v0.5 logic: parsing a change's declared effects
    (`# Scenario changes` with `add:`/`modify:`/`remove:`, or `# Regression cases`), F10,
    change F4, and the release↔change half of F7. On v0.6, a name one Change both removes
    and adds is *replaced* (`removals`): a lexicon fix can turn a finished rename into that. A directory under `changes/` is a task
    directory when a sibling `<name>.md` exists and a group otherwise.
  - Feature statuses `draft → specified → planned → implementing → done` drive the F4/F8
    status↔artifact invariants (e.g. `planned` requires `slug.test.md` with a case per
    Gherkin scenario; `done` requires all tasks done). v0.5 adds terminal `retired`, which
    is exempt from F8 and must be named by exactly one `done` Change carrying a
    `# Rationale`. `Options.FreshStubsAdvisory`
    downgrades F9 (unfilled Context stub) from error to warning — only `fdf migrate` sets it.

- **`cli/internal/scaffold`** (`init`, `new`, and `Adopt`, which scaffolds an adopted map
  entry; `pin.go` mirrors the validator's gates for the commands: `ReservedDirs` is which
  bundle-root directories the pin reserves, and `RequirePin` refuses to write a document
  type the pin predates, pointing at `fdf migrate`; `listing.go` keeps the indexes:
  `ListEntry`/`Unlist` for a document, `ListGroup` for a new group in its parent's index,
  the root `INDEX.md` for a feature group), **`cli/internal/install`** (skills + a
  `## Feature Document Format` primer, idempotent, never clobbers user edits; each
  skill's `.fdf-version` reads `<version> skills=<digest> primer=<digest> root=<root>`, so
  another build of the same version upgrades, and a primer an earlier install recorded
  counts as fdf's own; also removes the superseded slash commands by exact name), and **`cli/internal/migrate`** (mechanical upgrades to the current spec
  version; 0.3→0.4 rewrites nested trail files to stem siblings and scaffolds
  `SURFACES.md`; 0.4→0.5, 0.5→0.6 and 0.6→0.7 are **additive**, so the pre-0.4 layout
  transform — pre-flight included — is skipped for bundles already in the stem layout.
  0.5→0.6 moves nothing but adds a fifth Context stub, so F9 holds the bundle to filling
  `DOMAIN.md` before its next feature; migrate itself passes via `FreshStubsAdvisory`.
  0.6→0.7 scaffolds `bugs/INDEX.md`, refuses a bundle whose `bugs/` is a feature group,
  drops the status tags older tools wrote after index listings, logs the migration in
  the root `LOG.md`, and reports the new F12, F8, surface and timestamp counts and the
  debts that may be bugs).
  **`cli/internal/register`** (`fdf debt` and `fdf bug`: `register.Debt` and
  `register.Bug` are two `Kind`s over one implementation — list with optional status
  filter, scaffold, and `--cleanup`, which folds resolved entries into `<dir>/LOG.md`
  newest-first, with the first *paragraph* of `# Resolution`, and removes their files,
  listings and any group left empty; open and accepted entries are never touched, `--dry-run` previews, `--no-log` skips
  the log). **`cli/internal/changes`** (`fdf change`/`fdf fix` scaffolding, with
  `--from bugs/<id>` copying a bug's analysis and writing `resolves`, and `fdf history`,
  which computes a feature's post-delivery trail and known bugs from `affects:` rather
  than from back-links) and **`cli/internal/release`** (`fdf release`, which derives a
  release's `# Features`/`# Changes` lists from `version:` fields and preserves
  hand-written `# Notes`; it never derives membership). **`cli/internal/adopt`** is the
  adoption map (`fdf adopt` without an ID: features with scenario and test counts, then
  `git ls-files` code no `resource` claims). **`cli/internal/refactor`** holds the two
  maintenance edits a tool does whole: `Move` (`fdf mv` — plan every file move, rewrite
  links, edges, declaration headings and ID mentions in every document, move index
  listings, log, and report references outside the bundle) and `Lexicon`
  (`fdf lexicon`, and `--fix` one `--term` at a time: plurals, capitals and a/an kept,
  scenario names renamed across their joins; italic mentions, quoted Gherkin labels,
  table cells and names are left for a person). **`cli/internal/logs`** is `fdf log`
  and the one newest-first `Insert` every log writer shares (`mv`, `lexicon`, the
  registers' `--cleanup`): an entry goes in the log of the document it is about —
  a feature's, change's, practice's, debt's or bug's `<slug>.log.md`, created on
  first use (beside a `draft` only from v0.7, where a log is the one sibling a
  draft may have), a group's `LOG.md`, or the root `LOG.md` for the bundle as a
  whole.

### Layout the validator expects (v0.7)

```
docs/features/
├── STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md, DOMAIN.md   # type: Context
└── group/
    ├── slug.md                    # Feature
    ├── slug.spec.md / .plan.md / .test.md
    ├── slug.surface.md / .log.md  # optional
    └── slug/                      # tasks only: NN-….md
changes/                          # post-delivery, flat or one level of groups
├── INDEX.md
├── slug.md                       # type: Change | Fix
├── slug.spec.md / .plan.md / .log.md
└── slug/                         # tasks only
practices/                        # v0.6: flat or one level of groups
├── INDEX.md
├── slug.md                       # type: Practice (status: active | superseded)
└── slug.log.md                   # the ONLY legal sibling; no tasks, no spec/plan/test
debts/                            # v0.6: same shape as practices/
├── INDEX.md
├── LOG.md                        # where `fdf debt --cleanup` retires resolved debts
├── slug.md                       # type: Debt (status: open | accepted | resolved)
└── slug.log.md                   # the ONLY legal sibling
bugs/                             # v0.7: same shape as debts/
├── INDEX.md
├── LOG.md                        # `fdf bug --cleanup`; F10 reads cleared IDs here
├── slug.md                       # type: Bug (open | accepted | resolved)
└── slug.log.md
```

Trail roles are only `spec`, `plan`, `test`, `surface`, `log` under a group — only
`spec`, `plan`, `log` under `changes/`, since `test` and `surface` are living documents
owned by the affected feature — and only `log` under `practices/` and `debts/`, since both
are living references with no episodic trail at all — and bugs/ follows debts/. An `adopted`
feature (v0.7) owns only `slug.test.md`/`slug.surface.md`/`slug.log.md`, never a spec, plan
or task directory. Task dirs must not contain nested SPEC/PLAN/TEST or LOG.md.

Under a **1.0** pin (`spec/1.0.md`) the same documents live in six registers at the root:
feature groups move under `features/`, where a feature may also be filed flat; groups
nest to any depth in every register but `releases/`; a directory beside a practice, debt
or bug is an error; and the root holds no other Markdown. `layout` is the one place
those rules are written.

### The conformance contract

`testdata/*/` fixtures are the **executable spec**. Each fixture is a directory with a
`bundle/` (and optionally a `repo/` wrapper for R1 tests) plus an `expect.txt` of `exit:`,
`contains:` and `not-contains:` assertions; a `flags: --strict-domain` line runs the fixture as
`fdf validate --strict-domain` would. `TestConformanceFixtures` (`conformance_test.go`) runs
`Validate` over every fixture and checks the output. **When you change validation behavior, add or update a
fixture** — the fixtures, not the Go assertions, are where conformance is pinned. Fixture names
describe the case they lock in (e.g. `done-with-open-task`, `depends-on-cycle`,
`context-stub-blocks-feature`).

### Specs and versioning

`spec/<version>.md` files are normative. `spec/README.md` and the top-level `SPEC.md` index
them; the current version is **0.7**. `spec/1.0.md` is embedded and the validator checks
1.0 pins, but `currentVersion` stays 0.7, and the indexes leave 1.0 out, until the
commands write 1.0 bundles. `currentVersion` is defined in
`cli/internal/scaffold/scaffold.go`. A bundle vendors a copy of its pinned spec at its own
`docs/features/SPEC.md`, so bundles are self-describing. Bumping the spec means: add
`spec/<new>.md`, extend `supportedVersions` in `validate.go`, add a `migrate` path, update
`currentVersion`, add fixtures for the new rules, and refresh **skills + install primer +
README** so agents teach the new layout (re-run `fdf install` after users migrate). The
primer's superseded text must be appended to `legacyPrimers` so an upgrade can recognize and
replace an untouched managed section, and the new skill must be added to `skillNames`.
Prefer referencing `currentVersion` over a version literal in tests — the 0.5 bump had to
de-hardcode a dozen of them.
