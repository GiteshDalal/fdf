---
name: fdf-brainstorm
description: Use when a feature idea has no Feature document yet, or a draft FDF feature still lacks its slug.spec.md — before any design or implementation work.
---

# FDF Brainstorm

Turn a feature idea into a validated FDF feature document and approved design.

New to FDF? Run `fdf spec` for the format rules (also vendored in the bundle
at `docs/features/SPEC.md`) and `fdf help` for the CLI. The fdf-help skill
explains how the fdf skills fit together.

**This skill is for capabilities that do not exist yet.** If the capability is
already delivered (`done` or `retired`) and needs to change or be fixed, stop
and use fdf-change — a second feature document for one capability is the drift
FDF exists to prevent. The one exception: a `retired` capability that is
coming back returns as a new feature, brainstormed here, which may name the
retired one in `depends-on`. The retired document is never flipped back.

**Read the Context docs first.** `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, and `DOMAIN.md` at the bundle root are the
project's stack, architecture, surface (interface) principles, infrastructure,
and domain language. Ground the design in them — propose an approach that fits
the documented stack, principles, and surface conventions, and flag it
explicitly when a good design would require departing from them (a new
dependency, a new pattern, a new surface convention). If they're still
unfilled stubs, stop and run fdf-init first.

**Write the Gherkin in the project's words.** `DOMAIN.md` gives one canonical
name per concept and lists the words banned in its place; use the canonical
ones in every scenario, and in the names you propose for anything new. F12
reports a banned word in a feature's Gherkin, but the point is upstream of the
warning: a scenario that calls a Venue a "store" has already started the drift.
If the capability needs a concept the lexicon does not have, say so — a new
term is a DOMAIN.md change and needs the user's approval like any other
Context edit.

The lexicon is the project's *internal* language. What a person reads on a
surface — a screen title, a button, a locale string — may say "store" for a
Venue on purpose, and a scenario does not change that. Write steps around the
concept, not the copy: `When the merchant opens the Venue settings`, not
`When I tap "Store settings"`. The step definition binds the concept to
whatever the label says today — and `slug.surface.md` records the wording
when it is a surface decision — so the scenario survives a reworded button or
a new translation, and a banned word never enters the Gherkin through quoted
copy. Where the exact wording is the thing under test (an error message, a
URL, a button label), name the outcome in the scenario — `Then the merchant
is told the Venue is closed` — and pin the literal string in `slug.test.md`,
where the case that checks it lives. Never change the label itself to clear
an F12 warning.

**Read the practices that will govern the code.** `practices/` holds the
project's binding answers for recurring mechanisms; a practice whose
`applies-to` covers the paths this feature will touch constrains the design
before you propose it. Designing against one is a legitimate thing to
propose — designing in ignorance of one is not.

## Process

1. **Locate the bundle**: `fdf validate` (respects `--root`/`FDF_ROOT_DIR`).
   If no bundle exists, ask before running `fdf init`.
2. **Scaffold**: `fdf new <group>/<slug>` (lowercase). Read the generated file
   (`<group>/<slug>.md`). A `draft` feature has no trail siblings and no task
   directory yet.
3. **Understand the feature** through questions, ONE at a time: who is the
   user, what capability, what value, what are the edge cases? Prefer
   multiple-choice questions. **Chase ambiguous words**: when the user says
   "newest", "duplicate", "fast", ask which meaning they intend (newest =
   later in the file, or by a timestamp column?). Every ambiguous term you
   resolve silently is a design decision nobody approved.
4. **Write the Gherkin**: one `Feature:` fence (As-a / I-want / So-that), one
   fence per `Scenario:`, prose context between fences, link related docs.
   - **Coverage**: one happy path, one scenario per edge case the dialogue
     surfaced, plus limits and access control where they exist. Every answer
     that changed the design shows up in some scenario or SPEC line.
   - **Observable, not implementation**: scenarios state what a user can see
     or a system can observe. `Then the existing contact shows the new phone
     number` — observable. `Then the service upserts by normalized email` —
     implementation; that sentence belongs in `slug.spec.md`.
   - **Names**: short, distinct, stable — `slug.test.md` will reference them
     verbatim later.
   - **Surface details stay out of Gherkin** when they are not user-
     observable outcomes (layout choreography, API envelope rationale, CLI
     flag set). Put those in the spec, optional `slug.surface.md`, or
     SURFACES.md — not in scenario steps.
5. **Present the design** section by section — approach, then alternatives
   with trade-offs (lead with your recommendation), then accepted
   trade-offs. Every decision you made yourself while writing the Gherkin
   (a term you defined, a rule extended to a case the dialogue never
   covered) gets its own bullet here — the gate covers your decisions too.
   Pause for approval after each section, and end with one explicit gate:
   "Do you approve this design?" The spec is not written until yes.
6. **Write `<group>/<slug>.spec.md`** (`type: Spec`) with sections:
   `## What is being built`, `## Why`, `## Design decisions` (one bullet per
   resolved ambiguity), `## Alternatives rejected` (each with its reason).
   Path is a **stem sibling** of the feature file — not nested under
   `<group>/<slug>/`. In the same edit, flip the feature's `status` to
   `specified`: a `draft` may carry no trail siblings (F4), so the spec and
   the flip land together.
7. **Optional surface doc.** When the feature exposes non-trivial interface
   decisions that Gherkin cannot hold (API envelope for this feature, CLI
   flag set, UI choreography, event shapes, copy/a11y for this flow), after
   SPEC approval write `<group>/<slug>.surface.md` (`type: Surface`) with
   that rationale. A **surface** is any interface (UI, API, CLI, events) —
   not visual design only. Skip when SURFACES.md already covers it and the
   feature adds nothing feature-specific. Validation never requires
   `.surface.md` for a status.
8. **Self-review** before validating: re-read the feature doc against the
   conversation. Any user decision that no scenario or SPEC line records?
   Any two scenarios whose names could be confused? Fix silently; don't
   re-ask.
9. **Log and gate.** Update the feature's `timestamp`; log the approval in
   the feature's `slug.log.md` (create it on first use: `type: Log`, a
   `## YYYY-MM-DD` heading, one line). Feature-scoped entries stay out of the
   root `LOG.md`, which is for bundle-wide events. **Gate**: `fdf validate`
   exit 0 — use fdf-validate if it fails.

Next: the feature is `specified` — fdf-plan is the next skill.

## Rules

- Never skip the approval step; the SPEC records a human decision.
- No code, no scaffolding beyond `fdf new`, no implementation files until
  the design is approved — however simple the feature seems.
- One feature per brainstorm. If the idea spans independent subsystems,
  decompose it into features first, then brainstorm one.
- A feature that builds on a delivered one may record that with `depends-on`
  (feature IDs, must exist, acyclic). Use it for lineage — not as a substitute
  for a Change when you are really altering the existing capability.
- Write `slug.spec.md` (and optional `slug.surface.md`) as stem siblings —
  never `slug/SPEC.md` or other nested trail paths (those are the v0.3
  layout; the task directory holds only tasks).
