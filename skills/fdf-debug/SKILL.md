---
name: fdf-debug
description: Use when something is broken in an FDF project — a bug report, a failing test, a crash, a regression, or unexpected behavior — before any fix is written, including when the fix looks obvious or nobody will repair it now.
---

# FDF Debug

Find the root cause first; then let the bundle decide which document may
repair it.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

This skill ends where the writing starts. It hands a stated root cause to
fdf-change (a `Fix` or a `Change`), fdf-brainstorm (a new feature),
fdf-execute (a feature still being built), or fdf-adopt (code no feature
documents yet) — or, when nobody is repairing it now, files it as a `Bug`. It
never writes the repair itself.

## Two laws

```
NO FIX WITHOUT A ROOT CAUSE.
NO BEHAVIOR CHANGE WITHOUT THE DOCUMENT THAT AUTHORIZES IT.
```

Investigating is not fixing. Reading, reproducing, adding a temporary probe,
bisecting, and trying an experimental edit that you revert straight after the
run are all free, and none of them needs a document. The first change you
**keep** that alters what the software *does* is where the document comes
first.

## Mechanics

- `<feature-id>` and `<bug-id>` are full IDs, such as
  `features/payments/instant-refunds` and `bugs/refund-split-capture`; paths
  are relative to the bundle root, `docs/fdf/`.
- A `timestamp:` you set is the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run
  at the moment of the edit — never a time you estimate, round or reuse, and
  never your local time with `Z` added. A restored document keeps its old
  one; a malformed one already committed is repaired by fdf-validate's F1,
  never set to now.
- Log with single quotes — `fdf log <feature-id> '**Investigated**: …'` —
  because inside double quotes the shell runs anything between backticks. An
  apostrophe would end the quotes: write ’ instead.
- **Probes** — instrumentation, scratch scripts, a replay's scratch state and
  the binary built for it, trial edits — are removed before you file or hand
  off. A **failing reproduction test is not a probe**: when a
  Fix or Change will follow, keep it — it becomes that document's regression
  test: fold it into the scenario's existing test, or keep it and name it in
  the Fix's `# Regression cases`. Delete it only when you file a Bug instead,
  after pasting its source and output into the bug's `# Symptom`.
- **This skill never repairs the code.** Probes, reverted trial edits and a
  failing reproduction test are investigation; the repair waits for the
  route Phase 3 picks — for a Fix or Change, fdf-change's step 7, once the
  document exists.

## The bundle is evidence

Elsewhere you infer the expected behavior. Here it is written down:

| Question | Where the bundle answers it |
|---|---|
| What is this supposed to do? | the feature's Gherkin `Scenario:` blocks, and `<feature-id>.surface.md` for interface details |
| How is it verified? | `<feature-id>.test.md` — run that case (Phase 1, step 5) |
| Why was it built this way? | `<feature-id>.spec.md` (frozen: read it, never edit it) |
| What was built, and where? | `<feature-id>.plan.md` and the tasks under `<feature-id>/` — their `resource:` paths name the code |
| What happened since delivery? | `fdf history <feature-id>` (its Changes, Fixes and known bugs), `<feature-id>.log.md`, the root `LOG.md` |
| What conventions should the code follow? | `ARCHITECTURE.md`, `SURFACES.md`, `STACK.md`, `INFRA.md` |
| How was this mechanism supposed to be done? | the practice whose `applies-to` covers the file in the trace — its `# Rules` bind the code, and code that ignores one is a common root cause |
| Is this already known? | the bugs and debts whose `resource` or `affects` names this code or feature (Phase 1, step 4) |
| What is this thing actually called? | `DOMAIN.md` — search for the canonical term, not the word in the report |
| Is the bundle itself consistent? | `fdf validate` |

**Find the feature that owns the symptom** by searching the bundle for the
symptom's vocabulary — in `DOMAIN.md`'s canonical terms, since a report
usually names what the reporter saw on screen, and the surface may use a word
the lexicon bans — and for the file in the stack trace, because a task's
`resource:` names the code it produced:

```bash
grep -rln 'refund' docs/fdf/features --include='*.md'
grep -rn 'src/payments/refund.ts' docs/fdf     # which task built this file?
grep -rn 'src/payments' docs/fdf               # a resource may name its directory
```

