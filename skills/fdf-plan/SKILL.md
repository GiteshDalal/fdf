---
name: fdf-plan
description: Use when an FDF feature is specified (slug.spec.md approved) and has no implementation plan yet — before touching code.
---

# FDF Plan

Turn an approved `slug.spec.md` into an executable plan with provable acceptance.

New to FDF? The format is defined in the bundle itself at
`docs/features/SPEC.md` — exact frontmatter fields, casing and position
rules, and the F/R validation rules this skill cites. The fdf-help skill
explains how the fdf skills fit together.

Plan within the documented project context: `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, and `INFRA.md` at the bundle root tell you the real
technologies, code organization, surface conventions, and infrastructure —
task `resource:` paths, tools, and test commands must match them, not
invented ones. When the feature has an optional `slug.surface.md`, ground
interface choices (API shapes, CLI flags, UI flows) there and in SURFACES.md.

**Write for a zero-context implementer.** Whoever executes a task may be a
fresh agent that sees only that task file and `slug.spec.md` — no session
memory, no neighboring tasks. Every name, signature, path, endpoint, and
payload shape a task needs must be written in the task itself. "As discussed"
and "similar to task 01" are plan failures.

## Process

1. Read the feature doc and `<group>/<slug>.spec.md` (and
   `<group>/<slug>.surface.md` if present); run `fdf validate`.
   **Survey the project before writing anything**: find the real
   directories, existing patterns, test harness, and commands. `resource:`
   paths must exist (R1); invented paths are confidently wrong. Match
   surface constraints from SURFACES.md / optional `.surface.md`.
2. **SPEC gap → ask.** If a task needs a contract `slug.spec.md` doesn't
   define (a column schema, an API shape, a naming rule), STOP and ask the
   user; record the answer in `slug.spec.md` (or `slug.surface.md` for
   surface-only detail) — their answer is the approval for that edit. Never
   invent it mid-plan.
3. **Uncovered requirement → new scenario.** If the spec demands behavior no
   scenario names (a limit, an error path), add the Scenario to the feature
   document first — F8 only proves what scenarios name.
4. **Decompose** into tasks under the task directory only:
   `<group>/<slug>/01-slug.md`, `02-slug.md`, … (`type: Task`,
   `status: pending`). Each task:
   - `# Objective` — one sentence.
   - `# Steps` — executable without guessing: exact files, function
     signatures, status codes, payload shapes. Whatever one task produces
     and a sibling consumes is spelled out identically in BOTH files.
   - `# Acceptance` — names the scenarios (verbatim) it satisfies or
     contributes to, and how to check.
   - `resource:` and `depends-on:` are **frontmatter fields**, not body
     headings. See the task frontmatter rules below.
   - Right-sized: one sitting finishes one task; fold setup and scaffolding
     into the task whose deliverable needs them.

   **Task frontmatter — get these two right (they trip validation):**

   ```yaml
   ---
   type: Task
   status: pending
   resource: [src/count.rs, src/main.rs]   # OMIT for files that don't exist yet
   depends-on: [01-count-core, 02-cli]      # LIST for 2+; bare scalar for one
   ---
   ```

   - **`resource:`** lists *existing* project paths the task touches. R1
     fails the `planned` gate if a listed path doesn't exist yet — so on a
     **greenfield task that creates new files, omit `resource:` entirely**
     (it is optional). Only list paths that already exist at plan time; the
     `# Steps` still name the files the task will create.
   - **`depends-on:`** names sibling tasks by ID (filename minus `.md`),
     and the graph MUST be acyclic. One dependency is a bare scalar
     (`depends-on: 01-count-core`); **two or more MUST be a YAML list**
     (`depends-on: [01-count-core, 02-cli]`). A comma-joined string
     (`depends-on: 01-count-core, 02-cli`) is one bogus ID and fails F6.
     This graph drives parallel execution in fdf-execute.
5. **Write `<group>/<slug>.test.md`** (`type: Test`): `# Test Cases`, one
   entry per scenario, naming the scenario verbatim and giving the CONCRETE
   verification — the exact command, the test file/name to write, or a
   step-by-step manual procedure — **and what passing looks like** (expected
   status, output, resulting state). "Run the tests" proves nothing. If
   verification isn't obvious, STOP and ask how done-ness will be proven;
   record the answer. New APIs get an E2E/integration test; UI changes get a
   real-browser (Playwright) check.
6. **The final task always satisfies `slug.test.md`** — writing/running what
   it names. Every plan ends with it; it depends-on every other task.
7. **Write `<group>/<slug>.plan.md`** (`type: Plan`): `# Tasks` — ordered
   list linking every task file with **relative paths from the plan** (e.g.
   `instant-refunds/01-refund-api.md` → `slug/01-….md`). Plan order is the
   readable order; depends-on is execution truth.
8. Flip feature status to `planned`; LOG entry (bundle/group LOG.md and/or
   `slug.log.md`); `fdf validate` exit 0 (F8 enforces `slug.test.md`
   scenario coverage).

Next: the feature is `planned` — fdf-execute is the next skill.

## Rules

- No task without acceptance criteria; no scenario without a test case.
- The plan is done when a stranger could implement it. Re-read each task
  asking "what would I have to guess here?" — then put the answer in the file.
- Trail paths are stem siblings (`slug.plan.md`, `slug.test.md`); tasks live
  only under `slug/`. Never nest SPEC/PLAN/TEST inside the task directory.
