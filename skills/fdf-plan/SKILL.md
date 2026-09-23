---
name: fdf-plan
description: Use when an FDF feature is specified (slug.spec.md approved) and has no implementation plan yet — before touching code.
---

# FDF Plan

Turn an approved `slug.spec.md` into an executable plan with provable acceptance.

New to FDF? The format is defined in the bundle itself at
`docs/features/SPEC.md` — exact frontmatter fields, casing and position
rules, and the F/R validation rules this skill cites. The fdf-help skill
explains how the fdf skills fit together. Run `fdf spec` for the format rules
and `fdf help` for the CLI.

This skill plans a **feature**. A post-delivery `Change` is planned the same
way — `changes/<id>.plan.md` with `# Tasks` linking every task under
`changes/<id>/` — but it is driven by fdf-change, and it has no `.test.md` of
its own: the test obligation belongs to the feature it affects.

Plan within the documented project context: `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, and `DOMAIN.md` at the bundle root tell you the
real technologies, code organization, surface conventions, infrastructure, and
vocabulary — task `resource:` paths, tools, and test commands must match them,
not invented ones. When the feature has a `slug.surface.md`, ground interface
choices (API shapes, CLI flags, UI flows) there and in SURFACES.md. A feature
that adds or changes something a person or another system uses directly, and
has no surface document yet, gets one before its tasks dictate the interface
(what goes in it: fdf-brainstorm, *Surface document*). A feature with no such
interface says `surface: none` in its frontmatter; `fdf validate` warns about
a planned feature that has neither.
Name things in tasks the way `DOMAIN.md` names them: the identifiers a task
dictates are the ones that end up in the code. The lexicon stops at the
surface, though — when a task dictates user-facing copy (a screen title, a
button, a locale string), that wording follows SURFACES.md and the feature's
`slug.surface.md`, and may legitimately differ from the term: a task can name
the `Venue` type in one line and title its screen "My store" in the next.

**Write for a zero-context implementer.** Whoever executes a task may be a
fresh agent that sees only that task file and `slug.spec.md` — no session
memory, no neighboring tasks. Every name, signature, path, endpoint, and
payload shape a task needs must be written in the task itself. "As discussed"
and "similar to task 01" are plan failures.

**When the work reaches into another feature.** A plan may add to something
another feature built — a page, an endpoint, a table — as long as every
scenario of that feature stays true. The addition is this feature's behavior:
its interface goes in this feature's `slug.surface.md` and its proof in this
feature's `slug.test.md`. When a scenario of a `done` feature would stop being
true, that is not a task. Stop and tell the user: that feature changes only
through a Change (fdf-change), and they decide whether it comes first.

## Process

1. Read the feature doc and `<group>/<slug>.spec.md` (and
   `<group>/<slug>.surface.md` if present); run `fdf validate`.
   **Survey the project before writing anything**: find the real
   directories, existing patterns, test harness, and commands. `resource:`
   paths must exist (R1); invented paths are confidently wrong. Match
   surface constraints from SURFACES.md / optional `.surface.md`.
2. **Route each task to the practices that govern it.** A practice whose
   `applies-to` covers a path the task touches — in its `resource:`, or where
   its `# Steps` create files — is binding on that task's code. Name it in the task's `# Steps` — "follow
   `/practices/permission-checks`" — because the implementer may be a fresh
   agent with no reason to go looking. If the plan requires departing from a
   practice, that is a decision for the user, not a detail for the task.
3. **SPEC gap → ask.** If a task needs a contract `slug.spec.md` doesn't
   define (a column schema, an API shape, a naming rule), STOP and ask the
   user; record the answer in `slug.spec.md` (or `slug.surface.md` for
   surface-only detail) — their answer is the approval for that edit. Never
   invent it mid-plan.
4. **Knowingly out of scope → file a debt, don't bury it.** If the plan
   deliberately leaves something undone — a migration the feature does not
   need yet, a call site left on the old mechanism — `fdf debt
   [<group>/]<slug>` records it with the paths that carry it. A deferral
   written into a task's `# Steps` as a TODO is invisible the moment that task
   is `done`.
5. **Uncovered requirement → new scenario.** If the spec demands behavior no
   scenario names (a limit, an error path), add the Scenario to the feature
   document first — F8 only proves what scenarios name.
