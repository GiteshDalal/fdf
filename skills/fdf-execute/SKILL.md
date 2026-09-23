---
name: fdf-execute
description: Use when an FDF feature is planned or implementing and its tasks need to be worked — before writing implementation code.
---

# FDF Execute

Implement a planned feature task by task, statuses always truthful.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the fdf skills fit together.

This skill executes a **feature**. Tasks belonging to a post-delivery
`Change` live under `changes/<id>/` and are worked exactly the same way, but
the surrounding workflow — declaring effects, amending the affected feature's
living documents — is fdf-change's.

Stem paths for a feature `<group>/<slug>`:

- Feature: `<group>/<slug>.md`
- Spec / plan / test: `<group>/<slug>.spec.md`, `.plan.md`, `.test.md`
- Optional: `<group>/<slug>.surface.md`, `<group>/<slug>.log.md`
- Tasks only: `<group>/<slug>/NN-….md`

## Choose a mode

Say which mode you are using, and why, in one line.

- **Subagent-driven** — recommended whenever the harness has subagents and
  the plan has more than two tasks. Compute topological batches from
  `depends-on`; all tasks whose dependencies are `done` run in parallel (one
  subagent each); join; validate; next batch. Plan order breaks ties. Each
  subagent starts from a clean context, which is what the plan was written
  for, and your own context stays free to oversee the whole feature.
- **Workflow** — where the harness offers scripted multi-agent workflows, a
  plan with several parallel batches can run as one. It starts many agents at
  once, so offer it and start one only when the user agrees.
- **Serial** — one task at a time in `slug.plan.md` `# Tasks` order: for a
  one- or two-task plan, or when the harness has no subagents.

In every mode **you own the bundle**: you flip every status, update every
timestamp, and run every `fdf validate`; subagents never edit files under the
bundle. One writer means no races and a serialized validate after each change.

## Per task

1. Set `status: in-progress` in the task file; feature to `implementing` if
   this is the first task. `fdf validate` after every frontmatter change.
2. **Check the practices that govern those paths** before writing anything:
   a practice whose `applies-to` covers a path this task touches — its
   `resource:`, or where its `# Steps` create files — is binding, and its
   `# Rules` are what the code must do. Follow them; if you
   believe a rule is wrong here, stop and raise it rather than quietly doing
   it your way.
3. Do the work per `# Steps`; touch only paths consistent with `resource:`
   and `# Steps`.
4. Verify `# Acceptance`; set `status: done`; update `timestamp`. When
   completing the FINAL task, flip the task and the feature status in the
   same edit before validating — a lone final-task flip fails F4
   ("implementing but every task is done").

## Delegating a task

**Pick the agent.** If the project or harness defines specialized agents (in
Claude Code, those under `.claude/agents/`), use the one whose description
fits the task — a test-writing agent for test work, a frontend agent for UI
work. Otherwise use a general-purpose agent.

**Pick the model**, where the harness lets you, by what the task demands:

- **Mechanical** — the `# Steps` leave nothing to decide: exact files,
  signatures and payloads, following a pattern the code already has (wiring,
  a fully written-out migration, tests for a given list of cases). Use a fast
  model; in Claude, a Sonnet-class one.
- **Judgment** — the steps leave design latitude, or the task crosses module
  boundaries, touches concurrency, security or a data migration, or is the
  final task that proves `slug.test.md`. Use the most capable model; in
  Claude, an Opus-class one.

A mechanical task that fails its acceptance gets one re-run on the most
capable model, with the failure output. A second failure is a blocker (see
Blockers).

**Write the prompt.** The implementer prompt contains, in order:

1. Its task file path plus `slug.spec.md`, `slug.plan.md`, the feature doc,
   `slug.surface.md` if present, and **every practice whose `applies-to`
   covers a path the task touches** (its `resource:`, and where its
   `# Steps` create files) — read these first, task file foremost. A
   subagent has no other way to learn that a practice binds it.
