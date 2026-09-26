---
name: fdf-execute
description: Use when an FDF feature is planned or implementing and its tasks need to be worked — before writing implementation code.
---

# FDF Execute

Implement a planned feature task by task, with every status telling the truth.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

This skill executes a **feature**. The tasks of a post-delivery `Change` or
`Fix` live under `<change-id>/` and are worked exactly the same way, but the
work around them — declaring effects, amending the affected features' living
documents — is fdf-change's.

Below, `<feature-id>` is the feature's full ID, such as
`features/payments/instant-refunds`, and paths are relative to the bundle
root, `docs/fdf/`:

- Feature: `<feature-id>.md`
- Spec, plan, test: `<feature-id>.spec.md`, `.plan.md`, `.test.md`
- Optional: `<feature-id>.surface.md`, `<feature-id>.log.md`
- Tasks: `<feature-id>/NN-<name>.md`

## Mechanics

- **You own the bundle.** You flip every status, set every timestamp and run
  every `fdf validate`; an implementer you delegate to never edits a file
  under the bundle. One writer means no races.
- **Timestamps.** Every document you change in substance — a status flip
  included — gets `timestamp:` set to the output of
  `date -u +%Y-%m-%dT%H:%M:%SZ`, run at the moment of that edit — the task,
  the feature, the surface document alike. Never a time you estimate, round
  or reuse from an earlier edit, and never your local time with `Z` added. A
  maintenance edit (a lexicon fix, a
  reference or path repair) leaves `timestamp` as it is.
