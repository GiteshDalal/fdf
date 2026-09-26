---
name: fdf-plan
description: Use when an FDF feature is specified (its slug.spec.md is approved) and has no implementation plan yet — before touching code — including when its approved spec turns out to leave a case undecided.
---

# FDF Plan

Turn an approved `slug.spec.md` into an executable plan with provable
acceptance: the tasks, the test document and the plan that links them.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

This skill plans a **feature**. A post-delivery `Change` or `Fix` that needs
tasks is planned the same way — `<change-id>.plan.md` with `# Tasks` linking
every task under `<change-id>/` — but it is driven by fdf-change, and it has
no `.test.md` of its own: its test obligation belongs to the features it
affects.

Below, `<feature-id>` is the feature's full ID, such as
`features/payments/instant-refunds`, and every path is relative to the bundle
root, `docs/fdf/`: the feature is `<feature-id>.md`, and this skill writes
`<feature-id>.test.md`, `<feature-id>.plan.md` and the tasks in
`<feature-id>/`. There is no `fdf plan` command: you write these documents by
hand.

## Mechanics

- A `timestamp:` you set is the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run
  at the moment of the edit — never a time you estimate, round or reuse, and
  never your local time with `Z` added. A restored document keeps its old
  one; a malformed one already committed is repaired by fdf-validate's F1,
  never set to now.
- Log with single quotes — `fdf log <feature-id> '**Planned**: …'` — because
  inside double quotes the shell runs anything between backticks. An
  apostrophe would end the quotes: write ’ instead.
- In frontmatter, a `#` after a value is part of the value, not a comment:
  never put a comment on a field's line.

## Ground the plan in the project

Plan within the documented context. `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md` and `DOMAIN.md` at the bundle root give the real
technologies, code layout, surface conventions, infrastructure and vocabulary.
Task paths, tools and test commands must match them, never invented ones.

- **Names.** The identifiers a task dictates end up in the code, so name them
  the way `DOMAIN.md` does. The lexicon stops at the surface: when a task
  dictates user-facing copy (a screen title, a button, a locale string), that
  wording follows SURFACES.md and the feature's `slug.surface.md`, and may
  differ from the term — a task can name the `Venue` type in one line and
  title its screen "My store" in the next.
- **Interfaces.** When the feature has a `slug.surface.md`, take interface
  details (API shapes, CLI flags, UI flows) from it and from SURFACES.md. A
  feature that adds or changes something a person or another system uses
  directly, and has no surface document yet, gets one before its tasks
  dictate the interface (what goes in it: fdf-brainstorm, *Surface
  document*). A feature with no such interface says `surface: none` in its
  frontmatter.
- **Another feature's ground.** A plan may add to something another feature
  built — a page, an endpoint, a table — as long as every scenario of that
  feature stays true. The addition is this feature's behavior: its interface
  goes in this feature's `slug.surface.md`, its proof in this feature's
  `slug.test.md`, and this feature names the other in `depends-on` in its
  frontmatter (add it now if the brainstorm did not). When a scenario of a
  delivered (`done` or `adopted`) feature would stop being true, that is not a
  task: stop and tell the user. That feature changes only through a Change
  (fdf-change), and the user decides whether it comes first.

**Write for a zero-context implementer.** Whoever executes a task may be a
fresh agent that sees only that task file and `slug.spec.md` — no session
memory, no neighbouring tasks. Every name, signature, path, endpoint and
payload shape a task needs must be written in the task itself. "As discussed"
and "similar to task 01" are plan failures.

## Process

The bundle's existing plans, tasks and test documents may predate these
rules. Take paths and commands from them, never their shape: where one differs
from this skill, follow the skill.

1. **Read the feature.** Read `<feature-id>.md`, `<feature-id>.spec.md` and
   `<feature-id>.surface.md` if it exists, and run `fdf validate`. The
   feature must be `specified`; otherwise route with fdf-help. Unless you
   came here through fdf-help, do its steps 6 and 7 now: read `DOMAIN.md`,
   list the practices, debts and bugs whose paths overlap the code this
   feature will touch (the `grep` in fdf-help step 6), and announce "Using
   fdf-plan — <feature-id> is specified."
2. **Survey the project before writing anything**: the real directories, the
   patterns the code already follows, the test harness and the commands.
   Paths you invent are confidently wrong.
