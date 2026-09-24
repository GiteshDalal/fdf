---
name: fdf-change
description: Use when a delivered (done, adopted or retired) FDF feature needs to change, a diagnosed defect in it needs fixing — including a bug on the register — or it is being retired, including work spanning several features. An undiagnosed bug goes to fdf-debug first. Not for features still in draft/specified/planned/implementing.
---

# FDF Change

Amend a delivered feature without forking it or letting the bundle drift.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the skills fit together.

**Read the Context docs first** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md`. A change lands in a system that
already exists; ground it in the documented stack and conventions, and flag
explicitly when the change would depart from them. Write scenario names and
declarations in the vocabulary `DOMAIN.md` fixes — and read the practices
whose `applies-to` covers the code this will touch, since they bind this work
exactly as they bind a new feature.

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

Do not settle this from the report's wording. A defect's route follows from
its **root cause**, so if nobody has diagnosed it yet, run **fdf-debug**
first and come back with the cause in one sentence — that is also what the
Fix body's `# Symptom` and `# Root cause` sections are waiting for.

## Process

1. **Confirm the feature is delivered.** `affects` may only name features that
   are `done`, `adopted` or `retired`. A feature still in flight is edited
   directly — route back to fdf-brainstorm/plan/execute. Code no feature
   documents is adopted first (fdf-adopt), then changed here. For an `adopted`
   feature, every scenario this work will `modify:` or `remove:` must exist
   first — backfill them with fdf-adopt; one it only `add:`s needs none.
2. **Scaffold**, flags before the slug:
   - `fdf change --affects <group>/<slug>[,…] [<group>/]<slug>`
   - `fdf fix --affects <group>/<slug>[,…] [<group>/]<slug>`

   When the work repairs a bug on the register, scaffold **from** it:
   `fdf fix --from bugs/<id> [<group>/]<slug>` (or `fdf change --from …`). It
   takes over the bug's analysis as this document's permanent record — its
   `# Symptom` and `# Root cause`, its `# Violates` scenarios as regression
   cases — defaults `affects` to the bug's, and writes `resolves: bugs/<id>`.
   A bug is never repaired in place; this is how it is repaired.

   Group it (`payments/refund-window`, usually after the affected feature's
   group) once `changes/` holds more than about ten documents; below that,
   flat is fine. The group is filing only — nothing ties it to the affected
   feature's group, and nothing checks that it does. Below, `<id>` is the document's path under
   `changes/` without `.md` — `refund-window` or `payments/refund-window`.
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

   A rename is `remove:` the old plus `add:` the new. A rename whose only
   reason is a word `DOMAIN.md` bans is not a Change at all: it is a lexicon
   fix, made in place across the bundle — `fdf lexicon --term <Term> --fix`
   renames a scenario everywhere it is a join (fdf-validate, F12). Delete
   every scaffold line that starts `TODO —` once it is answered or does not
   apply; `fdf validate` warns about any left behind.

   For a `Fix`, under `# Regression cases`, one heading per affected feature
   and one entry per scenario it proves:
   `- <scenario name> — <command, test path, or manual procedure>`.
   Every name must **already exist** in that feature. Fill `# Symptom` with
   the verbatim reproduction and its failing output, and `# Root cause` with
   why the code diverged — both come straight out of the fdf-debug
   investigation; neither is a guess written after the patch.
5. **For a `Change`, get the design approved.** Present approach, alternatives
   with trade-offs (lead with your recommendation), accepted trade-offs. One
   explicit gate: "Do you approve this design?" Only on yes, write
   `changes/<id>.spec.md` (`## What is being built`, `## Why`,
   `## Design decisions`, `## Alternatives rejected`) and flip to `specified`.
   A `Fix` skips this entirely — it restores behavior already approved.
6. **Plan it if it needs decomposing.** More than a couple of steps → write
   `changes/<id>.plan.md` with `# Tasks` linking every task under
   `changes/<id>/` (plan and tasks together — a task directory without its
   plan fails F6), flip to `planned`, then work them as fdf-execute does
   (`in-progress` before working, per task). A small change needs no tasks at
   all: it skips to step 7.
7. **Do the work and prove it.** Implement the change — through its tasks, or
   directly when there are none. Then run the affected features'
   `slug.test.md` cases this work touches, plus their neighbours, and for a
   `Fix` the regression command, which must now pass where `# Symptom` showed
   it failing. Report each command with its actual output; a case you did not
   run keeps the document short of `done`.
8. **Amend the living documents.** This is the step that keeps the bundle
   true, and F10 enforces it:
   - The feature's **Gherkin** — apply exactly the adds/modifies/removes you
     declared. Write new and changed steps around the concept, not the label
     a surface shows: the literal label belongs in the step definition or
     `slug.surface.md`, and wording that is itself under test (an error
     message, a URL, a button label) is named as an outcome in the scenario
     and pinned in `slug.test.md`. Never reword the label to suit
     `DOMAIN.md` — the lexicon stops at the surface.
   - The feature's **`slug.test.md`** — a `## <scenario name>` case for every
     scenario, the name exactly as in the Gherkin (F8), and for a `Fix` a case
     for each regression you named (F10). Drop the case of a removed
     scenario; a case that names no scenario draws a warning.
   - The feature's **`slug.surface.md`**, when the work changed or added an
     interface: an endpoint's codes, a screen's copy, a flag, an event's
     fields. It describes the interfaces as they are today, so the old shape
     is replaced, not kept beside the new one. A feature that has an interface
     but no surface document gets one now.

   Do **not** touch the feature's `slug.spec.md`, `slug.plan.md`, or tasks.
   Those are the frozen record of how it was first built — the context someone
   needs to judge this change and the next one.