A UI label or locale string that says "store" for a Venue is not a finding:
the lexicon governs the bundle and the code's identifiers, not what a person
reads on screen. "The label uses a banned word" is never a root cause.

## Phase 1 — Reproduce, and read what was promised

0. **Came here straight, not through the fdf-help skill?** Do its steps 1
   and 6 first: run `fdf validate` (a `FAIL` goes to fdf-validate before
   anything else), and read the practices, debts and bugs whose paths overlap
   the code in the report.
1. **Read the error completely** — the whole stack trace, line numbers, exit
   codes. Errors routinely name their own cause.
2. **Reproduce it, and write the command down verbatim**, with its failing
   output: that command becomes the regression case later. A reported defect
   happened, so it can be run: build the program and replay the report
   against scratch state in a directory made for it (`mktemp -d`), outside
   the project — delete it, with any binary you built, before you file or
   hand off — or write a failing test.
   Reading the code shows you a cause; it is not a reproduction. A replay
   proves the defect but leaves no test: step 5 names the test a Fix will
   strengthen. An adopted
   map entry has no scenarios and no `slug.test.md`: reproduce from the code,
   and expect a Change — nobody wrote the promise down.
3. **Find the owning feature(s)** and note each one's status. A feature owns
   the symptom when the behavior that goes wrong is part of what it does — a
   scenario, or its `Feature:` block when no scenario covers the case; a
   feature whose `resource` names the same file for another capability does
   not. A symptom can
   cross features: that is one document with several `affects:` entries, never
   one document per feature. An `adopted` feature is delivered, like a `done`
   one. **No feature owns the code at all** — common in a codebase that
   predates its bundle — is a finding too: the repair will go through fdf-adopt.
4. **Check the registers.** `fdf history <feature-id>` lists the bugs that
   name the feature, with their status. For the code itself:

   ```bash
   grep -rn -A3 -e '^resource:' -e '^affects:' docs/fdf/bugs docs/fdf/debts
   ```

   (A bare `resource:` lists its paths on the `- ` lines below it.)

   A bug or debt whose `resource` is the same as, a parent of, or inside a file
   in the trace — or whose `affects` names the feature — may already describe
   this defect: read it. When its `# Symptom` or `# Gap` is this defect, it
   was known, and says why it was left; the question is then whether that
   decision still holds. Amend that entry with what you learned rather than
   filing a second one; and when the repair comes, scaffold it **from** that
   bug (`--from <bug-id>`, Phase 3). An entry about a different defect is
   left as it is.
5. **Run the feature's `slug.test.md` case, and compare it with its
   scenario.** Open the test the case names and compare its setup with the
   scenario's `Given` steps, value by value. A value the scenario states that
   the test never uses — an amount of exactly zero, the last day of a window —
   makes the case weaker than its scenario: the Fix will strengthen that test
   so it tries that input. If the case is as strong as its scenario and still
   passes, the bug is outside what the scenario promises: a Change.
6. **Read the scenario** and state the gap in the bundle's own terms:
   - a scenario says X, and the software does Y → documented behavior broke;
   - no scenario covers this situation → nobody ever decided it.

   That distinction *is* the routing decision in Phase 3. Do not blur it.
7. **Check what changed.** `fdf history <feature-id>` lists every Change and
   Fix that touched the feature; read the `timestamp:` line of each (the
   listing shows none) and compare it with when the symptom started. A recent one is a prime suspect, and its
   `# Scenario changes` says exactly what was meant to move. Then the git log
   and diff, dependency bumps, config and environment differences.

## Phase 2 — Root cause

1. **Trace backwards to the origin.** An error surfaces where a bad value is
   *used*, not where it was born. Follow it up the call chain — what passed
   this, and what passed that — until you reach the code that created it.
   Repair at the source; the symptom site earns validation at most.
2. **Instrument the boundaries** in a multi-component system (request →
   service → store; CI → build → sign). Log what enters and leaves each
   component, run once, and let the evidence say which hop breaks.
3. **Compare against something that works** — a sibling scenario, the code a
   sibling task produced, a reference implementation. List every difference,
   including the ones that "can't matter".
