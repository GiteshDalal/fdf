---
name: fdf-change
description: Use when a delivered (done or adopted) FDF feature must change, be repaired or be retired — including the repair of a bug on the register, and work spanning several features. A report that something is broken goes to fdf-debug first. Not for features still in draft, specified, planned or implementing.
---

# FDF Change

Amend a delivered feature without forking it or letting the bundle drift.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md`; `fdf help` documents every command.

**Read the Context documents first** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md`. A change lands in a system that
already exists: ground it in the documented stack and conventions, and say
explicitly when it would depart from them. Write scenario names and
declarations in `DOMAIN.md`'s vocabulary, and read the practices whose
`applies-to` covers the code this will touch
(`grep -rn -A3 -e '^status:' -e '^applies-to:' docs/fdf/practices`; a bare
`applies-to:` lists its paths on the `- ` lines below it): they bind this work
exactly as they bind a new feature.

## Mechanics

- `<feature-id>` and `<change-id>` are full IDs:
  `features/payments/instant-refunds`, `changes/payments/refund-window`.
  Paths are relative to the bundle root, `docs/fdf/`: the Change is
  `<change-id>.md`, its spec `<change-id>.spec.md`, its tasks `<change-id>/`.
- **Timestamps.** Every document you change in substance — a status flip, a
  living document you amend in step 8 — gets `timestamp:` set to the output
  of `date -u +%Y-%m-%dT%H:%M:%SZ`, run at the moment of that edit — never a
  time you estimate, round or reuse, and never your local time with `Z` added.
- **Log** with single quotes — `fdf log <feature-id> '**Changed**: …'` —
  because inside double quotes the shell runs anything between backticks. An
  apostrophe would end the quotes: write ’ instead.
- **The document comes first.** Change no code before the Change or Fix
  exists (step 3) — and, for a Change, before its design is approved
  (step 5).
- Validate after every edit to the bundle, except inside a set of edits that
  is only valid together; the steps below name those sets.

## Which document am I writing?

One question decides it:

> **Does a scenario of the feature already say what should happen, and the
> code does something else?**

| | `Fix` (`fdf fix`) | `Change` (`fdf change`) |
|---|---|---|
| Answer | Yes — the code drifted from what the document says | No — the software must do something the document does not say (it was silent, or wrong) |
| Gherkin | Unchanged | Changes — or, for an interface detail only, `slug.surface.md` changes |
| Design gate | None | Yes: an approved `slug.spec.md` |
| Smallest form | One file | The Change and its spec |
| Declares | `# Regression cases` | `# Scenario changes` |

- **The case people get wrong:** a bug that reveals the document was *silent*
  on a situation is a `Change`, not a `Fix`. Nobody ever decided that
  behavior, so someone has to decide it now — and that needs the gate. If you
  are about to write a regression case for a scenario that does not exist,
  stop: it is a Change.
- **An interface detail no scenario names** — a message's wording, a flag's
  name, an error code: how existing behavior is shown or invoked, recorded
  only in `slug.surface.md` — is a Change too. It lists the feature under
  `# Scenario changes` with its `## <feature-id>` heading and no entries, and
  amends `slug.surface.md`. An option that makes the software do something
  new (another output format, a filter) is behavior, not a detail: `add:`
  its scenario, with its test case, and amend `slug.surface.md`.
- **A new feature instead:** if the new behavior reads as its own `Feature:`
  block, with its own As a / I want / So that, it is not a change at all — it
  is a new feature (fdf-brainstorm), which names the feature it builds on in
  `depends-on`.
- **Do not settle this from the report's wording.** A defect's route follows
  from its **root cause**: if nobody has diagnosed it yet, run fdf-debug
  first and come back with the cause in one sentence — a Fix's `# Symptom`
  and `# Root cause` are waiting for it.

## Statuses

| Status | `Change` | `Fix` |
|---|---|---|
| `draft` | Scaffolded; no spec, plan or tasks yet | Scaffolded |
| `specified` | Its spec is written and approved (required) | Only if you choose to write a spec |
| `planned` | Its plan and tasks are written | Its plan and tasks are written |
| `implementing` | Its first task is started | Its first task is started |
| `done` | The work is in, and every declared effect is true | The work is in, and every regression case has landed |

The paths through them:

- a Fix with no tasks: `draft` → `done`;
- a Fix with tasks: `draft` → `planned` → `implementing` → `done`;
- a Change with no tasks: `draft` → `specified` → `done`;
- a Change with tasks: `draft` → `specified` → `planned` → `implementing` →
  `done`.

