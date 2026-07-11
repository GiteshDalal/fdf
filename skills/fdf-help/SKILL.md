---
name: fdf-help
description: Use when starting any conversation in a project with an FDF bundle (docs/features/ or FDF_ROOT_DIR) — establishes how to route work to the fdf skills by feature status BEFORE writing any code, including "quick", "tiny", and "just do it" changes.
---

# Using FDF

## What is FDF

FDF (Feature Document Format) documents software features as a directory of
markdown files — the **bundle**, at `docs/features/` in this project. Each
feature is one Markdown + Gherkin file with a lifecycle `status` in its YAML
frontmatter; its approved design (`SPEC.md`), implementation plan
(`PLAN.md`), acceptance tests (`TEST.md`), tasks (`NN-slug.md`), and an
optional decision `LOG.md` live in a paired directory beside it. Three
bundle-root **Context documents** — `STACK.md`, `ARCHITECTURE.md`,
`INFRA.md` — hold the project's current stack, architecture, and
build/deployment infrastructure. The `fdf` CLI keeps it honest: `fdf validate`
must exit 0 after any bundle edit, `fdf new <group>/<slug>` scaffolds a
feature. You are not expected to know FDF — the complete format rules ship
inside the bundle at `docs/features/SPEC.md`; read that file for exact
frontmatter fields, casing and position rules, and the validation rules the
fdf skills cite by number (F1–F9, R1).

The bundle is the source of truth for what the software does. Code that
changes behavior without touching the bundle makes the bundle lie — that is
the failure FDF exists to prevent, and tiny changes are where it happens.

**Context documents are critical.** STACK/ARCHITECTURE/INFRA are the project's
living context; accurate, they let you build the project's way instead of
guessing — agentic engineering, not vibe coding. They are filled once by the
fdf-init interview and changed only with explicit user approval (the
post-feature step in fdf-execute), each change logged. Never edit them
casually. Until filled they carry a `<!-- fdf:stub -->` stub marker, and F9
blocks feature work while any stays a stub.

## The Rule

Before writing any code — and before answering "how should we build X" —
find the bundle state and route by **feature status**. Status is the dispatch
key: not the verb the user used, not the size of the change.

| Bundle state | Skill |
|---|---|
| Context docs missing or still `<!-- fdf:stub -->` | fdf-init (fill them first) |
| No feature file for this capability | fdf-brainstorm |
| `draft` | fdf-brainstorm (finish the spec) |
| `specified` | fdf-plan |
| `planned` or `implementing` | fdf-execute |
| `done` | Shipped; changing it is a new feature → fdf-brainstorm |

Check the Context docs first: if `fdf validate` warns or fails on unfilled
STACK/ARCHITECTURE/INFRA stubs, route to fdf-init before any feature work —
F9 will block it otherwise.

Then announce: "Using fdf-<skill> — <feature> is <status>."

A request may span stages: "get it implemented" on a `specified` feature
means fdf-plan **then** fdf-execute. Chain them in order; never enter a later
stage because the user named it.

**Does this change need a feature document?** Test: does it change what the
software does for a user (new capability, new flag, changed output, changed
behavior)? Yes → feature document. No — pure refactor, bugfix restoring
already-documented behavior, typo, tooling — work normally without one.

## "It's tiny, just do it"

Size never routes around FDF. The minimal compliant path is cheap: `fdf new
<group>/<slug>`, one `Feature:` fence, one `Scenario:` — minutes, and the
skills scale down to match. State that cost once. If the user still
explicitly opts out, their instruction wins — do the work, then say plainly
that the bundle now lacks this change. The violation is the *silent* skip,
and so is the shortcut of doing it first and asking later.

## Red Flags — STOP, you are rationalizing

| Thought | Reality |
|---|---|
| "It's a one-line change" | Size doesn't route; status does. One line that changes behavior gets a feature doc. |
| "The user said just do it" | Offer the two-minute path first. Only an explicit opt-out after that counts. |
| "I'll backfill the docs later" | The bundle lies the whole interim. Draft first, code second. |
| "The full pipeline is process theater" | The trail is what the next agent trusts. Scale it down, don't skip it. |
| "User said 'implement', so fdf-execute" | Verb ≠ status. Read the frontmatter, route by it. |
| "I'll flip statuses in a batch at the end" | Statuses reflect reality *now* — `in-progress` before working, per task. |
| "Skip validate just this once" | `fdf validate` exit 0 is the gate after every bundle edit. No exceptions. |

## Precedence

Explicit user instructions outrank skills. But "quick", "we're late", and
"don't waste tokens" are pressure, not opt-outs — the only opt-out is the
user declining the minimal path after you've named it.
