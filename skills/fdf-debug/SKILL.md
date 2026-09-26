---
name: fdf-debug
description: Use when something is broken in an FDF project — a bug report, failing test, crash, regression, or unexpected behavior — to find the root cause before any fix and route the repair to a Fix, a Change, a new feature, in-flight task work, or adoption of undocumented code — or file it as a Bug when nobody is repairing it now.
---

# FDF Debug

Find the root cause first; then let the bundle decide which document may
repair it.

New to FDF? Run `fdf spec` for the format rules and `fdf help` for the CLI.
The fdf-help skill explains how the fdf skills fit together.

This skill ends where the writing starts. It hands a stated root cause to
fdf-change (a `Fix` or a `Change`), fdf-brainstorm (a new feature),
fdf-execute (a feature still in flight), or fdf-adopt (code no feature
documents yet) — or, when nobody is repairing it now, files it as a `Bug`. It
never writes the repair itself.

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
| What happened since it was delivered? | `fdf history <group>/<slug>`, `<group>/<slug>.log.md`, bundle `LOG.md` |
| What conventions should the code follow? | `ARCHITECTURE.md`, `SURFACES.md`, `STACK.md`, `INFRA.md` |
| How was this mechanism supposed to be done? | the practice whose `applies-to` covers the file in the trace — `# Rules` is binding, and code that ignores one is a common root cause |
| Is this already known? | `fdf bug --open` and `fdf debt --open` — an entry whose `resource` names the file in the trace, or whose `affects` names the feature, has already diagnosed this, and says why it was left |
| What is this thing actually called? | `DOMAIN.md` — grep the bundle for the canonical term, not the word in the bug report; a report usually names what the reporter saw on screen, and the surface is allowed to use a word the lexicon bans |
| Is the bundle itself consistent? | `fdf validate` |

To find the feature that owns a symptom, grep the bundle for the symptom's
vocabulary — in `DOMAIN.md`'s canonical terms, since the report may use a word
the lexicon bans and the documents therefore never use — and grep it for the file in the stack trace, because a task's
`resource:` line names the code that task produced:

```bash
grep -rn "refund" docs/fdf --include='*.md'
grep -rn "src/payments/refund.ts" docs/fdf   # which task built this?
```

The report's word maps to the term through `instead-of`; the surface that
showed the word does not. A UI label or locale string that says "store" for a
Venue is not a finding — the lexicon governs the bundle and the code's
identifiers, not what a person reads on screen — so "the label uses a banned
word" is never a root cause and never routes to a Fix.

## Phase 1 — Reproduce, and read what was promised

1. **Read the error completely** — the whole stack trace, line numbers, exit
   codes. Errors routinely name their own cause.
2. **Reproduce it, and write the command down verbatim.** That command
   becomes the regression case later, so capture it exactly, with its failing
   output. Start from the feature's `slug.test.md` case. If it still passes,
   either the bug is outside what the scenario promises, or the case is
   weaker than its scenario — a boundary it never exercised. Read the scenario
   to tell which: the first is a Change, the second a Fix whose regression
   case strengthens the test. An adopted map entry has no scenarios and no
   `slug.test.md`: reproduce from the code, and expect the route to be a
   Change — nobody has written the promise down yet.
3. **Find the owning feature(s)** and note each status. A symptom can cross
   features; that is one document with several `affects:` entries, never one
   document per feature. An `adopted` feature is delivered like a `done` one.
   **No feature owns the code at all** — common in a codebase that predates
   its bundle — is a finding too: the repair will route through fdf-adopt.
4. **Check both registers** — `fdf bug --open` and `fdf debt --open` (below
   v0.7 there is no bug register, and `fdf bug` says so). A bug
   or debt whose `resource` names a file in the trace, or whose `affects`
   names the feature, has already diagnosed this and says why it was left.
   Then the conversation changes: the defect was known, and the question is
   whether that decision still holds — not what is broken. Amend the entry
   with what you learned rather than filing a second one.
5. **Read the scenario** and state the gap in the bundle's own terms:
   - a scenario says X and the software does Y → documented behavior broke;
   - no scenario covers this situation → nobody ever decided it.

   That distinction *is* the routing decision in Phase 3. Do not blur it.
6. **Check what changed.** `fdf history <group>/<slug>` lists every Change and
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
| Nothing a user can observe changes — build, CI, deps, formatting, pure refactor | Repair it directly, no feature document. Say plainly that the bundle is untouched and why the work is behavior-neutral — and if the repair changes what `STACK.md` or `INFRA.md` says (a version, a build step), propose that Context edit. |
| The bundle contradicts itself; `fdf validate` fails | fdf-validate. The code may be fine and the documents may be the drift. |
| The owning feature is `draft`, `specified`, `planned`, or `implementing` | Not post-delivery. Repair it in the in-flight workflow — a task under fdf-execute; or, if the design itself was wrong, amend the Gherkin and `slug.spec.md` directly with the user's approval (the feature is still in flight) and re-plan with fdf-plan. Never open a Change or Fix against an undelivered feature; F10 rejects it. |
| No feature documents the code at all | **Adopt first**: map the capability (fdf-adopt, one map entry), then route by its new status — usually the next row but one: nobody wrote the promise down, so a Change adds it |
| A `done`/`adopted`/`retired` feature has a scenario saying otherwise — the code drifted | **Fix**: `fdf fix --affects <group>/<slug>[,…] [<group>/]<slug>` → fdf-change |
| No scenario covers the case, or the scenario itself is what is wrong | **Change**: `fdf change --affects <group>/<slug>[,…] [<group>/]<slug>` → fdf-change |
| The behavior that should exist reads as its own `Feature:` block | New feature → fdf-brainstorm, naming the feature it builds on in `depends-on` |
| The root cause is upstream — a dependency, a platform, an external service | Still ours to answer: what should our software do when that happens? That decision is a Change. A version pin with no observable difference is neutral. |

