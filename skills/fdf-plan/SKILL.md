---
name: fdf-plan
description: Use when a specified FDF feature needs an implementation plan — produces PLAN.md, NN-task files with depends-on, and TEST.md, moving status specified → planned.
---

# FDF Plan

Turn an approved SPEC.md into an executable plan with provable acceptance.

## Process

1. Read the feature doc and `<group>/<slug>/SPEC.md`. Run `fdf validate`.
2. **Decompose** into tasks `01-slug.md`, `02-slug.md`, … (`type: Task`,
   `status: pending`). Each task: `# Objective`, `# Steps` (concrete),
   `# Acceptance` (names the scenarios it satisfies). Set `resource:` to the
   project paths the task will touch. Set `depends-on:` to the sibling tasks
   it truly needs — this graph drives parallel execution and MUST be acyclic.
3. **Write TEST.md** (`type: Test`): `# Test Cases` with one entry per
   scenario, naming the scenario verbatim and giving the CONCRETE
   verification — a command to run, a test file to write, or an explicit
   manual procedure. **If the feature has no obvious verification, STOP and
   ask the user how done-ness will be proven; record the answer.** New APIs
   get an E2E/integration test; UI changes get a browser/Playwright check.
4. **The final task always satisfies TEST.md** — writing/running the tests it
   specifies. Every plan ends with it; it depends-on every other task.
5. **Write PLAN.md** (`type: Plan`): `# Tasks` — ordered list linking every
   task file. Plan order is the readable order; depends-on is execution truth.
6. Flip feature status to `planned`; LOG.md entry; `fdf validate` must exit 0
   (F8 enforces TEST.md scenario coverage).

## Rules

- No task without acceptance criteria; no scenario without a test case.
- Tasks are right-sized when one sitting finishes one task.