- **Validate** after every change to a status or other frontmatter, except
  inside a pair of edits that is only valid together (the final task and the
  feature's `done`, below).
- **Log** with single quotes — `fdf log <feature-id> '**Done**: …'` —
  because inside double quotes the shell runs anything between backticks. An
  apostrophe would end the quotes: write ’ instead. Never edit an entry once
  it is written, except by a maintenance edit (fdf-help).
- **Filing a debt or bug**: `fdf debt` and `fdf bug` scaffold a document full
  of `TODO` text (its `description:` included) and add a listing line to the
  `INDEX.md` beside it that ends `- TODO.`. Replace all of it with real
  text.

## Start

1. Read `<feature-id>.md`, its spec, plan, test document, surface document (if
   any), and every task. Run `fdf validate`. Then do fdf-help's steps 6 and
   7, even if you did them while routing: read `DOMAIN.md` and every practice,
   debt and bug whose paths overlap the tasks' paths, and announce "Using
   fdf-execute — <feature-id> is <status>". Coming straight from fdf-plan in
   this conversation? Start only once the user has answered its question —
   continue here, or in a fresh session — and say which they chose: "You
   chose to continue here." If they chose a fresh session, or gave no
   answer, stop.
2. **Resuming a feature already `implementing`?** Skip tasks that are `done`.
   Read the body of each `in-progress` task for a recorded blocker before you
   continue it.
3. **Choose a mode**, and say it in one line —
   `Mode: <subagent-driven | workflow | serial> — <why>`:
   - **Subagent-driven** — whenever the harness has subagents and the plan has
     more than two tasks. Work in batches: a batch is every task whose
     `depends-on` tasks are all `done`, one implementer each, in plan order.
     Tasks in one batch whose `resource:` paths overlap run one after the
     other instead. Wait for every implementer in the batch, check their
     work, validate, then start the next batch.
   - **Workflow** — where the harness offers scripted multi-agent workflows, a
     plan with several parallel batches can run as one. It starts many agents
     at once, so offer it, and start one only when the user agrees.
   - **Serial** — one task at a time in the plan's `# Tasks` order: for a
     one- or two-task plan, or when the harness has no subagents.

## Per task

In every mode, each task goes through these steps. "You" is the agent that
owns the bundle; in subagent mode, step 3 is the implementer's.

1. **(You) Mark it `in-progress` — before any code.** Run
   `date -u +%Y-%m-%dT%H:%M:%SZ`. Set the task's `status: in-progress` and
   its `timestamp` to that output. If this is the feature's first task, set
   the feature's `status: implementing` and its `timestamp` in the same edit.
   Run `fdf validate`.
2. **(You) Find the practices that govern it**:
   `grep -rn -A3 -e '^status:' -e '^applies-to:' docs/fdf/practices` (a bare
   `applies-to:` lists its paths on the `- ` lines below it), skipping any
   practice whose `status:` is `superseded`. A practice governs the task when one of its
   paths is the same as, a parent of, or inside one of the task's paths — its
   `resource:`, or where its `# Steps` create files. Its `# Rules` are what the
   code must do. Read, too, the conventions `ARCHITECTURE.md` and
   `SURFACES.md` give for the layer the task touches.
3. **(Implementer) Do the work** as `# Steps` says, touching only paths
   consistent with the task's `resource:` and `# Steps`, and following the
   governing practices' `# Rules`. A rule that seems wrong here is raised with
   the user, never quietly bypassed. Then verify every `# Acceptance` item by
   running it.
4. **(You) Check it.** Re-run each `# Acceptance` command yourself and read the
   output. Only when every item passes: run `date -u +%Y-%m-%dT%H:%M:%SZ`
   again, set the task's `status: done` and its `timestamp` to that output,
   and run `fdf validate`.
   - **The final task** (the one that satisfies `<feature-id>.test.md`) is
     never flipped here. Once it is the only task left, go to *Completion* and
     work its six numbered steps in order; the flip is step 3 (a lone
     final-task flip fails F4, "implementing but every task is done").

## Delegating a task

**Pick the agent.** If the project or harness defines specialized agents (in
Claude Code, those under `.claude/agents/`), use the one whose description
fits the task — a test-writing agent for test work, a frontend agent for UI
work. Otherwise use a general-purpose agent.

**Pick the model**, where the harness lets you, by what the task demands:

- **Mechanical** — `# Steps` leave nothing to decide: exact files, signatures
  and payloads, following a pattern the code already has (wiring, a fully
  written-out migration, tests for a given list of cases). Use a fast model;
  in Claude, a Sonnet-class one.
- **Judgment** — the steps leave design latitude, or the task crosses module
  boundaries, touches concurrency, security or a data migration, or is the
  final task that proves `slug.test.md`. Use the most capable model; in
  Claude, an Opus-class one.

A mechanical task that fails its acceptance gets one re-run on the most
capable model, with the failure output. A second failure is a blocker (see
*Blockers*).

**Write the prompt.** The implementer prompt contains, in order:

1. The paths to read first, task file foremost: the task file, the feature
   document, `slug.spec.md`, `slug.plan.md`, `slug.surface.md` if it exists,
   `slug.test.md` for the final task, `DOMAIN.md` (the names the code must
   use), `ARCHITECTURE.md` and `SURFACES.md` (the conventions of the layer it
   touches), and **every practice that governs the task** (step 2 above). A
   fresh implementer has no other way to learn what binds it.
2. The scope rule: touch only paths consistent with the task's `resource:`
   and `# Steps` — sibling tasks run in parallel, and staying in scope is what
   prevents collisions — and never edit anything under `docs/fdf/`.
3. The exit contract: verify every `# Acceptance` item by actually running it,
   then report the files changed, the exact command and its output for each
   acceptance item, and anything not verified. Stop and report a blocker
   rather than improvising around the spec.

## Blockers

- **A fix attempt failed and the cause isn't obvious and mechanical** → stop
  retrying and use **fdf-debug** to find the root cause. The task stays
  `in-progress`; write the blocker, the attempts and the exact errors into the
  task's body. The feature is still in flight, so the repair stays task work —
  a Change or Fix is only for delivered features.
- Independent tasks continue; batches that need the blocked task wait.
- Report with a specific question ("Acceptance requires X; the spec's section
  Y implies Z — which wins?"), never just "it's broken". Record the user's
  answer before you code it, as fdf-plan step 3 does: a decision in the
  spec's `## Design decisions`, an interface detail in `slug.surface.md`, new
  behavior as a new scenario with its `slug.test.md` case (and its name in the
  final task's `# Acceptance`); add any task it needs (*A task the plan
  lacks*).
- **Descoping.** If the user decides the blocked work will not be done in this
  feature, descope it rather than hold the feature at `implementing`
  indefinitely. Never mark a blocked task `done`. In one set of edits:
  1. delete the task file;
  2. remove its line from the plan's `# Tasks`, and its name from every other
     task's `depends-on`;
  3. remove each scenario only it would have proven, with that scenario's
     case in `slug.test.md`, and the scenario's name from every task's
     `# Acceptance` — the feature must not promise what it will not do;
  4. log the decision: `fdf log <feature-id> '**Decision**: … descoped, agreed with <who>.'`
     (`<who>` is the name the user gave, or "the user" — never a name you
     guessed);
  5. file a debt naming what was left undone — part of the descoping the user
     decided: `fdf debt --resource <paths> [<group>/…]<slug>`, then replace
     its `TODO` text (`description`, `# Gap`, `# Cost`) and the `TODO.` of its
     listing line;
  6. run `fdf validate`.

**A task the plan lacks.** When the work shows a task is missing — a defect
inside this feature's scope, a step the plan forgot — add it before the final
task: write it as fdf-plan describes, link it in the plan's `# Tasks`, and add
it to the final task's `depends-on`. To renumber tasks, use `fdf mv`, which
repairs the plan's links — then correct each moved task's link text in
`# Tasks`, which it leaves as it was. Then validate.

## Completion

When every other task is `done`, the final task finishes the feature:

1. **Run every case in `<feature-id>.test.md`** — the commands it names, and
   each manual procedure; a UI case in a real browser, with the tool the case
   names. The final task can only be `done` when every case passes. A case you
   did not run is reported as unrun, and the feature stays `implementing`. A
   manual case you cannot run yourself (a device, an account you lack) goes to
   the user: give them its steps, and record what they report as its
   evidence.
2. **Check `<feature-id>.surface.md` against the build.** Run every form it
   lists — the success output, and each error with its status or exit code —
   and compare what the software does with what the document says. Where
   they differ on something the document already stated, the code is wrong:
   repair it before step 3, never the document. The build settles interface
   details the design left open (an error code, a flag, a label, an event
   field): add those to the document, with a new `timestamp`, so it
   describes the interfaces as they now are. Say one line:
   `surface — <forms run>: matches`, `surface — edited: <what>`, or
   `surface — none` for a feature with no interface. A feature
   that exposes an interface and has no surface document gets one now
   (fdf-brainstorm, *Surface document*); one with no interface says
   `surface: none` in its frontmatter.
3. **Flip the final task and the feature to `done`** in one edit, with their
   timestamps, then run `fdf validate` — exit 0, or use fdf-validate.
4. **Log the completion** in the feature's own log, never the root `LOG.md`:
   `fdf log <feature-id> '**Done**: <what shipped>; <any decision the build took that the spec does not record, and who agreed to it>.'`
   (Who: the name the user gave, or "the user" — never a name you guessed.)
5. **Report with evidence, not claims** — the surface line from step 2, and
   one line per `slug.test.md` case —
   `<case> — <command> → <its actual output>` (a screenshot for a browser
   check).
6. **Run the project-document review** below. The feature is not finished
   until its four questions are answered.

## Project-document review (after the feature is done)

The Context documents and the practices are **critical and change only with
explicit approval**. This review is where one piece of work proposes such
changes; fdf-change ends with the same four questions, and fdf-checkpoint
audits everything periodically. Ask the four questions in order. Question 1
takes five lines, one per Context document; questions 2–4 take one line
each, where "no" is a valid answer.

**1. Is a Context document now stale?**

Open each of the five and compare it with what this feature changed (`git
diff`, and the files its tasks created). Answer in one line per document,
naming what you checked — "no updates" without having looked is not an
answer. For example:

```text
STACK.md — dependencies (git diff go.mod package.json): none added — current
ARCHITECTURE.md — new directory src/refunds/ (git status --short): not in the code map — stale, propose adding it
SURFACES.md — new endpoint POST /refunds, checked against each convention (inputs, success output, errors, status codes): returns HTML errors where SURFACES.md says JSON — stale or a defect: ask
INFRA.md — new environment variable or build step: REFUNDS_PSP_KEY is new — stale, propose adding it
DOMAIN.md — new named concept: Refund window, no term yet — stale, propose the term
```

For each new directory, `grep -n '<new directory>' docs/fdf/ARCHITECTURE.md`
tells you whether the code map lists it. Any of the five is stale when the
new code adds what it should list: a directory for `ARCHITECTURE.md`'s code
map, an argument shape or an output form for `SURFACES.md` — propose the
amendment. When new or older code contradicts one of them, even when older
code did so first, there are two readings: the document is stale, or the
code drifted — a bug to propose, when a user could see it. When the evidence
does not say which, ask the user.

- A new dependency, language or data store → `STACK.md`.
- A new pattern, module or boundary, or convention → `ARCHITECTURE.md`.
- A new surface convention — an API error envelope, a naming rule, a
  pagination style, a CLI flag or output convention, a UI pattern, the wording
  a surface shows people for a term, an exemplar link, an event shape →
  `SURFACES.md`.
- New infrastructure — a cache, queue, service, environment variable,
  deployment target → `INFRA.md`.
- A new concept with a name, or a name the team has now settled →
  `DOMAIN.md`, with its `instead-of` words so the old names stop spreading. The
  lexicon is internal vocabulary; a label that shows a term to people under
  another word is a surface matter, not a new term and not drift.

**2. Did this feature diverge from a practice that governs it?**

If the code you wrote under a practice's `applies-to` does not follow its
`# Rules`, exactly one of two things is true, and you must say which: the code
is wrong and you fix it, or the divergence is deliberate and belongs in that
practice's `# Exceptions`, with its reason — approved, like any other practice
edit. Silence is neither.

**3. Did this feature establish — or repeat — a mechanism the next one will
repeat too?**

The threshold is the **second** occurrence, not the first — and a mechanism
this feature is the third or later to follow, with still no practice, counts
the same. (A repeated surface or naming convention — an output format, a
message style — belongs to question 1, as a `SURFACES.md` or
`ARCHITECTURE.md` edit, never to a practice.) One feature doing a
thing is what its `slug.spec.md` records; two features doing it the same way
is a practice waiting to be written before a third guesses differently. When
that line is crossed, propose a practice. On approval, write it as fdf-init
*Practices in an existing project*, *Writing one*, steps 2 and 4 describe:
`fdf practice [<group>/…]<slug>`; replace the scaffold's text — its
commented `# applies-to:` line whole, since uncommented as it is it fails R1
— and the `TODO.` of its listing line; then log it.

A practice is **extracted, not authored**: the decision was already made, in
the specs of the features that made it. Read them and lift the rules out. The
spec stays frozen as the episode ("we decided X for this feature"); the
practice carries the present tense ("X is how this project does it") and is
the one later work amends. `# Rules` is required and non-empty (F11):
imperative, short, what code MUST do. `applies-to` lists the existing repo
paths it governs (R1) and is how later work finds it — a feature never lists
the practices it follows. A practice has no Gherkin, spec, plan or tasks; its
only sibling may be `<slug>.log.md`. Write only what the code **already
does**: a better way nobody has adopted yet is a proposal to the user, not a
practice.

**4. Did this feature knowingly leave something undone — or find something
broken?**

- **Left undone** — work deferred to ship, a rule the new code follows that
  older code does not, a task descoped because it was blocked → a **debt**.
  `# Gap` states it concretely (files and counts, not impressions — F13),
  `# Cost` says what carrying it risks, and `resource` names the paths.
  When an open debt already names the same gap, amend its `# Gap` and
  `resource` instead of filing a second one. A feature that shipped without
  its batch import is genuinely done *and* has left a gap: the debt is what
  lets you state both.
- **Broken outside this feature's scope** — the software doing something
  observably wrong in code this feature does not own → a **bug**, carrying
  the reproduction under `# Symptom` and what should happen under
  `# Expected`: `fdf bug --affects <owning-feature-id>[,…] --resource <paths> [<group>/…]<slug>`,
  where `--affects` names the features the defect shows in — not the one you
  are building — and is left out when no feature documents that code. A defect
  inside this feature's scope is simply its next task.

**Then, for every answer that is not "no": propose** the specific edit — the
Context document change, the practice, the debt or the bug — to the user, and
wait for explicit approval. Only on approval, make it:

- a Context document edit gets a new `timestamp` and a root log entry:
  `fdf log '**Context**: <what changed, and why>.'` — a banned word the entry
  names goes in a code span, or F12 flags the log itself;
- a practice edit is logged in the practice's own log:
  `fdf log <practice-id> '**Amended**: …'` (its full ID, such as
  `practices/permission-checks`);
- a debt or bug is filed with the command above; replace every `TODO` in it,
  its `description` included, and the `TODO.` of its listing line.

Never edit a Context document or a practice silently, or one the user did not
approve. Remind the user, briefly, that keeping these accurate is what keeps
the work grounded — agentic engineering, not vibe coding.

## After a feature is done

It is delivered. From here its behavior never changes in place: later work on
it is a `Change` or a `Fix` under `changes/`, driven by fdf-change, and a
report that it is broken starts with fdf-debug, which decides which of the two
the repair is. Never fork a second feature document for the same capability,
and never edit a delivered feature's Gherkin directly.

## Rules

- Log at the **narrowest scope that fits**: a decision about one feature goes
  in its `slug.log.md`, one about a group in that group's `LOG.md`, and only a
  bundle-wide decision in the root `LOG.md`. A task's entry goes in its
  feature's log.
- Statuses reflect reality, not intent: `in-progress` before working, task by
  task; never a batch of flips at the end.
- A blocked task stays `in-progress` with the blocker written in its body —
  until it is unblocked or, on the user's decision, descoped.
- Context documents and practices change only with explicit user approval,
  and every change is logged.