4. **One hypothesis at a time.** Write it as "X is the cause because Y", run
   the smallest experiment that would prove it — a probe, a failing test, or a
   trial edit you revert right after the run — and look at the result. Wrong →
   revert, and form a *new* hypothesis; never stack a second change on an
   unproven first.
5. **Say when you don't know.** "I don't understand why X" is progress. A
   confident guess is not.

Stop rules:

- Three failed hypotheses is not a fourth hypothesis — it is a design
  question. Stop, re-read `ARCHITECTURE.md`, and put the question to the user.
- Cannot reproduce it → gather data, don't guess. A bug you cannot reproduce
  is one you cannot prove you fixed. Flakiness is a root cause you have not
  found yet; a rerun is not a diagnosis.
- Remove every probe and revert every trial edit before handing off.
  Instrumentation left behind is behavior no document covers.

Before you write the sentence below, run `git log -L <line>,<line>:<file>`
on the line it names: a commit that changed that line is part of the root
cause — name it. An empty
`fdf history` never dates a defect.

End the phase with one sentence: **"<symptom> happens because <cause>, at
`<source file>:<line>`."** — a line of the code or of its build, CI or
dependency configuration, never a bundle document. That no scenario covers
the case is Phase 1 step 6's routing finding, never the root cause. If you
cannot write that sentence, you are still in Phase 2.