6. **Decompose** into tasks under the task directory only:
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
   resource: [src/cli.rs, src/report/]     # a file it edits; the existing dir it adds files to
   depends-on: [01-count-core, 02-cli]      # LIST for 2+; bare scalar for one
   ---
   ```

   - **`resource:`** lists *existing* project paths the task touches. R1
     fails the `planned` gate if a listed path doesn't exist yet — so for a
     file the task will **create**, list the existing directory it goes into
     instead. R1 accepts a directory, and it keeps the task's scope and its
     practice routing working. Omit `resource:` only when not even the
     directory exists yet (a greenfield project); the `# Steps` always name
     the files the task creates.
   - **`depends-on:`** names sibling tasks by ID (filename minus `.md`),
     and the graph MUST be acyclic. One dependency is a bare scalar
     (`depends-on: 01-count-core`); **two or more MUST be a YAML list**
     (`depends-on: [01-count-core, 02-cli]`). A comma-joined string
     (`depends-on: 01-count-core, 02-cli`) is one bogus ID and fails F6.
     This graph drives parallel execution in fdf-execute.
7. **Write `<group>/<slug>.test.md`** (`type: Test`): a `# Test Cases`
   section with one case per scenario. Each case is a `## <scenario name>`
   heading, the name exactly as the Gherkin spells it (F8 matches it exactly,
   so a bullet or a table row naming the scenario does not count), followed by
   the CONCRETE verification — the exact command, the test file/name to write,
   or a step-by-step manual procedure — **and what passing looks like**
   (expected status, output, resulting state). "Run the tests" proves nothing. If
   verification isn't obvious, STOP and ask how done-ness will be proven;
   record the answer. New APIs get an E2E/integration test; UI changes get a
   check in a real browser, using the browser-test tool `STACK.md` or
   `INFRA.md` names (e.g. Playwright) — and if they name none, ask. When a
   scenario names an outcome whose exact wording matters — an error message,
   a URL, a button label — the literal string belongs in its case here: the
   Gherkin names the outcome, the test pins the words. The literal may use a
   word `DOMAIN.md` bans; the lexicon stops at the surface.
8. **The final task always satisfies `slug.test.md`** — writing/running what
   it names. Every plan ends with it; it depends-on every other task.
9. **Write `<group>/<slug>.plan.md`** (`type: Plan`): `# Tasks` — ordered
   list linking every task file with **relative paths from the plan** (e.g.
   `instant-refunds/01-refund-api.md` → `slug/01-….md`). Plan order is the
   readable order; depends-on is execution truth.
10. Flip feature status to `planned`, set its `timestamp` to now (in UTC,
    like every date and time in the bundle), and log it in the feature's own
    log, not the root `LOG.md`:
    `fdf log <group>/<slug> "**Planned**: <n> tasks; <what planning decided>."`
    Then `fdf validate` exit 0 —
    fdf-validate on failure (F8 enforces `slug.test.md` scenario coverage).
    Validate here, not between steps 6 and 9: tasks, test document and plan
    are only valid together — a task directory fails F6 until its plan
    exists.

## Hand off to fdf-execute

The feature is `planned`; fdf-execute is next. Planning filled this
conversation with exploration and dialogue that execution does not need —
the plan was written so that a reader with none of it can carry it out. So
recommend, in one line, that the user compact the conversation (`/compact`
in Claude Code) or start a fresh session, and give them a prompt to resume
with:

```text
Use the fdf-execute skill on <group>/<slug> (status: planned).
Plan: docs/features/<group>/<slug>.plan.md (<N> tasks). Batches from depends-on:
  1. 01-…, 02-…   2. 03-…   3. 04-… (satisfies <slug>.test.md)
Suggested models: 01, 02 mechanical → fast (e.g. Sonnet-class);
  03, 04 need judgment → most capable (e.g. Opus-class).
```

A task is **mechanical** when its `# Steps` leave nothing to decide, and needs
**judgment** when they leave design latitude, cross module boundaries, or
touch concurrency, security or a data migration; the task that proves
`slug.test.md` always needs judgment.

The prompt points at files and never carries a decision they lack. If you
are about to write a sentence of context the plan does not contain, the plan
is incomplete: put it in the task or the spec, then write the prompt. When
the user asked for the implementation too, ask whether to continue here or
from the prompt in a fresh context — do not simply stop.

## Rules

- No task without acceptance criteria; no scenario without a test case.
- The plan is done when a stranger could implement it. Re-read each task
  asking "what would I have to guess here?" — then put the answer in the file.
- Trail paths are stem siblings (`slug.plan.md`, `slug.test.md`); tasks live
  only under `slug/`. Never nest SPEC/PLAN/TEST inside the task directory.