3. **Find the gaps, and ask.** List every contract a task will need: a column
   or field, an API shape, a message, a naming rule, a concept `DOMAIN.md` has
   no term for, and — above all — what happens in each edge case the
   scenarios touch. For each one the spec and
   the surface document do not define, STOP and ask the user, one question at
   a time. Then record each answer:
   - **an answer that says what the software does** in some case — it
     refunds, rejects, retries, sends — is behavior. Do all three:
     a. one new `Scenario:` in the feature document per outcome that can be
        checked on its own;
     b. its case in the test document (step 6) — only what a scenario names
        gets one (F8);
     c. a bullet under the spec's `## Design decisions`, and set the spec's
        `timestamp`. The feature is still in flight, so its spec may be
        amended, and the user's answer is the approval;
   - an answer about where or how the code is built → a bullet under the
     spec's `## Design decisions` only (the feature is still in flight, so its
     spec may be amended; the user's answer is the approval for that edit);
   - an interface detail → `<feature-id>.surface.md`;
   - **a concept with no term in `DOMAIN.md`** that the code will name (a new
     type, table or command) → propose the term, with its `instead-of`
     words, to the user now: a new term is a Context-document edit, and once
     approved it is written to `DOMAIN.md` and logged
     (`fdf log '**Context**: …'`). An entry that names the banned words puts
     each in a code span — `` `shop` `` — or F12 flags the log itself.

   A `specified` feature, sent here only to close a gap in its approved spec,
   not for a plan? Record a. and c., run `fdf validate` (F8 asks for the case
   only from `planned` on), log it in the feature's log, and stop: b. comes
   with step 6. From `planned` on, do all three.

   Never invent an answer mid-plan. A gap you only notice later — while
   writing a task's steps — is handled the same way: stop, ask, record. Ask
   about every undecided case whose result a user or another system could
   see (a customer applying the same coupon twice); decide purely internal
   details yourself, and write them into the task.
4. **Anything knowingly left out → a debt, proposed.** If the plan
   deliberately leaves something undone — a migration the feature does not
   need yet, a call site left on the old mechanism — propose a debt to the
   user; on approval, file it with `fdf debt --resource <paths> [<group>/…]<slug>`
   and replace its `TODO` text (`description`, `# Gap`, `# Cost`) and the
   `TODO.` of its listing line in `debts/INDEX.md`. A deferral written into a task's
   `# Steps` as a TODO
   is invisible the moment that task is `done`. Something the user rules out
   that no document promises — "partial refunds: out of scope for now" — is not
   a debt: record it as a decision in the feature's log,
   `fdf log <feature-id> '**Decision**: <what is out of scope>, agreed with <who>.'`
   (`<who>` is the name the user gave, or "the user" — never a name you
   guessed.)
5. **Decompose into tasks.** Write each task as a file in the task directory:
   `<feature-id>/01-<name>.md`, `02-<name>.md`, … — two digits, then a short
   lowercase hyphenated name. One sitting finishes one task; fold setup and
   scaffolding into the task whose deliverable needs them. Every task leaves
   the project building and its existing tests passing: a task that changes a
   signature also updates every caller of it — list them with
   `grep -rn '<name>(' .`, tests included, and name each in the task. Each
   task writes the tests of the scenarios it builds — the file, the test's
   name, what it asserts — and runs them in `# Acceptance`; the final task
   writes only what step 7 allows. Each task:

   ```markdown
   ---
   type: Task
   status: pending
   title: Refund API
   description: POST /payments/{id}/refunds calls the PSP and records the refund.
   resource: [internal/payments, cmd/api/routes.go]
   depends-on: 01-refund-model
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Objective

   One sentence: what exists once this task is done.

   # Steps

   1. Exact files, function signatures, status codes, payload shapes and
      messages. Whatever one task produces and another consumes is spelled
      out identically in both files.
   2. Follow `/practices/permission-checks` — its `# Rules` bind this code.
   3. Write `TestRefund_Full` in `internal/payments/refund_test.go`: a settled
      payment refunded in full is marked `refunded`. Write
      `TestRefundEndpoint_Full` in `tests/refund_test.go`: the endpoint returns
      `200` and `status: "completed"`.

   # Acceptance

   - Scenario "Full refund of a settled payment" — `go test ./internal/payments -run TestRefund_Full` and `go test ./tests -run TestRefundEndpoint_Full` pass.
   ```

   The paths, test names and commands in this template are the example
   project's: take yours from step 2's survey. A unit test belongs to the
   task that builds the code it tests.

   - **`resource:`** lists the project paths the task touches, and each must
     already exist (R1 fails otherwise). For a file the task will **create**,
     list the nearest directory that already exists: `src/refunds/policy.ts`
     in a project with no `src/refunds/` yet → `src`. Never create a
     placeholder file or directory to satisfy R1. When the nearest existing
     directory is the project root itself (a greenfield project), leave
     `resource:` out — never write `.`; the `# Steps` always name the files
     the task creates.
   - **`depends-on:`** names other tasks in the same directory by file name
     without `.md` (`01-refund-model`) — never by full ID. Write it as a YAML
     list, `depends-on: [01-refund-model, 02-refund-api]`; a single name may
     also be written bare, `depends-on: 01-refund-model`. Never comma-joined
     without the brackets (`depends-on: 01-refund-model, 02-refund-api`): that
     is one bogus name, and fails F6. The graph must be acyclic; fdf-execute
     runs tasks in parallel by it.
   - **Practices.** For each task, find the practices that govern it:
     `grep -rn -A3 -e '^status:' -e '^applies-to:' docs/fdf/practices` (a bare
     `applies-to:` lists its paths on the `- ` lines below it), skipping any
     whose `status:` is `superseded`. A practice governs the task when one of its
     paths is the same as, a parent of, or inside one of the task's paths —
     its `resource:`, or where its `# Steps` create files. Name each one in the
     task's `# Steps` ("Follow `/practices/<slug>`"), because the implementer
     may be a fresh agent with no reason to go looking. If the plan must depart
     from a practice, that is a decision for the user, not a detail for the
     task.
   - **`# Acceptance`** names each scenario the task satisfies or contributes
     to, one bullet per scenario —
     `- Scenario "<exact name>" — <how to check it>` — the name copied exactly
     as its `Scenario:` line spells it, on one line. Behavior that only a task
     or a test case names, and no scenario, is a plan failure: it needs its
     scenario first (step 3).