Two findings are not routes of their own. They ride along with whichever row
matched:

- **The code ignored a practice that governs its path.** Say so in the
  handoff: it is the same class of defect the practice exists to prevent. If
  the practice itself is wrong, that is a practice change and needs the
  user's approval, never a quiet edit.
- **An open debt already names this gap.** It is not a new finding — say so.
  Closing the gap for good means flipping the debt to `resolved` with a
  `# Resolution`; fixing only the site that surfaced means the debt stays
  open with its `resource` narrowed. Never leave the register saying
  something the tree no longer does.

Rows four and five are the ones people get wrong. "It's a bug" does not make
it a Fix. A Fix restores behavior **someone already approved**, so it names
scenarios that already exist, verbatim. If you are about to write a
regression case for a scenario that is not in the Gherkin, it is a Change and
it needs the design gate.

**The document was silent** is the most common finding of all, and it is a
Change: nobody decided what should happen, so someone has to decide now.

Then announce: "Root cause: <one sentence>. That is a
<Fix | Change | feature | task> — using fdf-<skill>." When nobody is repairing
it now: "Root cause: <one sentence>. Its repair will be a <Fix | Change>;
filing it as a Bug."

## Not repairing it now — file a Bug

Some defects are found, diagnosed or half-diagnosed, and not repaired in this
session: the fix waits on a decision, belongs to someone else's work, or is
simply deferred. Everything the investigation learned goes on the register, so
the next person starts from it instead of from zero:

```bash
fdf bug --affects <group>/<slug>[,…] [<group>/]<slug>
```

On a bundle pinned below v0.7, `fdf bug` refuses, because the bug register
arrived in v0.7: propose `fdf migrate`, or file the defect as a debt until
then.

- `# Symptom` — the verbatim reproduction and its failing output; or, for a
  defect found by reading, the code path and the input that reaches it. Say
  which: an unreproduced defect is still a bug, but the reader needs to know.
  When the reproduction is throwaway code — a probe test you must not leave
  failing in the suite — delete the file, and paste its source and its
  failing output here: it is the regression case the repair will start from.
- `# Expected` — what should happen. When a scenario already says so, name it
  under `# Violates` (verbatim — F14 checks it): the repair is a Fix. When no
  document decides it, say so and write the open question: the repair is a
  Change, and someone has to answer it first.
- `# Root cause` — the one sentence, if you got that far.
- `resource` — the paths in the trace, so work that touches them finds this.

**A bug or a debt?** If someone could observe the software doing something
wrong, it is a bug — even when the root cause is a practice the code ignores.
A gap nothing observable shows yet (unfinished work, a rule not followed
everywhere, missing tests, dead code) is a debt. The line matters because a
debt is often paid with plain code work, and a bug never is: its repair
changes what the software does, so it goes through a Fix or Change and leaves
a regression case behind.

When the repair comes, `fdf fix --from bugs/<id> …` (or `fdf change --from`)
copies this analysis into the Fix or Change and writes `resolves`; once that
work is `done`, the bug is flipped to `resolved`.

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

When the repair is not happening now, all of this goes into a `Bug` instead
(see *Not repairing it now*), and the Fix or Change that eventually repairs it
takes it over with `--from`.

Record the conclusion in the feature's log when it is worth the next agent's
time — a decision, or a dead end the next investigator should not repeat:
`fdf log <group>/<slug> "**Investigated**: …"`. A filed bug needs no log line: `fdf history` lists it from its
`affects`, and a hand-written pointer to it is a back-link that drifts. Never edit the feature's frozen
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
| "It's undocumented code, so no document is needed to fix it" | Undocumented is not unowned. Map the capability (fdf-adopt), then a Change adds the promise — otherwise the fix leaves no scenario and no test, and the defect comes back unnoticed. |
| "We're not fixing it now; I'll mention it in the summary" | A defect known only to a chat log is found again from zero. File it: `fdf bug`, with the reproduction and the expected behavior. |
| "I'll file it as a debt" | The software does something wrong, so it is a bug. A debt's repair can be plain code work; a bug's never is. |
| "No root cause — it's just the environment" | Almost always an unfinished investigation. If it truly is environmental, say what you ruled out, then decide what the software should do about it. |

## Rules

- No proposed fix before the root cause is stated in one sentence.
- No behavior-changing edit before the document that authorizes it exists.
- Reproduce before fixing; verify afterwards with the same command.
- Never weaken a scenario or a test case to make a symptom go away — that is
  the silencing fdf-validate refuses.
- Temporary instrumentation is reverted before the work is called done.
- `fdf validate` exit 0 after every bundle edit.
