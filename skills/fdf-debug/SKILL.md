---
name: fdf-debug
description: Use when something is broken in an FDF project — a bug report, failing test, crash, regression, or unexpected behavior — to find the root cause before any fix and route the repair to a Fix, a Change, a new feature, or in-flight task work.
---

# FDF Debug

Find the root cause first; then let the bundle decide which document may
repair it.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the fdf skills fit together.

This skill ends where the writing starts. It hands a stated root cause to
fdf-change (a `Fix` or a `Change`), fdf-brainstorm (a new feature), or
fdf-execute (a feature still in flight). It never writes the repair itself.

## Two laws

```
NO FIX WITHOUT A ROOT CAUSE.
NO BEHAVIOR CHANGE WITHOUT THE DOCUMENT THAT AUTHORIZES IT.
```

Investigating is not fixing. Read, reproduce, instrument, bisect, revert the
instrumentation — all of that is free and none of it needs a document. The
first line that changes what the software *does* is where the document comes
first.

## The bundle is evidence

Elsewhere you infer the expected behavior. Here it is written down, and that
shortens every phase:

| Question | Where the bundle answers it |
|---|---|
| What is this supposed to do? | the feature's Gherkin `Scenario:` blocks |
| How is it verified? | `<group>/<slug>.test.md` — run that case first |
| Why was it built this way? | `<group>/<slug>.spec.md` (frozen: read it, never edit it) |
| What was built, and where | `<group>/<slug>.plan.md` and the tasks under `<group>/<slug>/` — their `resource:` paths name the code |
| What happened since it shipped? | `fdf history <group>/<slug>`, `<group>/<slug>.log.md`, bundle `LOG.md` |
| What conventions should the code follow? | `ARCHITECTURE.md`, `SURFACES.md`, `STACK.md`, `INFRA.md` |
| Is the bundle itself consistent? | `fdf validate` |

To find the feature that owns a symptom, grep the bundle for the symptom's
vocabulary — and grep it for the file in the stack trace, because a task's
`resource:` line names the code that task produced:

```bash
grep -rn "refund" docs/features --include='*.md'
grep -rn "src/payments/refund.ts" docs/features   # which task built this?
```

## Phase 1 — Reproduce, and read what was promised

1. **Read the error completely** — the whole stack trace, line numbers, exit
   codes. Errors routinely name their own cause.
2. **Reproduce it, and write the command down verbatim.** That command
   becomes the regression case later, so capture it exactly, with its failing
   output. Start from the feature's `slug.test.md` case: if that case still
   passes, the bug is outside what the feature promises — which is itself a
   finding.
3. **Find the owning feature(s)** and note each status. A symptom can cross
   features; that is one document with several `affects:` entries, never one
   document per feature.
4. **Read the scenario** and state the gap in the bundle's own terms:
   - a scenario says X and the software does Y → documented behavior broke;
   - no scenario covers this situation → nobody ever decided it.

   That distinction *is* the routing decision in Phase 3. Do not blur it.
5. **Check what changed.** `fdf history <group>/<slug>` lists every Change and
   Fix that touched the feature; a recent one is a prime suspect, and its
   `# Scenario changes` says exactly what was meant to move. Then git log and
   diff, dependency bumps, config and environment differences.

## Phase 2 — Root cause

1. **Trace backwards to the origin.** The error surfaces where a bad value is
   *used*, not where it was born. Follow it up the call chain — what passed
   this, and what passed that — until you reach the code that created it. Fix
   at the source; the symptom site earns validation at most.
2. **Instrument the boundaries** in a multi-component system (request →
   service → store; CI → build → sign). Log what enters and what leaves each
   component, run once, and let the evidence say which hop breaks. Measuring
   which layer is wrong is cheaper than guessing.
3. **Compare against something that works** — a sibling scenario, the code a
   sibling task produced, the reference implementation. List every
   difference, including the ones that "can't matter".
4. **One hypothesis at a time.** Write it as "X is the cause because Y", make
   the smallest change that would prove it, and run. Wrong → form a *new*
   hypothesis; never stack a second fix on an unproven first.
5. **Say when you don't know.** "I don't understand why X" is progress. A
   confident guess is not.

Stop rules:

- Three failed hypotheses is not a fourth hypothesis — it is a design
  question. Stop, re-read `ARCHITECTURE.md`, and put the question to the user.
- Cannot reproduce it → gather data, don't guess. A bug you cannot reproduce
  is one you cannot prove you fixed. Flakiness is a root cause you have not
  found yet; a rerun is not a diagnosis.
- Remove every temporary probe before handing off. Instrumentation left
  behind is behavior no document covers.

End the phase with one sentence: **"<symptom> happens because <cause>, at
`file:line`."** If you cannot write that sentence, you are still in Phase 2.