6. **Write the test document**, `<feature-id>.test.md`:

   ```markdown
   ---
   type: Test
   title: Instant refunds — test cases
   description: How each instant-refund scenario is proven.
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Test Cases

   ## Full refund of a settled payment

   `go test ./internal/payments -run TestRefund_Full` passes: the payment is
   marked refunded and the PSP sandbox shows one refund. End to end:
   `go test ./tests -run TestRefundEndpoint_Full` calls
   `POST /payments/{id}/refunds` on the running API and checks the `200` and
   the body's `status: "completed"`.
   ```

   One `## <scenario name>` heading per scenario, the name copied exactly as
   the Gherkin spells it (F8 matches it character for character; a bullet or
   a table row naming the scenario does not count). Under each: the concrete
   verification — the exact command, the test file and test name to write, or
   a step-by-step manual procedure — **and what passing looks like** (expected
   status, output, resulting state). "Run the tests" proves nothing.
   - If you cannot say how a scenario will be proven, STOP and ask the user;
     record the answer here.
   - A new endpoint or command gets an end-to-end check in each of its
     scenarios' cases, beside any unit test: it runs the endpoint or command
     and checks its output and its status or exit code — refusals included.
     If the project has no test of that kind yet, the plan adds the smallest
     harness (call the entry point, capture its output and exit code) and,
     through it, a check in every scenario's case — unless the user prefers a
     manual procedure. A partial preference — a manual procedure for some
     checks, automated tests for the rest — waives nothing in step 9.5: write the manual procedure, with its
     exact expected output, under the case of each scenario it exercises, and
     check every other output, message and exit code in `slug.surface.md`,
     refusals included, with a test that calls the command's entry point. If the spec does not say how a new command or
     endpoint is proven end to end, ask (step 3).
   - A UI change gets a check in a real browser, with the browser-test tool
     `STACK.md` or `INFRA.md` names (e.g. Playwright); if they name none,
     ask.
   - Every scenario keeps its case. A second check of a scenario goes under
     that scenario's heading; a check of behavior no scenario promises means
     the scenario is missing (step 3) — never a case of its own, and never a
     reason to delete the check.
   - When a scenario names an outcome whose exact wording matters — an error
     message, a URL, a button label — the literal string belongs in its case
     here: the Gherkin names the outcome, the test pins the words. The literal
     may use a word `DOMAIN.md` bans; the lexicon stops at the surface.
7. **The final task satisfies the test document.** Every plan ends with a task
   that runs every case in `<feature-id>.test.md` and passes them all; it
   `depends-on` every other task. It writes only a check no earlier task
   could — such as the project's first end-to-end harness and the checks
   that go through it: the task that builds a scenario writes that scenario's
   other tests. Its `# Acceptance` has one bullet per scenario, each on its
   own line — `- Scenario "<exact name>" — passes as <slug>.test.md says` —
   whatever an older plan in the bundle does. A one-task plan is allowed: its one task then does the work and satisfies
   the test document.
8. **Write the plan**, `<feature-id>.plan.md`: a `# Tasks` section with an
   ordered list that links every task file, each by a path relative to the
   plan — the task directory's name, then the file:

   ```markdown
   ---
   type: Plan
   title: Instant refunds — plan
   description: Three tasks; the last one proves the test document.
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Tasks

   1. [01-refund-model.md](instant-refunds/01-refund-model.md)
   2. [02-refund-api.md](instant-refunds/02-refund-api.md)
   3. [03-prove-refunds.md](instant-refunds/03-prove-refunds.md) — satisfies `instant-refunds.test.md`
   ```

   Plan order is the readable order; `depends-on` is the execution truth.
