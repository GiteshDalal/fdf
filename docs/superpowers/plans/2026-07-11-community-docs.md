# Community Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CODE_OF_CONDUCT.md (Contributor Covenant 2.1), a repo-specific CONTRIBUTING.md, and a README `## Why` section.

**Architecture:** Three root-level Markdown deliverables, one commit each, on the existing `worktree-install-script` branch (PR #2). No code changes; the only verification is the CI-gate run and visual/markdown sanity.

**Tech Stack:** Markdown (UTF-8), mermaid, GitHub community-profile conventions.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-11-community-docs-design.md`
- CoC is Contributor Covenant **v2.1 verbatim**; enforcement contact `giteshdalal@gmail.com`; no other edits.
- CONTRIBUTING is repo-specific, ~70 lines, lists the five CI gates verbatim and the conformance-fixture contract.
- README `## Why` sits between the Context-documents paragraph and `## Install`; closing phrase verbatim: **agentic engineering, not vibe coding**.
- All files UTF-8; root placement (not `.github/`).
- CI gates must stay green: `go vet ./...`, `go test ./...`, `test -z "$(gofmt -l .)"`, skills frontmatter lint, `shellcheck install.sh`.

---

### Task 1: CODE_OF_CONDUCT.md

**Files:**
- Create: `CODE_OF_CONDUCT.md`

**Interfaces:**
- Produces: root file linked from Task 2's CONTRIBUTING.md as `[Code of Conduct](CODE_OF_CONDUCT.md)`.

- [ ] **Step 1: Write the file** — Contributor Covenant 2.1 verbatim (from https://www.contributor-covenant.org/version/2/1/code_of_conduct/), with the enforcement contact set to `giteshdalal@gmail.com` in the Enforcement section. Standard structure: Our Pledge / Our Standards / Enforcement Responsibilities / Scope / Enforcement / Enforcement Guidelines (Correction, Warning, Temporary Ban, Permanent Ban) / Attribution (with the contributor-covenant.org links and the Mozilla ladder credit).

- [ ] **Step 2: Verify**

Run: `head -3 CODE_OF_CONDUCT.md && grep -c giteshdalal@gmail.com CODE_OF_CONDUCT.md`
Expected: `# Contributor Covenant Code of Conduct` header; grep count `1`.

- [ ] **Step 3: Commit**

```bash
git add CODE_OF_CONDUCT.md
git commit -m "docs: adopt Contributor Covenant 2.1"
```

---

### Task 2: CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

**Interfaces:**
- Consumes: `CODE_OF_CONDUCT.md` (Task 1) via relative link.

- [ ] **Step 1: Write the file**

```markdown
# Contributing to fdf

Thanks for your interest! This page covers everything a change needs to land.

## What this repo is

This is the source of the `fdf` CLI, its agent skills, and the versioned FDF
specs — **not** a project that uses FDF. The bundles under `testdata/` are
conformance fixtures, not real documentation. Bugs, ideas, and format
discussion all belong in [GitHub issues](https://github.com/GiteshDalal/fdf/issues).

## Development setup

You need Go 1.26+ (and optionally `shellcheck` for `install.sh`):

​```bash
go build ./cli/cmd/fdf     # build the CLI
go test ./...              # full test suite, including conformance fixtures
go run ./cli/cmd/fdf validate   # run without installing
​```

## Before you open a PR

CI runs these five gates on ubuntu and macos; run them locally first:

​```bash
go vet ./...
go test ./...
test -z "$(gofmt -l .)"        # gofmt -w . to fix
shellcheck install.sh
# skills lint: every skills/*/SKILL.md starts with --- and has name:/description:
​```

## The conformance contract

`testdata/*/` fixtures are the executable spec. **If your change alters
validation behavior, it must add or update a fixture** — a `bundle/` directory
plus an `expect.txt` of `exit:`/`contains:` assertions. The fixtures, not the
Go test assertions, are where conformance is pinned. Name fixtures after the
case they lock in (e.g. `done-with-open-task`).

## Changing the spec

`spec/<version>.md` files are normative and frozen once released. Format
changes start as an issue; a version bump touches the new spec file,
`supportedVersions` in the validator, a `migrate` path, `currentVersion`,
fixtures for the new rules, and the skills/primer — see CLAUDE.md for the
full checklist.

## Pull requests

​```mermaid
flowchart LR
    A[fork + branch] --> B[change + tests/fixtures]
    B --> C[run the five gates]
    C --> D[PR against main]
    D --> E[review + merge]
​```

Keep PRs focused; one logical change each. By participating you agree to the
[Code of Conduct](CODE_OF_CONDUCT.md). fdf is [MIT licensed](LICENSE) — your
contributions are too.
```

(The `​```` fences above are escaped for plan nesting — write real fences.)

- [ ] **Step 2: Verify**

Run: `grep -c '^## ' CONTRIBUTING.md && grep -n 'CODE_OF_CONDUCT.md\|mermaid' CONTRIBUTING.md`
Expected: 6 sections; hits for both the CoC link and the mermaid block.

- [ ] **Step 3: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add contributing guide"
```

---

### Task 3: README `## Why` section

**Files:**
- Modify: `README.md` (insert between the Context-documents paragraph ending "…not vibe coding." and `## Install`)

**Interfaces:**
- Consumes: nothing; standalone prose.

- [ ] **Step 1: Insert the section**

```markdown
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
```

- [ ] **Step 2: Verify placement and gates**

Run: `grep -n '^## ' README.md`
Expected order: `## Why`, `## Install`, `## Use`, `## Spec`.
Run: `go vet ./... && go test ./... > /dev/null && test -z "$(gofmt -l .)" && shellcheck install.sh && echo GATES PASS`
Expected: `GATES PASS`.

- [ ] **Step 3: Commit and push**

```bash
git add README.md
git commit -m "docs: add Why section to README"
git push
```