## Process

1. **Confirm the feature is delivered.** `affects` may only name features that
   are `done`, `adopted` or `retired`.
   - A feature still in flight (`draft` to `implementing`) is edited directly —
     route back to fdf-brainstorm, fdf-plan or fdf-execute.
   - Code no feature documents is adopted first (fdf-adopt), then changed
     here.
   - For an `adopted` feature, first backfill whatever behavior this work will
     alter — decided from the code, not from the scenario list (a map entry
     has none): fdf-adopt, *Phase 2*. Behavior the code does not have yet needs
     no backfill, and a defect is never backfilled.
2. **Understand the change — before you scaffold anything.** Ask questions,
   ONE at a time: what is wrong, for whom, what should happen instead, what
   must not break? Even when the request looks complete, name every word it
   leaves open and ask about each: a time ("within 30 days" — counted from
   when, in which time zone, and is day 30 itself allowed?), a boundary, an
   amount, a message. Each one you settle yourself is a design decision
   nobody approved. A Fix coming from fdf-debug already has its evidence; ask
   only what it lacks. For a Change, also read the affected feature's
   `slug.spec.md` — above all its `## Alternatives rejected` — and the specs
   of the Changes `fdf history <feature-id>` lists. Copy out every
   `## Design decisions` bullet and every `## Alternatives rejected` entry
   about the rule you are changing, each with its reason — never edit those
   specs: the design shows them (step 5), rather than overriding them
   silently.
3. **Scaffold the document** — flags before the name:
   - **Check the register first**: `fdf bug --open`. A bug whose `# Symptom`
     is the defect this work repairs is repaired from it (next bullet); any
     other bug is left as it is.
   - **Repairing a bug on the register**, scaffold from it:
     - the bug cites a scenario under `# Violates` →
       `fdf fix --from <bug-id> [<group>/…]<slug>`. It copies the bug's
       `# Symptom` and `# Root cause`, turns its `# Violates` scenarios into
       regression cases, takes its `affects`, and writes `resolves: <bug-id>`;
     - the bug cites none →
       `fdf change --from <bug-id> [<group>/…]<slug>`. It copies the bug's
       `# Symptom` and `# Expected` into `# Problem`, takes its `affects`, and
       writes `resolves: <bug-id>`; carry its root cause into the spec's
       `## Why` yourself.
   - **Otherwise**:
     `fdf change --affects <feature-id>[,…] [<group>/…]<slug>` or
     `fdf fix --affects <feature-id>[,…] [<group>/…]<slug>`, for example
     `fdf change --affects features/payments/instant-refunds payments/refund-window`.
   - **Filing**: flat is fine until `changes/` holds about ten documents;
     then group them, usually after the affected feature's group
     (`payments/refund-window`). The group is filing only: nothing ties it to
     the feature's group.
   - **Clean the scaffold**: write a real one-sentence `description:`; replace
     or delete every line that contains `TODO`; and replace the listing line
     the command added to the `INDEX.md` beside it (it ends `- TODO.`) with
     that description.
4. **Declare the effects** in the document's body. This is what makes the work
   verifiable, so be exact: names are matched **character for character**
   against the feature's Gherkin.
   - **A Change** — fill `# Problem` with what is wrong and for whom, in the
     request's and the user's words — no reason they did not give — then
     under `# Scenario changes`, one
     `## <feature-id>` heading per affected feature (for example
     `## features/payments/instant-refunds`), each entry one of:
     - `- add: <scenario name>` — must exist in that feature once done;
     - `- modify: <scenario name>` — exists before and after, **under the same
       name**; only its steps change;
     - `- remove: <scenario name>` — must not exist once done.

     Use `modify:` only while the old name stays true. When the change makes
     the name itself false — "A settled payment cannot be refunded", once
     some can be — it is a **rename**: `- remove: <old name>` plus
     `- add: <new name>`. Copy every `modify:` and `remove:` name from the
     feature file **before** you edit it: F10 checks only the result, so it
     cannot tell a renamed scenario declared as `modify:` from a kept one — the
     declaration is yours to get right. Name each new scenario for the rule
     as decided, in the feature's naming style, with the boundary's value in
     the name — "A refund on day 30 is accepted", never "A refund inside the
     window is accepted" — and repeat the value in its steps, with its unit
     or time zone. A feature affected with no scenario
     change (an interface detail, a retirement) keeps its heading with no
     entries.

     A rename whose only reason is a word `DOMAIN.md` bans is not a Change at
     all: it is a lexicon fix, made in place across the bundle (fdf-validate,
     F12).
   - **A Fix** — fill `# Symptom` with the verbatim reproduction and its
     failing output, and `# Root cause` with why the code diverged, at
     `<file>:<line>`, and the commit that `git log -L <line>,<line>:<file>`
     shows last changed that line: both come straight out of the fdf-debug
     investigation, never guesses written after the patch. Under `# Regression cases`, one `## <feature-id>` heading per
     affected feature, and one entry per scenario the Fix proves:
     `- <scenario name> — <command, test path, or manual procedure>`
     (an em dash, an en dash or `--`, with a space on each side). Every name
     must **already exist** in that feature.