9. **Check every task** before you flip the status, and say one line per
   task as you do, naming what you fixed where a check failed:

   ```text
   01: practices ✓ acceptance ✓ checks ✓ build ✓ surface ✓ depends-on ✓ terms ✓ cases ✓
   ```

   For each one:
   1. every practice whose `applies-to` overlaps its paths is named in its
      `# Steps`;
   2. its `# Acceptance` names every scenario whose outcome it builds —
      including what a command prints or an endpoint returns;
   3. its checks exist once it is done: the task that builds a scenario
      writes that scenario's tests, its end-to-end checks too (unless step 7
      gives them to the final task), each under the exact name its case cites
      (a `-run` pattern that matches no test still passes); the final task
      runs them all;
   4. after this task alone, the project still builds and its tests pass —
      a test that calls a changed signature is a caller too;
   5. every command, message, status and exit code in `slug.surface.md` is
      built by some task and checked by some case — a usage error's exit code
      included, even where older features lack one. A message, status or exit
      code no scenario covers gets a `Scenario:` block in the feature
      document now (step 3a), with its case: the approved surface already
      decided it, so no question is needed. A name in a task's
      `# Acceptance` creates nothing;
   6. a task that adds to code another feature built (the files its tasks
      created, not a broad parent such as `internal`) or documents (an
      adopted feature's `resource`) means that feature is in this feature's
      `depends-on`;
   7. every new domain concept a task names (a new type or table) has a
      `DOMAIN.md` term, or one you proposed in step 3; a field or command
      that spells out an existing term needs none;
   8. each of its cases in the test document states what passing looks like:
      the resulting state, the output, or the literal error text. "`-run X`
      passes" alone is no case.
10. **Flip the feature to `planned`** and set its `timestamp` (run
   `date -u +%Y-%m-%dT%H:%M:%SZ` now), then log it in
   the feature's own log:
   `fdf log <feature-id> '**Planned**: <n> tasks; <what planning decided>.'`
   Now run `fdf validate` — it must exit 0, and use fdf-validate if it does
   not. Validate here and not earlier: tasks, test document, plan and the
   `planned` status are only valid together (a task directory fails F6 until
   its plan exists).
11. **Hand off** (next section): give the user the resume prompt. If they
    asked for the implementation too, ask "The plan is ready — continue here,
    or start fresh from this prompt?" and wait for the answer. Never go
    straight into fdf-execute, and never simply stop.

## Hand off to fdf-execute

The feature is `planned`; fdf-execute is next. Planning filled this
conversation with exploration and dialogue that execution does not need — the
plan was written so that a reader with none of it can carry it out.

- **If the user asked only for a plan**, recommend in one line that they
  compact the conversation (`/compact` in Claude Code) or start a fresh
  session, and give them a prompt to resume with.
- **If the user asked for the implementation too**, give them the resume
  prompt below, then ask whether to continue here or from that prompt in a
  fresh session — do not simply stop, and do not simply carry on.

The resume prompt:

```text
Use the fdf-execute skill on <feature-id> (status: planned).
Plan: docs/fdf/<feature-id>.plan.md (<N> tasks). Batches from depends-on:
  1. 01-…, 02-…   2. 03-…   3. 04-… (satisfies <slug>.test.md)
Suggested models: 01, 02 mechanical → fast (e.g. Sonnet-class);
  03, 04 need judgment → most capable (e.g. Opus-class).
```

A task is **mechanical** when its `# Steps` leave nothing to decide, and needs
**judgment** when they leave design latitude, cross module boundaries, or
touch concurrency, security or a data migration; the task that proves
`slug.test.md` always needs judgment.

The prompt points at files and never carries a decision they lack. If you are
about to write a sentence of context the plan does not contain, the plan is
incomplete: put it in the task or the spec, then write the prompt.

## Rules

- No task without acceptance criteria; no scenario without a test case; no
  line of `slug.surface.md` that no task builds.
- Checks live in the case of the scenario they prove. Never move a check out
  of `# Test Cases` to silence a warning; use fdf-validate for `warn:` lines
  too.
- The plan is done when a stranger could implement it. Re-read each task
  asking "what would I have to guess here?" — then put the answer in the file.
- Trail documents are siblings of the feature (`slug.plan.md`,
  `slug.test.md`); the task directory holds only tasks. Never nest a spec,
  plan or test document inside it.
- No code while planning: this skill writes documents only.
