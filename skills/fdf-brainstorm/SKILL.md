---
name: fdf-brainstorm
description: Use when starting a new feature in an FDF bundle — dialogue that produces a Feature document (Markdown + Gherkin) and its SPEC.md, moving status draft → specified.
---

# FDF Brainstorm

Turn a feature idea into a validated FDF feature document and approved design.

## Process

1. **Locate the bundle**: `fdf validate` (respects `--root`/`FDF_ROOT_DIR`).
   If no bundle exists, ask before running `fdf init`.
2. **Scaffold**: `fdf new <group>/<slug>` (lowercase). Read the generated file.
3. **Understand the feature** through questions, ONE at a time: who is the
   user, what capability, what value, what are the edge cases? Prefer
   multiple-choice questions.
4. **Write the Gherkin**: one `Feature:` fence (As-a / I-want / So-that), one
   fence per `Scenario:`. Scenarios are observable behavior, not
   implementation. Keep prose context between fences, link related docs.
5. **Present the design** for the feature (approach, alternatives, trade-offs)
   section by section; get explicit approval.
6. **Write `<group>/<slug>/SPEC.md`** (`type: Spec`): the approved design —
   what is being built, why, alternatives rejected.
7. **Flip status** to `specified` in the feature frontmatter; update the
   feature's `timestamp`; add a LOG.md entry.
8. **Gate**: run `fdf validate` — exit 0 required before you are done.

## Rules

- Never skip the approval step; the SPEC records a human decision.
- Scenario names are contracts — TEST.md will reference them verbatim later.
- One feature per brainstorm; split oversized ideas into multiple features.