9. **Flip to `done`** — with tasks, the last task and the document in the
   same edit, as fdf-execute does — and set their timestamps to now, in UTC.
   If the work closes an open debt, flip it to `resolved` with a
   `# Resolution` now too. Every bug this document `resolves` is flipped to
   `resolved` in the same edit, its `# Resolution` naming this document — F10
   refuses a done repair whose bug still reads as open.
   **Gate**: `fdf validate` exit 0. F10 refuses a `done` change whose declared
   effects are not reality.
10. **Log it** in each affected feature's own log, one entry naming this
    document, so the feature's log tells its whole life:
    `fdf log <group>/<slug> "**Changed**: [<id>](/changes/<id>.md) <what now differs>."`
    (`**Fixed**` for a Fix). Decisions taken while doing the work go in this
    document's own log (`fdf log changes/<id> "**Decision**: …"`). Both are
    feature-scoped, so neither goes in the root `LOG.md`.
11. **Project-document review**, exactly as in fdf-execute — all four
    questions. Did this make a Context document stale (including a term in
    DOMAIN.md)? Did the code diverge from a practice that governs its paths,
    and is that a defect or an approved `# Exceptions` entry? Did it establish
    a mechanism a second feature now repeats, which should become a practice?
    Did it knowingly leave something undone — `fdf debt [<group>/]<slug>` —
    or find a defect it is not repairing — `fdf bug [<group>/]<slug>`? When
    the work spread a gap an open debt already names to new code, amend that
    debt's `# Gap` and `resource` instead of filing a second one.
    *Propose* each edit and wait for explicit approval. Never edit a Context
    document or a practice silently. See *Practices and a change* below —
    post-delivery work is where practice drift actually surfaces.

## Practices and a change

Post-delivery work is where the gap between what a practice says and what the
code does becomes visible, so three cases come up here that do not come up
during a feature.

**A Fix whose root cause was an unwritten rule.** The code diverged, and when
you look, nothing told it not to. That is a practice waiting to be written:
the same defect will arrive again in the next handler. Propose one, with the
Fix as its evidence — this is the strongest kind of practice, because it costs
nothing to argue for. Fixing the one call site the bug surfaced in while the
other eleven still do it the old way is a **debt**, not a finished job: file it
with the paths, so the next reader sees a known gap instead of an inconsistency
they have to re-derive.

**A Fix whose root cause was a written rule nobody followed.** The practice
exists and was ignored. Do not amend the practice to match the code; fix the
code, and consider whether the `# Rules` line was too vague to follow —
sharpening it is an edit worth proposing.

**A Change that alters the mechanism itself.** When the new behavior means the
project now does something a *different* way, the practice must change too. It
is a **living** document, so amend it in place — the practice always describes
today — and let the Change document record why. Two shapes:

- *The mechanism is refined* — edit the `# Rules`, log it in the practice's
  own log (`fdf log practices/<slug> "**Amended**: …"`).
- *The mechanism is replaced* — set the old practice to
  `status: superseded` with `superseded-by: practices/<new-slug>` and write
  the replacement. Never delete it: code in the tree still follows it, and the
  old document is what explains that code. F11 requires the `superseded-by`
  target to exist. The code still on the old mechanism is a debt — file it,
  because "superseded" describes the document, not the tree.

A practice is never forked per change, and a change never gets its own copy of
one. One practice per mechanism, amended forever.

Both require explicit user approval before you write. A practice binds all
future code, so changing one is a bigger decision than the change that
prompted it.

## Retiring a feature

When a delivered capability is removed from the product, it does not get
deleted — the document records behavior the software once had.

- Write a `Change` with `retires: <feature-id>` and a `# Rationale` section
  saying why it is going and what supersedes it. F10 will not accept a
  `retired` feature without exactly one `done` Change that retires it.
- The feature is in `affects`, so the Change still carries
  `# Scenario changes` with a `## <feature-id>` heading for it (F10). Leave
  the heading empty: the retired feature keeps its Gherkin as the record of
  what it did.
- Set the feature to `status: retired`, and `replaced-by: <feature-id>` when a
  successor exists.
- Declare `retires:` from the start, like the rest of the Change's effects.
  While the Change is not `done`, the feature it retires is still delivered
  (`done` or `adopted`), and it must be in `affects`. Land the retirement in
  one edit — flip the Change to `done` and the feature to `retired` — because
  a done Change's `retires` names a `retired` feature and a `retired` feature
  needs a done Change retiring it (F10). Validate after that edit.
- A `Fix` can never retire anything — removing behavior is deliberate.
- Retirement is terminal. A capability that comes back is a new feature
  (fdf-brainstorm), which may name the retired one in `depends-on`; the retired
  document is never flipped back to `done`.

## Rules

- Never fork a second feature document for the same capability.
- Never edit a delivered feature's Gherkin outside a Change or Fix: then
  nothing records why, and nothing verified that the code followed. A
  lexicon fix is the one exception — it changes words, not behavior — and so
  is backfilling an adopted feature: adding a scenario that its code already
  passes (fdf-adopt).
- Never rewrite the feature's original spec, plan, or tasks, except for a
  maintenance edit — a lexicon fix, a reference repair after `fdf mv`, a path
  repair after the code moved.
- Never resolve a bug in place. Its repair is this document, which names it in
  `resolves`.
- Never fork a practice per change: amend the one that exists, or supersede
  it. A practice describes today, and there is only one today.
- Never hand-write a back-link on the feature. `affects:` is the whole link;
  `fdf history <group>/<slug>` computes the trail.
- A `Fix` names only scenarios that already exist. If you need a new one, it
  is a `Change`.
- `fdf validate` exit 0 after every bundle edit.
