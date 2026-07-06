---
name: fdf-execute
description: Use when implementing a planned FDF feature — works the tasks serially or dependency-parallel, flips statuses, and gates completion on TEST.md, moving status planned → implementing → done.
---

# FDF Execute

Implement a planned feature task by task, statuses always truthful.

## Choose a mode

- **Serial** (default; required when subagents are unavailable): work tasks
  one at a time in PLAN.md `# Tasks` order.
- **Subagent-driven**: compute topological batches from `depends-on`; all
  tasks whose dependencies are `done` run in parallel (one subagent each);
  join; validate; next batch. PLAN.md order breaks ties.

## Per task

1. Set `status: in-progress` in the task file; feature to `implementing` if
   this is the first task. `fdf validate` after every frontmatter change.
2. Do the work per `# Steps`; touch only paths consistent with `resource:`.
3. Verify `# Acceptance`; set `status: done`; update `timestamp`. When
   completing the FINAL task, flip the task and the feature status in the
   same edit before validating — a lone final-task flip fails F4
   ("implementing but every task is done").

## Completion gate

- The final task (the TEST.md-satisfying one) can only be `done` when every
  test case in TEST.md passes — run the commands it names; UI cases are
  verified in a real browser (Playwright) when specified.
- All tasks done → feature `status: done`, LOG.md entry, `fdf validate`
  exit 0. Never flip a feature to done with a failing or unrun TEST.md case.

## Rules

- Statuses reflect reality, not intent — flip in-progress before working.
- A blocked task stays in-progress with the blocker noted in the task body.
