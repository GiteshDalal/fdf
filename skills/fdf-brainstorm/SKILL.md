---
name: fdf-brainstorm
description: Use when a feature idea has no Feature document yet, or a draft FDF feature still lacks its slug.spec.md — before any design or implementation work.
---

# FDF Brainstorm

Turn a feature idea into a validated FDF feature document and approved design.

New to FDF? The format is defined in the bundle itself at
`docs/features/SPEC.md` — exact frontmatter fields, casing and position
rules, and the F/R validation rules this skill cites. The fdf-help skill
explains how the fdf skills fit together.

**Read the Context docs first.** `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, and `INFRA.md` at the bundle root are the project's stack,
architecture, surface (interface) principles, and infrastructure. Ground the
design in them — propose an approach that fits the documented stack,
principles, and surface conventions, and flag it explicitly when a good
design would require departing from them (a new dependency, a new pattern, a
new surface convention). If they're still unfilled stubs, stop and run
fdf-init first.

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
   "Do you approve this design?" Nothing is written until yes.
6. **Write `<group>/<slug>.spec.md`** (`type: Spec`) with sections:
   `## What is being built`, `## Why`, `## Design decisions` (one bullet per
   resolved ambiguity), `## Alternatives rejected` (each with its reason).
   Path is a **stem sibling** of the feature file — not nested under
   `<group>/<slug>/`.
7. **Optional surface doc.** When the feature exposes non-trivial interface
   decisions that Gherkin cannot hold (API envelope for this feature, CLI
   flag set, UI choreography, event shapes, copy/a11y for this flow), after
   SPEC approval write `<group>/<slug>.surface.md` (`type: Surface`) with
   that rationale. A **surface** is any interface (UI, API, CLI, events) —
   not visual design only. Skip when SURFACES.md already covers it and the
   feature adds nothing feature-specific. Validation never requires
   `.surface.md` for a status.
8. **Self-review** before flipping: re-read the feature doc against the
   conversation. Any user decision that no scenario or SPEC line records?
   Any two scenarios whose names could be confused? Fix silently; don't
   re-ask.
9. **Flip status** to `specified`; update the feature's `timestamp`; add a
   bundle (or group) LOG.md entry, and optionally start `slug.log.md`
   (date heading + one line). **Gate**: `fdf validate` exit 0.

Next: the feature is `specified` — fdf-plan is the next skill.

## Rules

- Never skip the approval step; the SPEC records a human decision.
- No code, no scaffolding beyond `fdf new`, no implementation files until
  the design is approved — however simple the feature seems.
- One feature per brainstorm. If the idea spans independent subsystems,
  decompose it into features first, then brainstorm one.
- Write `slug.spec.md` (and optional `slug.surface.md`) as stem siblings —
  never `slug/SPEC.md` or other nested trail paths (those are the v0.3
  layout; v0.4 forbids trail files inside the task directory).
