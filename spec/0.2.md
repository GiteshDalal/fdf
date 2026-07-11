# Feature Document Format (FDF) — v0.2

FDF is a **self-contained, opinionated format** for documenting software
features and their implementation trail as a directory of markdown files with
YAML frontmatter. A feature is written in Markdown + Gherkin; its design spec,
implementation plan, acceptance tests, and tasks live beside it. There is no
schema registry and no required tooling: if you can `cat` a file you can read
it. The reference tool is the `fdf` CLI in this repository.

Base rules: every non-reserved `.md` file is a **document** with YAML
frontmatter and a required `type`; a document's **ID** is its path minus
`.md`; `INDEX.md` and `LOG.md` are reserved filenames; consumers tolerate
unknown types, extra keys, and broken links rather than rejecting a bundle.

# Structure

- A **bundle** is a directory tree of markdown files rooted at the project's
  bundle root (see *Location*).
- A **group** is a plain directory with an `INDEX.md` listing (e.g. `wdise/`).
  Groups are not documents and carry no type.
- A **feature** `<group>/<slug>.md` owns the **paired directory**
  `<group>/<slug>/`, holding its trail: `SPEC.md`, `PLAN.md`, `TEST.md`, and
  ordered `NN-slug.md` tasks. **Position is the link** — no frontmatter
  pointers. The paired directory may contain nothing else. A `draft` feature
  has no paired directory at all.
- Releases live in `releases/<version>.md`; the release ID is the version.
- `README.md` at the bundle root is legal and ignored (git-forge landing page).

# Location

The bundle lives at **`docs/features/`** by default — the recommended
location. Tools MUST honor overriding it via the `FDF_ROOT_DIR` environment
variable or a `--root` flag (precedence: flag → env → default); relative
values resolve against the project root, absolute values are used as-is.

The mount may be a plain directory **or a git submodule at the same path**
(a wiki-style separate docs repo). Conforming tools MUST treat both
identically: `resource:` paths resolve against the **project** root — when
the bundle is a submodule (its root has a `.git` *file*), tools walk up to the
superproject working tree. A bundle checked out standalone gets format
validation with repo-integrity checks skipped and a warning, never a false
failure.

# Casing

Case marks reserved-ness. Uppercase fixed names: `INDEX.md`, `LOG.md`
(reserved machinery) and `SPEC.md`, `PLAN.md`, `TEST.md` (typed trail
documents). Everything else is lowercase and enforced: groups and slugs
`[a-z0-9-]`, tasks `NN-[a-z0-9-]+.md`, releases `<version>.md`. Any other
uppercase filename is an error.

# Types

| Type        | What it is                                        | Where it lives              |
|-------------|---------------------------------------------------|-----------------------------|
| `Reference` | Meta documents                                    | bundle root                 |
| `Feature`   | One capability, Markdown + Gherkin                | `<group>/<slug>.md`         |
| `Spec`      | Approved design for that feature                  | `<group>/<slug>/SPEC.md`    |
| `Plan`      | Implementation plan; body links every task        | `<group>/<slug>/PLAN.md`    |
| `Test`      | How the feature's scenarios are proven            | `<group>/<slug>/TEST.md`    |
| `Task`      | One implementable unit of the plan                | `<group>/<slug>/NN-slug.md` |
| `Release`   | A planned or shipped release referencing features | `releases/<version>.md`     |

A document's `type` must match its position; validators reject, say, a
`Feature` file at the bundle root or inside a paired directory.

# Frontmatter

Shared fields on every document: `type` (**required**), `title`,
`description`, `tags`, `timestamp`. Per-type fields:

| Field        | On      | Meaning                                                                 |
|--------------|---------|-------------------------------------------------------------------------|
| `status`     | Feature | **Required.** `draft → specified → planned → implementing → done`       |
| `status`     | Task    | **Required.** `pending → in-progress → done`                            |
| `status`     | Release | **Required.** `planned → shipped`                                       |
| `version`    | Feature | Optional — the release this feature targets or shipped in               |
| `date`       | Release | Target date while `planned`, actual date once `shipped`                 |
| `resource`   | Task    | Optional — project-relative path(s) to the code the task touches; string or list, each must exist |
| `depends-on` | Task    | Optional — sibling task IDs (filename minus `.md`) that must complete first; the graph MUST be acyclic |

`Spec` and `Plan` carry only shared fields. The bundle root `INDEX.md` pins
the spec: frontmatter `fdf_version: "0.2"` plus a link to this document.

# Lifecycle

```mermaid
stateDiagram-v2
    direction LR
    [*] --> draft
    draft --> specified : SPEC.md written
    specified --> planned : PLAN.md + TEST.md written
    planned --> implementing : first task started
    implementing --> done : all tasks done
    done --> [*]
```