2. The scope rule: touch only paths consistent with the task's `resource:`
   and `# Steps`; siblings run in parallel and staying in-scope is what
   prevents collisions. Never edit anything under the bundle directory.
3. The exit contract: verify every `# Acceptance` item by actually running
   it, then report files changed, the exact command and output per
   acceptance item, and anything unverified — stop and report a blocker
   rather than improvising around the spec.

## Blockers

- A fix attempt failed and the cause isn't obvious and mechanical → stop
  retrying and use **fdf-debug** to find the root cause. The task stays
  `in-progress`; note the blocker, the attempts, and the exact errors in the
  task body. The feature is still in flight, so the repair stays task work —
  a Change or Fix is only for delivered features.
- Independent siblings continue; batches needing the blocked task stall.
- Report with a specific question ("Acceptance requires X; SPEC section Y
  implies Z — which wins?"), never just "it's broken".
- If the user decides the blocked work will not be done in this feature,
  descope it rather than hold the feature at `implementing` indefinitely:
  delete the task file, its `# Tasks` link and every sibling's `depends-on`
  entry for it (F6); remove any scenario only it would have proven, along
  with that scenario's `slug.test.md` case — the feature must not promise
  what it will not do; log the decision
  (`fdf log <group>/<slug> "**Decision**: …"`); and file a debt
  (`fdf debt [<group>/]<slug>`) naming what was left undone. Never mark a
  blocked task `done`.

## Completion gate

- The final task (the `slug.test.md`-satisfying one) can only be `done` when
  every test case in `slug.test.md` passes — run the commands it names; UI
  cases are verified in a real browser, with the tool the case names.
- All tasks done → feature `status: done`, `fdf validate` exit 0 (fdf-validate
  on failure). Never flip a feature to done with a failing or unrun
  `slug.test.md` case.
- Bring `slug.surface.md` up to date. The build settles interface detail the
  design left open (an error code, a flag, a label, an event field), and the
  surface document describes the interfaces as they now are. A feature that
  exposes an interface and has no surface document yet gets one now (see
  fdf-brainstorm, *Surface document*).
- Log the completion: `fdf log <group>/<slug> "**Done**: …"`. Say what
  shipped, and any decision the build took that the spec does not record, with
  who agreed to it. The entry goes in the feature's own log, never the root
  `LOG.md` (see Rules).
- The completion report shows evidence, not claims: per `slug.test.md` case,
  the command run and its actual output (screenshot for browser checks). A
  case you didn't run is reported as unrun — and the feature stays
  `implementing`.

## Project-document review (after the feature is done)

The Context docs and the practices are **critical and change only with
explicit approval**. This review is where one piece of work proposes such
changes: fdf-change ends with the same four questions, and fdf-checkpoint
audits everything periodically for what no single piece of work caught.
After completing the feature, ask four questions in order.

**1. Is a Context document now stale?**

- New dependency, language, or data store → STACK.md.
- New pattern, module boundary, or convention → ARCHITECTURE.md.
- New surface convention — API error envelope, naming rule, pagination
  style, CLI flag/output convention, UI pattern, the wording a surface shows
  people for a term, exemplar link, event shape → SURFACES.md.
- New infrastructure — a cache (e.g. Redis), queue, service, env var,
  deployment target → INFRA.md.
- A new concept with a name, or a name the team has now settled →
  DOMAIN.md (with its `instead-of` words, so the old names stop spreading).
  The lexicon is internal vocabulary — the bundle and the code's identifiers.
  A label or locale string that shows a term to people under another word is
  a surface matter, not a new term and not drift.

**2. Did this feature diverge from a practice it is governed by?**

A practice governs the paths in its `applies-to`. If the code you wrote there
does not follow its `# Rules`, exactly one of two things is true, and you must
say which: the code is wrong and you fix it, or the divergence is deliberate
and belongs in that practice's `# Exceptions` with its reason — approved, like
any other practice edit. Silence is neither.

**3. Did this feature establish a mechanism the next one will repeat?**

The threshold is the **second** occurrence, not the first. One feature doing a
thing is what its `slug.spec.md` records; two features doing it the same way
is a practice waiting to be written before a third guesses differently. When
that line is crossed, propose `fdf practice [<group>/]<slug>`.

A practice is **extracted, not authored**: the decision was already made, in
the specs of the features that made it. Read them and lift the rules out. The
spec stays frozen as the episode ("we decided X for this feature"); the
practice carries the present tense ("X is how this project does it") and is
the one later work amends. `fdf practice` scaffolds its sections, and
`fdf spec` shows a complete example.

`# Rules` is required and non-empty (F11): imperative, short, what code MUST
do. `applies-to` lists the existing repo paths it governs (R1) and is how
later work finds it — a feature never lists the practices it follows, so a
practice with no `applies-to` is a document nothing routes to. No Gherkin, no
spec, no plan, no tasks; the only sibling it may have is `<slug>.log.md`.
Link it from `practices/INDEX.md`.

Write only what the code **already does**. If the feature revealed a better
way nobody has adopted yet, that is a proposal for the user, not a practice
asserting it is already the rule.

**4. Did this feature knowingly leave something undone?**

Work deferred to ship, a rule the new code follows that older code does not, a
task that ended blocked on something outside the project. `fdf debt
[<group>/]<slug>` files it: `# Gap` states it concretely (files and counts, not
impressions — F13), `# Cost` says what carrying it risks, and `resource` names
the paths, which is how later work finds it.

This is the question that keeps `done` honest. A feature that shipped without
its batch import is genuinely done *and* has left a gap, and those are two
facts, not a contradiction — the debt is what lets you state both instead of
quietly rounding one off. A task descoped because it was blocked (see
Blockers) is the other case: its debt is what lets the feature reach `done`
without pretending the work happened.

Debt is a register entry, not a unit of work, so it gets no plan and no tasks.
When it is paid, flip it to `resolved` with a `# Resolution` saying what closed
it; `fdf debt --cleanup` clears resolved entries into `debts/LOG.md` so the
register stays a list worth reading.

A **defect** this work found outside its own scope — the software doing
something wrong in code this feature does not own — is not a debt and not this
feature's task: file it with `fdf bug`, carrying the reproduction under
`# Symptom` and what should happen under `# Expected`, so the next person
starts from your evidence. A defect inside this feature's scope is simply its
next task.

If something changed, **propose** the specific edit to the user and wait for
explicit approval. Only on approval: make the edit (a Context document gets a
new `timestamp`) and log what changed and why at its scope: the root
`LOG.md` for a Context document (`fdf log "**Context**: …"`), the practice's
own log for a practice (`fdf log practices/<slug> "…"`). If
nothing changed, say so in one line. Never edit a Context document or a
practice silently, and never edit one the user didn't approve. Remind the
user, briefly, that keeping these accurate is what keeps the work grounded —
agentic engineering, not vibe coding.

## After a feature is done

It is delivered. From here its behavior never changes in place: a later
alteration is a `Change` and a later bug is a `Fix`, both under `changes/`,
both driven by fdf-change. Do not fork a second feature document for the same
capability, and do not edit a delivered feature's Gherkin directly.

## Rules

- Log at the **narrowest scope that fits**: a decision about one feature goes
  in its `slug.log.md`, one about a group in that group's `LOG.md`, and only a
  bundle-wide decision in the root `LOG.md`. Logs are newest-first and never
  rewritten, so the only thing that keeps the root log readable is not writing
  feature-scoped entries into it. `fdf log <id> "<entry>"` finds the right
  log and creates it on first use; a task's entry goes in its feature's log.
- Statuses reflect reality, not intent — flip in-progress before working.
- A blocked task stays in-progress with the blocker noted in the task body —
  until it is unblocked or, on the user's decision, descoped (see Blockers).
- Context docs and practices change only with explicit user approval, and
  every change is logged.