## Phase 3 — Route the repair

The root cause says what broke. The bundle says which document may repair it.
Work down; the first row that matches wins:

| What the investigation found | Route |
|---|---|
| Nothing a user can observe changes — build, CI, deps, formatting, pure refactor | Fix it directly, no document. Say plainly that the bundle is untouched and why the work is behavior-neutral. |
| The bundle contradicts itself; `fdf validate` fails | fdf-validate. The code may be fine and the documents may be the drift. |
| The owning feature is `draft`, `specified`, `planned`, or `implementing` | Not post-delivery. Repair it in the in-flight workflow — a task under fdf-execute, or back to fdf-plan / fdf-brainstorm if the design was wrong. Never open a Change or Fix against an undelivered feature; F10 rejects it. |
| A `done`/`retired` feature has a scenario saying otherwise — the code drifted | **Fix**: `fdf fix --affects <group>/<slug>[,…] [<group>/]<slug>` → fdf-change |
| No scenario covers the case, or the scenario itself is what is wrong | **Change**: `fdf change --affects <group>/<slug>[,…] [<group>/]<slug>` → fdf-change |
| The behavior that should exist reads as its own `Feature:` block | New feature → fdf-brainstorm, recording lineage with `depends-on` |
| The root cause is upstream — a dependency, a platform, an external service | Still ours to answer: what should our software do when that happens? That decision is a Change. A version pin with no observable difference is neutral. |

Rows four and five are the ones people get wrong. "It's a bug" does not make
it a Fix. A Fix restores behavior **someone already approved**, so it names
scenarios that already exist, verbatim. If you are about to write a
regression case for a scenario that is not in the Gherkin, it is a Change and
it needs the design gate.

**The document was silent** is the most common finding of all, and it is a
Change: nobody decided what should happen, so someone has to decide now.

Then announce: "Root cause: <one sentence>. That is a
<Fix | Change | feature | task> — using fdf-<skill>."

## Hand off the evidence

The investigation is the raw material for the document, and the scaffolds
have slots waiting for it. Carry over:

- the one-sentence root cause with `file:line` → a Fix's `# Root cause`;
- the verbatim reproduction and its failing output → a Fix's `# Symptom`, and
  the verification in `# Regression cases`;
- the scenario names involved, copied **verbatim** — F8 and F10 match them
  literally;
- every feature the symptom touches → `affects:`;
- what must not break — the neighbouring scenarios whose cases you re-run.

For a Change the same material becomes `## Why` in `changes/<id>.spec.md`,
and the design gate decides what the behavior should become.

The durable artifact of a defect is never the patch — it is the case in the
affected feature's `slug.test.md`. If the work ends with no test that would
have caught this, the bug is not finished.

Record the conclusion in the feature's `slug.log.md`, and the bundle `LOG.md`,
when it is worth the next agent's time. Never edit the feature's frozen
`slug.spec.md`, `slug.plan.md`, or tasks — they record how it was built,
which is the context that made this diagnosis possible.

## Red flags — STOP, you are rationalizing

| Thought | Reality |
|---|---|
| "I can see the fix — patch now, document after" | Code first is the drift FDF exists to stop, and a patch with no document has no regression case, so the bug returns. |
| "It's a bug, so it's a Fix" | Only when a scenario already says otherwise. A case nobody specified is a Change. |
| "The scenario is wrong, I'll correct the Gherkin" | Delivered Gherkin changes only through a Change. Editing it directly erases the record of why it ever said that. |
| "The code is the real spec now" | The document is a promise, not a mirror of the code. Changing the promise is a decision someone makes deliberately. |
| "Just change X and see if it helps" | One written hypothesis, one minimal test — otherwise you cannot tell what worked. |
| "The test is flaky, rerun it" | Flaky means a root cause you have not found. Nondeterminism has a cause. |
| "Emergency — no time for process" | Systematic is faster than thrashing, and the FDF floor for a Fix is one file. |
| "The third fix failed; one more idea" | 3+ failures is an architecture question, not a fourth patch. Stop and ask. |
| "While I'm in here, I'll also…" | Scope past the root cause needs its own document. One cause, one repair. |
| "I'll leave the debug logging in, it's useful" | No document covers it. Remove it, or make it part of the documented work. |
| "No root cause — it's just the environment" | Almost always an unfinished investigation. If it truly is environmental, say what you ruled out, then decide what the software should do about it. |

## Rules

- No proposed fix before the root cause is stated in one sentence.
- No behavior-changing edit before the document that authorizes it exists.
- Reproduce before fixing; verify afterwards with the same command.
- Never weaken a scenario or a test case to make a symptom go away — that is
  the silencing fdf-validate refuses.
- Temporary instrumentation is reverted before the work is called done.
- `fdf validate` exit 0 after every bundle edit.
