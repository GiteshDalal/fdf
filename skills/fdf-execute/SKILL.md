---
name: fdf-execute
description: Use when an FDF feature is planned or implementing and its tasks need to be worked — before writing implementation code.
---

# FDF Execute

Implement a planned feature task by task, statuses always truthful.

New to FDF? The format is defined in the bundle itself at
`docs/features/SPEC.md` — exact frontmatter fields, casing and position
rules, and the F/R validation rules this skill cites. The fdf-help skill
explains how the four fdf skills fit together.

Stem paths for a feature `<group>/<slug>`:

- Feature: `<group>/<slug>.md`
- Spec / plan / test: `<group>/<slug>.spec.md`, `.plan.md`, `.test.md`
- Optional: `<group>/<slug>.surface.md`, `<group>/<slug>.log.md`
- Tasks only: `<group>/<slug>/NN-….md`

## Choose a mode

- **Serial** (default; required when subagents are unavailable): work tasks
  one at a time in `slug.plan.md` `# Tasks` order.
- **Subagent-driven**: compute topological batches from `depends-on`; all
  tasks whose dependencies are `done` run in parallel (one subagent each);
  join; validate; next batch. Plan order breaks ties. **You own the
  bundle**: you flip every status, update every timestamp, and run every
  `fdf validate`; subagents never edit files under the bundle. One writer
  means no races and a serialized validate after each change.

## Per task

1. Set `status: in-progress` in the task file; feature to `implementing` if
   this is the first task. `fdf validate` after every frontmatter change.
2. Do the work per `# Steps`; touch only paths consistent with `resource:`.
3. Verify `# Acceptance`; set `status: done`; update `timestamp`. When
   completing the FINAL task, flip the task and the feature status in the
   same edit before validating — a lone final-task flip fails F4
   ("implementing but every task is done").

## Dispatching a subagent

The implementer prompt contains, in order:

1. Its task file path plus `slug.spec.md`, `slug.plan.md`, the feature doc,
   and `slug.surface.md` if present — read these first, task file foremost.
2. The scope rule: touch only paths consistent with the task's `resource:`;
   siblings run in parallel and staying in-scope is what prevents
   collisions. Never edit anything under the bundle directory.
3. The exit contract: verify every `# Acceptance` item by actually running
   it, then report files changed, the exact command and output per
   acceptance item, and anything unverified — stop and report a blocker
   rather than improvising around the spec.

## Blockers

- A fix attempt failed and the cause isn't obvious and mechanical → stop
  retrying. The task stays `in-progress`; note the blocker, the attempts,
  and the exact errors in the task body.
- Independent siblings continue; batches needing the blocked task stall.
- Report with a specific question ("Acceptance requires X; SPEC section Y
  implies Z — which wins?"), never just "it's broken".

## Completion gate

- The final task (the `slug.test.md`-satisfying one) can only be `done` when
  every test case in `slug.test.md` passes — run the commands it names; UI
  cases are verified in a real browser (Playwright) when specified.
- All tasks done → feature `status: done`, `fdf validate` exit 0. Never flip a
  feature to done with a failing or unrun `slug.test.md` case.
- Log the completion in the feature's optional `slug.log.md` (stem sibling;
  not a nested `LOG.md` inside the task directory) — major decisions and
  notable user interactions — and note it in the bundle-root LOG.md.
- The completion report shows evidence, not claims: per `slug.test.md` case,
  the command run and its actual output (screenshot for browser checks). A
  case you didn't run is reported as unrun — and the feature stays
  `implementing`.

## Context-document review (after the feature is done)

The Context docs (`STACK.md`, `ARCHITECTURE.md`, `SURFACES.md`, `INFRA.md`)
are **critical and immutable without explicit approval** — this is one of
only two places they may change (the other is fdf-init). After completing
the feature, check whether the work made any of them stale:

- New dependency, language, or data store → STACK.md.
- New pattern, module boundary, or convention → ARCHITECTURE.md.
- New surface convention — API error envelope, naming rule, pagination
  style, CLI flag/output convention, UI pattern, exemplar link, event
  shape → SURFACES.md.
- New infrastructure — a cache (e.g. Redis), queue, service, env var,
  deployment target → INFRA.md.

If something changed, **propose** the specific edit to the user and wait for
explicit approval. Only on approval: make the edit, and log it (what changed
and why) in `slug.log.md` (if used) and the bundle LOG.md. If nothing
changed, say so in one line. Never edit a Context doc silently, and never
edit one the user didn't approve. Remind the user, briefly, that keeping
these accurate is what keeps the work grounded — agentic engineering, not
vibe coding.

## Rules

- Statuses reflect reality, not intent — flip in-progress before working.
- A blocked task stays in-progress with the blocker noted in the task body.
- Context docs change only with explicit user approval, and every change is
  logged.