Then **stop and check `git diff`**: apart from a failing reproduction test
(and, when fdf-execute sent you here, the task's own work), no edit repairs
the bug. Revert any that does, however small: the repair comes back through
the route Phase 3 picks.

## Phase 3 — Route the repair

The root cause says what broke. The bundle says which document may repair it.
Go down the table; **the first row that matches wins**:

| # | What the investigation found | Route |
|---|---|---|
| 1 | Nothing a user or another system can observe changes — the build, CI, dependencies, formatting, a pure refactor | Repair it directly, with no feature document. Say plainly that the bundle is untouched and why the work changes no behavior. If the repair makes `STACK.md` or `INFRA.md` untrue (a version, a build step), propose that Context edit. |
| 2 | The bundle contradicts itself; `fdf validate` fails | **fdf-validate**. The code may be fine and the documents may be what drifted. |
| 3 | The owning feature is `draft`, `specified`, `planned` or `implementing` | Not post-delivery. Repair it inside that feature's own work: a task under **fdf-execute**; or, if the design itself was wrong, amend its Gherkin and `slug.spec.md` directly with the user's approval (the feature is still in flight) and adjust its plan. Never open a Change or Fix against an undelivered feature — F10 rejects it. |
| 4 | No feature documents the code at all | **Adopt first**: map the capability (**fdf-adopt**, one map entry), then go down this table again with its new status, `adopted`. Nobody wrote the promise down, so it usually ends at row 7 — a Change that adds it. |
| 5 | The behavior that should exist reads as its own `Feature:` block — its own As a / I want / So that | A new feature → **fdf-brainstorm**, naming the feature it builds on in `depends-on`. |
| 6 | A `done` or `adopted` feature has a scenario that says otherwise — the code drifted | A **Fix** → **fdf-change**. Scaffold it before you touch the code: `fdf fix --affects <feature-id>[,…] [<group>/…]<slug>`, or `fdf fix --from <bug-id> …` when the bug is already filed. |
| 7 | No scenario covers the case, or the scenario itself is wrong | A **Change** → **fdf-change**: `fdf change --affects <feature-id>[,…] [<group>/…]<slug>`, or `fdf change --from <bug-id> …` when the bug is already filed. It needs the design gate. |
| 8 | The root cause is upstream — a dependency, a platform, an external service | Still ours to answer: what should our software do when that happens? That decision is a Change (row 7). A version pin with no observable difference is row 1. |

Two findings are not routes of their own; they ride along with whichever row
matched:

- **The code ignored a practice that governs its path.** Say so in the
  handoff: it is the class of defect the practice exists to prevent. If the
  practice itself is wrong, that is a practice change and needs the user's
  approval, never a quiet edit.
- **An open bug or debt already names this.** It is not a new finding — say
  so. For a delivered feature (rows 6 and 7), scaffold the repair from the
  filed bug (`--from <bug-id>`), so that it names the bug in `resolves`. In a
  feature still being built (row 3), the task that repairs it closes it: set
  the bug to `resolved`, with a `# Resolution` naming the task. For a debt: closing the gap for good means flipping it to
  `resolved` with a `# Resolution`; repairing only the site that surfaced
  means the debt stays open with its `resource` narrowed. Never leave the
  register saying something the tree no longer does.

Rows 6 and 7 are the ones people get wrong. "It's a bug" does not make it a
Fix. A Fix restores behavior **someone already approved**, so it names
scenarios that already exist, verbatim. If you are about to write a regression
case for a scenario that is not in the Gherkin, it is a Change, and it needs
the design gate. **The document was silent** is the most common finding of all,
and it is a Change: nobody decided what should happen, so someone has to
decide now.

Then announce: "Root cause: <one sentence>. That is a
<Fix | Change | feature | task> — using fdf-<skill>." When nobody is repairing
it now: "Root cause: <one sentence>. Its repair will be a <Fix | Change>;
filing it as a Bug."

Then **open the skill the row names and carry on there**: for a Fix or a
Change, fdf-change from its step 2 — its questions (a Fix asks only what this
investigation lacks), the scaffold and its clean-up (step 3), the regression
case in `slug.test.md` (step 8), the feature's log (step 10) and the
project-document review (step 11) are all there, and none is optional because
the fix looked small.

## Not repairing it now — file a Bug

Some defects are diagnosed, or half-diagnosed, and not repaired in this
session: the fix waits on a decision, belongs to someone else's work, or is
deferred. Everything the investigation learned goes on the register, so the
next person starts from it instead of from zero.

1. **Finish Phase 1 first**: reproduce it (step 2) and check the registers
   (step 4). A report alone is not a reproduction, and neither is reading the
   code. An entry that already names this defect is amended, not filed
   twice. Only a defect that truly cannot run here — it needs production data,
   hardware, or a service you cannot reach — is filed unreproduced, and its
   `# Symptom` then says what you tried and why it did not run.
2. **File it**:

   ```bash
   fdf bug --affects <feature-id>[,…] --resource <path>[,…] [<group>/…]<slug>
   ```

   `--affects` names the features that own the symptom (Phase 1, step 3). `--resource` names the paths in the trace,
   so work that touches them finds the bug. Leave out `--affects` only when no
   feature documents the code.
3. **Fill it in**, replacing every `TODO`. Keep the `timestamp` that `fdf bug`
   wrote.
   - `description:` — one sentence on what the software does wrong.
   - `# Symptom` — the commands you ran, verbatim, their failing output, and
     the word `Reproduced.` When the reproduction was throwaway code, such as
     a probe test, delete the file and paste its source and output here: it is
     the regression case the repair will start from. Write only what you ran
     and saw — never a date or a circumstance the report did not give you.
     (A defect that truly cannot run here: what you tried, and why it did not
     run.)
   - `# Expected` — what should happen. If the user has said, write their
     answer. When no scenario decides it, first write this sentence:
     `No scenario decides this, so its repair is a Change.`
     Then write the user's answer, or the open question someone has to answer
     first.
   - `# Violates` — the scaffold holds an example of it inside an HTML
     comment, from `<!--` to `-->`. When a scenario of a delivered feature
     already says what should happen, replace that whole comment with:

     ```markdown
     # Violates

     ## <feature-id>

     - <the scenario's name, copied exactly>
     ```

     — the heading line is `# Violates` and nothing else, one `## <feature-id>`
     per feature in `affects`, and F14 checks each name. The repair is then a
     Fix. When no scenario covers the case, or the feature is still being
     built, delete the whole comment, from `<!--` to `-->`: the bug then has
     no `# Violates` heading.
   - `# Root cause` — the Phase 2 sentence, with the code's `<file>:<line>`.
     If Phase 2 did not finish, write `Not found yet:` and what you ruled out.
   - `# Cost` — what the defect costs while it stays; optional.
   - Replace the `TODO.` at the end of the listing line `fdf bug` added to the
     `INDEX.md` beside it with the description.
4. **Remove every probe**, and run `fdf validate`.

A filed bug needs no log line: `fdf history` lists it from its `affects`, and
a hand-written pointer to it is a back-link that drifts.

**A bug or a debt?** If someone could observe the software doing something
wrong, it is a bug — even when the root cause is a practice the code ignores.
A gap nothing observable shows yet (unfinished work, a rule not followed
everywhere, missing tests, dead code) is a debt. The line matters because a
debt is often paid with plain code work, and a bug never is: its repair
changes what the software does, so it goes through a Fix or Change and leaves
a regression case behind.

When the repair comes, `fdf fix --from <bug-id> …` (the bug cites a scenario
under `# Violates`) or `fdf change --from <bug-id> …` (it cites none) carries
this analysis into the Fix or Change and writes `resolves`; once that work is
`done`, the bug is flipped to `resolved`. In a feature still being built, the
task that repairs the bug closes it instead (`resolved`, with a
`# Resolution` naming the task).

## Hand off the evidence

The investigation is the raw material for the document, and the scaffolds
have slots waiting for it. Carry over:

- the one-sentence root cause with `<file>:<line>` → a Fix's `# Root cause`;
- the verbatim reproduction and its failing output → a Fix's `# Symptom`, and
  the verification in `# Regression cases`;
- the scenario names involved, copied **verbatim** — F8 and F10 match them
  character for character;
- every feature the symptom touches → `affects:`;
- what must not break — the neighbouring scenarios whose cases you re-run.

For a Change, the same material becomes `## Why` in `<change-id>.spec.md`, and
the design gate decides what the behavior should become.

The lasting artefact of a defect is never the patch — it is the case in the
affected feature's `slug.test.md`. If the work ends with no test that would
have caught this, the bug is not finished.

Record the conclusion in the feature's log when it is worth the next agent's
time — a decision, or a dead end the next investigator should not repeat:
`fdf log <feature-id> '**Investigated**: …'`. Never edit the feature's frozen
`slug.spec.md`, `slug.plan.md` or tasks — they record how it was built, which
is the context that made this diagnosis possible.

## Red flags — STOP, you are rationalizing

| Thought | Reality |
|---|---|
| "I can see the fix — patch now, document after" | Code first is the drift FDF exists to stop, and a patch with no document leaves no regression case, so the bug returns. |
| "The code plainly does it — no need to run it" | Reading shows a cause, not the defect. The command you run is the regression case the repair starts from. |
| "It's a bug, so it's a Fix" | Only when a scenario already says otherwise. A case nobody specified is a Change. |
| "The scenario is wrong, I'll correct the Gherkin" | Delivered Gherkin changes only through a Change. Editing it directly erases the record of why it ever said that. |
| "The code is the real spec now" | The document is a promise, not a mirror of the code. Changing the promise is a decision someone makes deliberately. |
| "Just change X and see if it helps" | One written hypothesis, one small experiment, reverted — otherwise you cannot tell what worked. |
| "The test is flaky, rerun it" | Flaky means a root cause you have not found. Nondeterminism has a cause. |
| "Emergency — no time for process" | Systematic is faster than thrashing, and the smallest Fix is one file. |
| "The third fix failed; one more idea" | Three failures is an architecture question, not a fourth patch. Stop and ask. |
| "While I'm in here, I'll also…" | Scope past the root cause needs its own document. One cause, one repair. |
| "I'll leave the debug logging in, it's useful" | No document covers it. Remove it, or make it part of the documented work. |
| "It's undocumented code, so no document is needed to fix it" | Undocumented is not unowned. Map the capability (fdf-adopt), then a Change adds the promise — otherwise the fix leaves no scenario and no test, and the defect comes back unnoticed. |
| "We're not fixing it now; I'll mention it in the summary" | A defect known only to a chat log is found again from zero. File it with `fdf bug`, with the reproduction and the expected behavior. |
| "I'll file it as a debt" | The software does something wrong, so it is a bug. A debt's repair can be plain code work; a bug's never is. |
| "No root cause — it's just the environment" | Almost always an unfinished investigation. If it truly is environmental, say what you ruled out, then decide what the software should do about it. |

## Rules

- No proposed fix before the root cause is stated in one sentence.
- No kept behavior change before the document that authorizes it exists.
- Reproduce before fixing; verify afterwards with the same command.
- Never weaken a scenario or a test case to make a symptom go away — that is
  the silencing fdf-validate refuses.
- Temporary probes and trial edits are reverted before the work is called
  done.
- `fdf validate` exit 0 after every bundle edit.
