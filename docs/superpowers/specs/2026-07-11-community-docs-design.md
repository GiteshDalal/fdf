# Community docs — CoC, CONTRIBUTING, README "Why"

**Date:** 2026-07-11
**Status:** approved

## Goal

Make the repo a well-formed open-source project: a recognized Code of
Conduct, a contributor guide specific to this codebase, and a README section
that states why FDF exists. Ships on the `worktree-install-script` branch
(PR #2).

## Deliverables

### 1. `CODE_OF_CONDUCT.md` (repo root)

Contributor Covenant **v2.1 verbatim** — the GitHub-recognized standard.
Enforcement contact: `giteshdalal@gmail.com` (already public in every commit's
author field, so no new exposure). No other customization.

### 2. `CONTRIBUTING.md` (repo root)

Short (~70 lines) and repo-specific, not boilerplate:

- What this repo is: the source of the `fdf` CLI, skills, and specs — not a
  project that uses FDF (`testdata/` bundles are conformance fixtures).
- Dev setup: Go 1.26; `go build ./cli/cmd/fdf`; `go test ./...`.
- The five CI gates, verbatim commands: `go vet ./...`, `go test ./...`,
  `test -z "$(gofmt -l .)"`, the skills frontmatter lint, and
  `shellcheck install.sh`.
- **Conformance contract:** changing validation behavior requires adding or
  updating a `testdata/` fixture — fixtures, not Go assertions, pin
  conformance.
- Spec changes: `spec/<version>.md` files are normative; format changes start
  as an issue, and version bumps follow the checklist in CLAUDE.md.
- PR flow: fork → branch → PR against `main`; CoC applies everywhere.
- One small mermaid flowchart of the contribution path (per repo owner's
  markdown conventions).

### 3. README `## Why` section

New section between the layout tree/Context paragraph and `## Install`.
Thesis (owner's take, elevated): agents made writing code cheap, so the
failure mode is unrecorded intent — vibe coding is code whose "why" lived in
a chat scrollback that's gone. FDF forces the opposite: intent, design, plan,
and proof live beside the code; statuses must reflect reality; the validator
turns drift into a build failure; human-approved Context docs keep agents
grounded. Closing phrasing kept verbatim: built for **agentic engineering,
not vibe coding**. The existing similar sentence in the Context paragraph
stays.

## Non-goals

- No issue/PR templates, no SECURITY.md (not requested; YAGNI).
- No README restructuring beyond inserting the one section.
- No `.github/` relocation — root placement for discoverability.

## Testing

- CI-equivalent run stays green (docs don't affect Go, but run the gates).
- Markdown is UTF-8; mermaid block renders on GitHub (syntax-check by eye).
- GitHub community profile (Insights → Community Standards) recognizes both
  files once merged — verify post-merge.
