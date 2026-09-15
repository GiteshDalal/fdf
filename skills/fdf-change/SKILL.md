---
name: fdf-change
description: Use when a delivered (done or retired) FDF feature needs to change or a bug in it needs fixing — change requests, defects, and retirements, including ones spanning several features. Not for features still in draft/specified/planned/implementing.
---

# FDF Change

Amend a delivered feature without forking it or letting the bundle drift.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the skills fit together.

**Read the Context docs first** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`. A change lands in a system that already exists;
ground it in the documented stack and conventions, and flag explicitly when
the change would depart from them.

## Which document am I writing?

One question decides it:

> **Does this require the feature's Gherkin to change?**

| | `Change` (`fdf change`) | `Fix` (`fdf fix`) |
|---|---|---|
| What is wrong | The feature does the wrong thing, or the document never covered this case | The code drifted from what the document already says |
| Gherkin | Changes | Unchanged |
| Design gate | Yes — `slug.spec.md`, approved | None |
| Floor | Spec + the change document | A single file |
| Declares | `# Scenario changes` | `# Regression cases` |

The case people get wrong: **a bug that reveals the document was silent** on
a situation is a `Change`, not a `Fix`. Nobody ever decided that behavior, so
someone has to decide it now — and that needs the gate. If you are about to
write a regression case for a scenario that does not exist, stop: it is a
Change.

If the new behavior reads as its own `Feature:` block with its own
As-a / I-want / So-that, it is not a change at all — it is a **new feature**
(fdf-brainstorm), which may record its lineage with `depends-on`.

## Process

1. **Confirm the feature is delivered.** `affects` may only name features that
   are `done` or `retired`. A feature still in flight is edited directly —
   route back to fdf-brainstorm/plan/execute.
2. **Scaffold**, flags before the slug:
   - `fdf change --affects <group>/<slug>[,…] [<group>/]<slug>`
   - `fdf fix --affects <group>/<slug>[,…] [<group>/]<slug>`

   Group it (`payments/refund-window`) when `changes/` is getting long. The
   group is filing only — nothing ties it to the affected feature's group, and
   nothing checks that it does.
3. **Understand the change** through questions, ONE at a time: what is wrong,
   for whom, what should happen instead, what must not break? Chase ambiguous
   words exactly as fdf-brainstorm does.
4. **Declare the effects** in the document body. This is the part that makes
   the change verifiable, so be precise — names are matched **verbatim**
   against the feature's Gherkin.

   For a `Change`, under `# Scenario changes`, one `## <feature-id>` heading
   per affected feature:
   - `- add: <scenario name>` — must exist in that feature once done.
   - `- modify: <scenario name>` — exists before and after; **name unchanged**,
     steps change.
   - `- remove: <scenario name>` — must not exist once done.

   A rename is `remove:` the old plus `add:` the new. Delete the scaffold's
   TODO lines that do not apply.

   For a `Fix`, under `# Regression cases`, one heading per affected feature
   and one entry per scenario it proves:
   `- <scenario name> — <command, test path, or manual procedure>`.
   Every name must **already exist** in that feature.
5. **For a `Change`, get the design approved.** Present approach, alternatives
   with trade-offs (lead with your recommendation), accepted trade-offs. One
   explicit gate: "Do you approve this design?" Only on yes, write
   `changes/<id>.spec.md` (`## What is being built`, `## Why`,
   `## Design decisions`, `## Alternatives rejected`) and flip to `specified`.
   A `Fix` skips this entirely — it restores behavior already approved.
6. **Plan it if it needs decomposing.** More than a couple of steps → write
   `changes/<id>.plan.md` with `# Tasks` linking every task under
   `changes/<id>/`, flip to `planned`, then work them as fdf-execute does
   (`in-progress` before working, per task). A small change needs no tasks at
   all and goes straight to `done`.
7. **Amend the living documents.** This is the step that keeps the bundle
   true, and F10 enforces it:
   - The feature's **Gherkin** — apply exactly the adds/modifies/removes you
     declared.
   - The feature's **`slug.test.md`** — a case for every scenario (F8), and
     for a `Fix` the regression case you named.
   - The feature's **`slug.surface.md`**, when the interface changed.

   Do **not** touch the feature's `slug.spec.md`, `slug.plan.md`, or tasks.
   Those are the frozen record of how it was first built — the context someone
   needs to judge this change and the next one.
8. **Flip to `done`** and update timestamps. **Gate**: `fdf validate` exit 0.
   F10 refuses a `done` change whose declared effects are not reality.
9. **Log it** — `slug.log.md` on the affected feature (what changed and why),
   and the bundle `LOG.md`.
10. **Context-doc review**, exactly as in fdf-execute: if the work made
    STACK/ARCHITECTURE/SURFACES/INFRA stale, *propose* the edit and wait for
    explicit approval. Never edit one silently.

## Retiring a feature

When a delivered capability is removed from the product, it does not get
deleted — the document records behavior the software once had.

- Write a `Change` with `retires: <feature-id>` and a `# Rationale` section
  saying why it is going and what supersedes it. F10 will not accept a
  `retired` feature without exactly one `done` Change that retires it.
- Set the feature to `status: retired`, and `replaced-by: <feature-id>` when a
  successor exists.
- A `Fix` can never retire anything — removing behavior is deliberate.

## Rules

- Never fork a second feature document for the same capability.
- Never edit a delivered feature's Gherkin outside a Change or Fix: then
  nothing records why, and nothing verified that the code followed.
- Never rewrite the feature's original spec, plan, or tasks.
- Never hand-write a back-link on the feature. `affects:` is the whole link;
  `fdf history <group>/<slug>` computes the trail.
- A `Fix` names only scenarios that already exist. If you need a new one, it
  is a `Change`.
- `fdf validate` exit 0 after every bundle edit.