| Feature status | Invariant                                              |
|----------------|--------------------------------------------------------|
| `draft`        | Feature file only; **no paired directory**             |
| `specified`    | `SPEC.md` exists                                       |
| `planned`      | `PLAN.md` and `TEST.md` exist (and `SPEC.md`)          |
| `implementing` | Plan, ≥ 1 task, not all tasks `done`                   |
| `done`         | Spec, plan, TEST.md, ≥ 1 task, all tasks `done`        |

Feature `status` tracks implementation; release `status` tracks shipping. A
release moves to `shipped` only when every listed feature is `done`.

# Body conventions

## Feature documents (Markdown + Gherkin)

Gherkin lives in ```` ```gherkin ```` fences under conventional headings, with
markdown prose between fences carrying what Gherkin can't:

- `# Feature` — exactly one fence with the `Feature:` block (name +
  As-a / I-want / So-that), then prose: context, links to related
  documentation, links to its own trail once it exists.
- `# Scenarios` — one fence per `Scenario:` / `Scenario Outline:`.
- `# Citations` — optional.

A Feature document contains exactly one `Feature:` declaration and at least
one `Scenario:` across its fences; concatenating the fences in order yields
one valid `.feature` file.

## Trail documents

- **SPEC.md** — the approved design: what is being built and why, alternatives
  considered.
- **PLAN.md** — a `# Tasks` heading with an ordered list linking every task
  file in the paired directory; the authoritative task ordering. The final
  task always satisfies TEST.md.
- **TEST.md** — `# Test Cases`: one entry per scenario, naming the scenario
  **verbatim** and giving the concrete verification — a command, a test-file
  path, or an explicit manual procedure. Optional `# Setup`. Every scenario
  in the feature document MUST have a test case here from `planned` onward.
- **Tasks** — `# Objective`, `# Steps`, `# Acceptance`; acceptance references
  the scenario names it satisfies. `depends-on` expresses execution ordering
  beyond `NN-` numbering; serial consumers follow plan order, parallel
  consumers schedule topologically.

## Release documents

`# Features` — links to every feature in the release; `# Notes` — optional.
A feature's `version: "X"` and `releases/X.md`'s list are two ends of one
relationship, checked in both directions. `releases/INDEX.md` lists newest
first. Releases are optional.

## Cross-linking

Bundle-relative `/…` links within the bundle (a feature's paired trail
conventionally uses relative links instead); relative paths for out-of-bundle
targets. Cross-linking a feature with the documentation of the code that
implements it is encouraged in both directions.

# Conformance

A **conforming bundle** satisfies all hard rules; a **conforming validator**
enforces them with these severities and reports the soft checks as warnings,
never errors:

| Rule | Hard requirement |
|------|------------------|
| F1 | Parseable YAML frontmatter with non-empty `type`; ISO-8601 `## YYYY-MM-DD` log headings; root `fdf_version` pin, when present, matches a spec version the tool supports (an absent pin is a warning) |
| F2 | `Feature`/`Task`/`Release` carry a valid `status` from the closed vocabularies |
| F3 | Positional integrity: paired directories contain only `SPEC.md`/`PLAN.md`/`TEST.md`/`NN-slug.md`; every paired directory has its sibling feature; `type` matches position; casing rules; nothing nests deeper |
| F4 | Status ↔ artifact invariants (table above) |
| F5 | Feature bodies: exactly one `Feature:`, ≥ 1 `Scenario:`, fences start with Gherkin keywords |
| F6 | `# Tasks` links exactly the task files present; `depends-on` names existing siblings; the dependency graph is acyclic |
| F7 | Release ↔ `version` bidirectional consistency; `shipped` lists only `done` features |
| F8 | From `planned` onward TEST.md exists (`type: Test`) and covers every scenario name |
| R1 | Every Task `resource` path exists in the project (skipped with a warning for standalone bundles) |

Soft: broken cross-links, missing recommended fields, index-listing presence,
log newest-first ordering.

# Versioning

Spec versions are `MAJOR.MINOR`. Bundles pin their version in the root
`INDEX.md`; tools state which versions they validate and offer mechanical
migration between adjacent versions (`fdf migrate`). This document is v0.2.

# Extending a bundle

1. `fdf new <group>/<slug>` — a `draft` feature; write the Gherkin.
2. Advance: `SPEC.md` (→ `specified`), then `PLAN.md` + `TEST.md`
   (→ `planned`), then tasks (→ `implementing`, → `done`). Keep `# Tasks`
   in lockstep; the final task satisfies TEST.md.
3. Link from the group `INDEX.md`; note changes in the nearest `LOG.md`.
4. `fdf validate` before committing.

# Acknowledgements

FDF's bundle conventions were inspired by
[OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md);
FDF is self-contained and not a profile of it. Gherkin keyword semantics:
[Gherkin reference](https://cucumber.io/docs/gherkin/reference/).