5. **A Change gets its design approved; a Fix skips this step.** Present, one
   part at a time:
   1. the open words you asked about in step 2, each with the user's answer
      — never listed among your own decisions (item 5);
   2. the `# Problem`, and the scenario changes you declared, one line each
      on what the scenario will promise;
   3. the approach you recommend and at least one other way you considered,
      each with its trade-off, and the trade-offs you accept;
   4. the interface change, when there is one — the exact wording, flags,
      codes and output that will go into `slug.surface.md`;
   5. every decision you took yourself, one bullet each;
   6. each bullet and entry you copied in step 2, quoted with its reason,
      whether or not you think this work reverses it — or "none copied". Ask
      the user, for each, whether this work reverses it and whether its
      reason still holds; record their answers, never yours.

   Then ask one explicit question: **"Do you approve this design?"** If the
   user declines the change altogether, delete the draft Change and its line
   in the `INDEX.md` beside it, and stop.

   **Gate — stop here.** No spec, no status flip and no code until the user
   says yes. On yes, write `<change-id>.spec.md` and set the Change's
   `status: specified` and `timestamp` (run `date -u +%Y-%m-%dT%H:%M:%SZ` now), in one edit:

   ```markdown
   ---
   type: Spec
   title: Refund window — design
   description: Why the refund window becomes 90 days, and what was rejected.
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   ## What is being built

   ## Why

   <the user's reason>

   ## Design decisions

   - <one bullet per resolved ambiguity and per decision you took>
   - Reverses: <the quoted decision or rejected alternative> — <the user's answer on why its reason no longer holds — or, when they gave none, "approved, no reason given">

   ## Alternatives rejected

   - <each alternative you presented in step 5 — at least one — with the reason you gave>
   ```

   Replace every `<…>` with real text. The spec records only what you
   presented and the user approved — never a reason the user did not give:
   `## Why` holds the reason from the request and the user's answers, and
   `## Alternatives rejected` only alternatives you presented in step 5. None
   presented? Go back to step 5; never add one here. Write one `Reverses:`
   bullet per entry the user said this work reverses; with none, drop it. Then log the approval in the Change's own
   log:
   `fdf log <change-id> '**Specified**: design approved by <who>; <the approach in one line>.'`
   `<who>` is the name the user gave, or "the user" — never a name you
   guessed.
6. **Tasks — only when needed.** Write tasks when one sitting cannot finish
   the work, or it splits into parts that are checked separately. Otherwise
   skip to step 7. Tasks follow fdf-plan's steps 5 and 8, with two
   differences: they live under `<change-id>/` and the plan is
   `<change-id>.plan.md`; and there is no test document — the affected
   features' `slug.test.md` carry the proof. Write the plan and the tasks and
   set the document's `status: planned` in one edit (a task directory without
   its plan fails F6). Then work them as fdf-execute does: the first task's
   `in-progress` and the document's `implementing` in one edit, task by task.
7. **Do the work and prove it**, in this order:
   1. **For a Fix, first**: run each `# Regression cases` command on the
      **unfixed** code and paste its failure. It passes? Then it cannot catch
      this defect: strengthen the test it runs (fdf-debug, Phase 1 step 5)
      until it fails as `# Symptom` shows.
   2. Implement the change — through its tasks, or directly when there are
      none — and test its boundary, when the rule has one: the last value
      allowed and the first refused, exactly — for refunds allowed up to 30
      days, day 30 accepted and day 31 refused; for refunds up to 500.00 EUR,
      500.00 accepted and 500.01 refused. A check far from the boundary passes
      an off-by-one.
   3. For a Fix, run the step 7.1 commands again: each must now pass.
   4. Run the affected features' `slug.test.md` cases this work touches, plus
      their neighbours.

   Report each command with its actual output; a case you did not run keeps
   the document short of `done`.
8. **Amend each living document this work made untrue.** Each one you edit
   gets a new `timestamp`; one you leave alone keeps its own. This is the step
   that keeps the bundle true, and F10 enforces most of it. The Gherkin and
   `slug.test.md` are only valid together (F8): amend both, then validate.
   - The feature's **Gherkin** — apply exactly the adds, modifies and removes
     you declared. Write new and changed steps around the concept, not the
     label a surface shows; wording that is itself under test (an error
     message, a URL, a button label) is named as an outcome in the scenario
     and pinned in `slug.test.md`. Never reword a label to suit `DOMAIN.md`.
   - The feature's **`slug.test.md`** — a `## <scenario name>` case for every
     scenario, the name exactly as in the Gherkin (F8); drop the case of a
     removed scenario. A scenario with a boundary gets a check on each side of
     it — the last value allowed and the first refused (a refund on day 30,
     and on day 31). **For a Fix**, edit the case of each scenario you named
     so it runs the check that now catches this defect — the regression
     command. The case states today's check: the command and the input it
     tries. Write no link to this Fix and no history ("previously…", "fixed
     by…"): `affects` and the feature's log hold that. Write the input in the
     case's text, in the scenario's words, even when the command is a test
     name: "`go test ./internal/payments -run TestRefund_Full` passes: a
     settled payment refunded in full is marked refunded." When the case's text
     already says the command and that input, keep it and set only the test
     document's `timestamp` — once you saw the command fail on the
     unfixed code in step 7.1, whenever its test was written. It passed
     there? Then it never caught this defect: go back to step 7.1.
     `fdf validate` only compares the test document's `timestamp` with the
     Fix's; it cannot tell whether the case catches the defect. The strengthened case is the
     Fix's whole lasting value.
   - The feature's **`slug.surface.md`**, when the work changed or added an
     interface — an endpoint's codes, a screen's copy, a flag, an event's
     fields. It describes the interfaces as they are today, so replace the old
     shape; don't keep it beside the new one. A feature with an interface but
     no surface document gets one now.
   - A **map entry's first scenario** (an adopted feature with none yet): add a
     `# Scenarios` heading with one gherkin fence per scenario; create its
     `slug.test.md` (frontmatter `type: Test`, `title`, `description`,
     `timestamp`, then `# Test Cases`); and write its `slug.surface.md`, or add
     `surface: none`.
   - The feature's `description`, and its listing line in the `INDEX.md`
     beside it, when the change made them untrue.

   Do **not** touch the feature's `slug.spec.md`, `slug.plan.md` or tasks:
   they are the frozen record of how it was first built — the context someone
   needs to judge this change and the next one.
9. **Flip to `done`, in one edit**, each with a new `timestamp` (run
   `date -u +%Y-%m-%dT%H:%M:%SZ` now): the document (and its
   last task, if it has tasks); every bug it `resolves`, flipped to `resolved`
   with a `# Resolution` naming this document (F10 refuses a done repair whose
   bug still reads `open`); and any open debt this work closes, flipped to
   `resolved` with a `# Resolution`. Then run `fdf validate`: exit 0. F10
   refuses a `done` document whose declared effects are not true.
10. **Log it** in each affected feature's own log, one entry naming this
    document, so the feature's log tells its whole life:
    `fdf log <feature-id> '**Changed**: [<change-id>](/<change-id>.md) <what now differs>.'`
    (`**Fixed**` for a Fix). Decisions taken while doing the work go in this
    document's own log: `fdf log <change-id> '**Decision**: …'`. Neither goes
    in the root `LOG.md`.
11. **Project-document review** — the four questions of fdf-execute's review,
    in order. Tell the user all four answers, even when each is no, in this
    shape:

    ```text
    1. Context documents — STACK.md: <what you checked> — current | stale: <edit> | stale or a defect: ask
       (one line each for ARCHITECTURE.md, SURFACES.md, INFRA.md, DOMAIN.md)
    2. Practices — <none governs these paths | followed | diverged: <how>>
    3. New practice — <no | the mechanism a second feature now repeats>
    4. Debt or bug — <none | what was left undone, or found broken>
    ```

    For question 1, open each of the five and compare it with what this work
    changed (`git diff`). For question 2, a divergence is a defect or an
    approved `# Exceptions` entry. For question 4, when the work spread a gap
    an open debt already names to new code, amend that debt's `# Gap` and
    `resource` instead of filing a second one. **Propose** each edit — Context
    document, practice, debt or bug — and wait for explicit approval before
    writing it. See *Practices and a change*: post-delivery work is where
    practice drift surfaces.

## Practices and a change

Post-delivery work is where the gap between what a practice says and what the
code does becomes visible. Three cases come up here that do not come up during
a feature.

**A Fix whose root cause was an unwritten rule.** The code diverged, and
nothing told it not to. That is a practice waiting to be written: the same
defect will arrive again in the next handler. Propose one, with the Fix as its
evidence. If other call sites still do it the old way, propose filing them
too: a **bug** where they misbehave observably now, a **debt** where the gap
is latent — so the next reader sees a known gap instead of an inconsistency to
re-derive.

**A Fix whose root cause was a written rule nobody followed.** The practice
exists and was ignored. Do not amend the practice to match the code; fix the
code, and consider whether the `# Rules` line was too vague to follow —
sharpening it is an edit worth proposing.

**A Change that alters the mechanism itself.** When the project now does
something a *different* way, the practice changes too. It is a living document,
so it is amended in place — the practice always describes today — and the
Change records why. Two shapes:

- *The mechanism is refined* — edit the `# Rules`, and log it in the
  practice's own log: `fdf log <practice-id> '**Amended**: …'`.
- *The mechanism is replaced* — set the old practice to `status: superseded`
  with `superseded-by: <new practice-id>`, and write the replacement. Never
  delete the old one: code in the tree still follows it, and it explains that
  code. F11 requires the `superseded-by` target to exist. The code still on
  the old mechanism is a debt — propose filing it, because "superseded"
  describes the document, not the tree.

A practice is never forked per change, and a change never gets its own copy of
one: one practice per mechanism, amended forever. Every practice edit needs the
user's explicit approval first — a practice binds all future code, so changing
one is a bigger decision than the change that prompted it.

## Retiring a feature

When a delivered capability is removed from the product, its document is not
deleted: it records behavior the software once had. A retirement is a
**Change**, and runs the Process above — design gate included — with these
differences:

1. **Scaffold** it as usual (`fdf change --affects <feature-id> …`), then add
   to its frontmatter `retires: <feature-id>` straight away, and a
   `# Rationale` section to its body saying why the capability is going and
   what supersedes it. Under `# Scenario changes`, keep the feature's
   `## <feature-id>` heading and leave it empty: the retired feature keeps its
   Gherkin as the record of what it did.
2. **Design gate** (step 5): the spec is approved and the Change is
   `specified`, like any other Change. A retirement backfills nothing
   (Process step 1 does not apply): the feature keeps the scenarios it has.
3. **Do the work**: remove the capability. Any `resource` path that names
   code you deleted, on the feature's finished tasks, Changes and Fixes, is
   removed from that list (a path repair; R1 fails otherwise), and the repair
   is logged. Prove it with the neighbouring features' cases: the retired
   feature's own tests go with its code.
4. **Land it in one edit**: the Change's `status: done`, the feature's
   `status: retired` — plus `replaced-by: <feature-id>` when a successor
   exists — with timestamps. An `adopted` feature's own `resource` is removed
   in this same edit: it is required while the feature is `adopted` (F4), and
   names deleted code once it is `retired` (R1). A done Change's `retires`
   must name a `retired` feature, and a `retired` feature needs a done Change
   retiring it (F10), so neither is valid without the other. Then validate.
5. **Log** it on the feature: `fdf log <feature-id> '**Retired**: [<change-id>](/<change-id>.md) <why>.'`

A `Fix` can never retire anything — removing behavior is a decision.
Retirement is terminal: a capability that comes back is a new feature
(fdf-brainstorm), which may name the retired one in `depends-on`; the retired
document is never flipped back.

## Rules

- Never fork a second feature document for the same capability.
- Never edit a delivered feature's Gherkin outside a Change or Fix: then
  nothing records why, and nothing verified that the code followed. Two
  exceptions: a lexicon fix (it changes words, not behavior), and backfilling
  an adopted feature — adding a scenario its code already passes (fdf-adopt).
- Never rewrite the feature's original spec, plan or tasks, except by a
  maintenance edit — a lexicon fix, a reference repair by `fdf mv`, or a path
  repair after code moved or was deleted.
- Never resolve a bug in place: its repair is this document, which names it in
  `resolves`.
- Never fork a practice per change: amend the one that exists, or supersede
  it.
- Never hand-write a back-link on the feature, its `slug.test.md` or its
  `slug.surface.md`: `affects:` is the whole link, and
  `fdf history <feature-id>` computes the trail.
- A `Fix` names only scenarios that already exist. If you need a new one, it
  is a `Change`.
- `fdf validate` exit 0 after every bundle edit.
