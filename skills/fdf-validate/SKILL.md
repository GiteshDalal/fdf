---
name: fdf-validate
description: Use after any edit to a file under an FDF bundle (docs/features/), and whenever `fdf validate` exits non-zero or prints FAIL lines — turns rule codes (F1–F13, R1) into the correct fix, and never silences a rule by weakening content.
---

# FDF Validate

`fdf validate` exit 0 is the gate after **every** bundle edit. This skill is
how you clear it honestly.

New to FDF? Run `fdf spec` for the format rules (also vendored in the bundle
at `docs/features/SPEC.md`) and `fdf help` for the CLI. The fdf-help skill
explains how the fdf skills fit together.

## When to run

Run `fdf validate` (it respects `--root`/`FDF_ROOT_DIR`):

- After **any** write to a file under the bundle — including a one-word
  status flip, a ticked task checkbox, or a typo fix. Especially then: those
  are the edits made without a workflow skill loaded, where drift goes unseen.
- Before telling the user a feature, task, or bundle change is done.
- After `fdf init`, `fdf new`, or `fdf migrate`.
- Whenever you are about to reason about bundle state — validate first, so
  you reason about what is actually there.

## Reading the output

```
FAIL: ARCHITECTURE.md: still an unfilled stub; ... (F9)
warn: <advisory>

6 document(s), 1 feature(s), 0 release(s) checked; 4 error(s), 0 warning(s).
Bundle is NOT conformant with FDF v0.6.
```

- Every FAIL line ends with its **rule code** — that code, not the prose, is
  what tells you which invariant broke.
- `warn:` lines are advisory and do **not** fail the run. Report them to the
  user; do not treat them as errors and do not "fix" them by guessing.
- Exit 0 is the only pass. Fix, re-run, repeat until 0 — do not stop at
  "fewer errors than before".

## Rule triage

| Code | Broke | Usual fix |
|---|---|---|
| **F1** | Frontmatter missing/unterminated, missing `type`, bad log date, unsupported `fdf_version` | Restore the `---` block and required fields. An unsupported pin means run `fdf migrate` — never hand-edit the pin. |
| **F2** | `status` is not a legal value for that `type` | Use a real status: Feature `draft→specified→planned→implementing→done→retired`; Change/Fix `draft→specified→planned→implementing→done`; Task per the spec. Set it to what is **true**, not what clears the error. |
| **F3** | Wrong `type`, wrong position, bad casing, illegal file in a task directory | Move the file to its FDF position. Directories and filenames are lowercase; uppercase is reserved. Task dirs hold **only** `NN-slug.md` — a trail doc nested there belongs at `<group>/<slug>.<role>.md`. Roles are only `spec`, `plan`, `test`, `surface`, `log` under a group — and only `spec`, `plan`, `log` under `changes/`, because `test` and `surface` belong to the affected feature. |
| **F4** | Status ↔ artifact mismatch | The status claims work the trail does not show, or vice versa. See "Which way to fix" below. |
| **F5** | Feature Gherkin malformed | One ```gherkin fence with exactly one `Feature:`, at least one `Scenario:`. |
| **F6** | Plan ↔ task drift: unlinked task, dead link, missing plan, bad or cyclic `depends-on` | Make `# Tasks` in `slug.plan.md` list exactly the task files that exist. `depends-on` must name sibling tasks and must not cycle. |
| **F7** | Release ↔ `version` linkage, for features **and** changes | Reconcile `releases/*.md` with the `version:` fields the documents carry. `fdf release <version>` derives both lists for you; a shipped release may list only `done` documents. |
| **F8** | `slug.test.md` missing, or a Gherkin scenario has no test case | Add the case. Scenario names are matched **verbatim** — fix the test file to match the scenario, not the scenario to match the test. |
| **F9** | A Context doc is missing or still an unfilled stub | Stop. Run the **fdf-init** interview. Do not invent STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN content to clear this. |
| **F10** | A Change/Fix under `changes/` does not match reality | See "F10" below. |
| **F11** | A practice under `practices/` is malformed | A practice needs a non-empty `# Rules` section and carries no Gherkin. A `superseded` one must name an existing replacement in `superseded-by`; an `active` one must not carry that field. Its only legal sibling is `<slug>.log.md` — there is no practice spec, plan, test or task directory. |
| **F12** | The domain lexicon is inconsistent, or a banned word reached a feature's Gherkin | See "F12" below. |
| **F13** | A debt under `debts/` is malformed | A debt needs a non-empty `# Gap`, concrete enough that someone else could confirm it, and carries no Gherkin. `accepted` requires a `# Rationale` (keeping a gap is a decision); `resolved` requires a `# Resolution` (what closed it). A debt owns no spec, plan, test or tasks — only `<slug>.log.md`. |
| **R1** | A `resource:` (task/change/fix/debt) or `applies-to:` (practice) path does not exist in the repo | The document points at a file that isn't there. Correct the path, or create the file if the task's work is what creates it. For a **debt**, a vanished path usually means the gap is gone: check, and if so flip it to `resolved` with a `# Resolution` rather than editing the path. |

## Which way to fix

An F4/F8 failure has two shapes, and they take opposite fixes:

- **The trail is behind the status** — e.g. `planned` with no `slug.test.md`,
  or `done` with open tasks. Build the missing artifact: route to
  **fdf-plan** (spec approved, no plan/test) or **fdf-execute** (tasks open).
  Do not demote the status to dodge the rule unless the status was genuinely
  wrong.
- **The status is ahead of reality** — someone marked `done` early. Set the
  status to what is actually true. That is the rule working as intended.

`implementing` with every task done is the one case with a single fix:
promote to `done`.

## F10

F10 checks that a post-delivery document's **declared effects actually
landed**. The declaration is the contract; the features are the evidence.

| Message | What it means | Fix |
|---|---|---|
| `done, but <feature> has no scenario "X"` | You declared `add:`/`modify: X` but the feature's Gherkin does not have it | Amend the feature's Gherkin. That is the work, not the error. |
| `done, but <feature> still has scenario "X"` | You declared `remove: X` and it is still there | Remove it from the Gherkin, and its case from `slug.test.md`. |
| `... a fix proves scenarios that already exist` | A `Fix` names a scenario the feature does not have | The document was silent on this case, so this is a **Change**, not a Fix. Convert it. |
| `<feature>.test.md has no case for "X"` | The regression case never reached the feature's living test doc | Add it there. That case is the whole point of the Fix. |
| `affects names <feature> with status '<s>'` | The feature is not delivered | Edit it directly through the normal workflow; a change request is for `done`/`retired` features. |
| `requires a # Scenario changes section` (or `# Regression cases`) | Wrong declaration section for the type | A `Change` declares scenario changes; a `Fix` declares regression cases. Never both. |
| `status 'retired' but no done Change ... retires it` | A capability went dark with no reason recorded | Write the retiring `Change` with a `# Rationale`. |

The fix for F10 is almost always **do the work you declared**, never edit the
declaration to match what you happened to do. If the declaration turned out
wrong, say so to the user and change it deliberately — that is a scope
change, not an error to silence.

## F12

Two different failures share this code.

**The lexicon itself is inconsistent** — a duplicate `## <Term>`, a term with
no definition, a word banned by one term that is another term's name, a word
claimed by two terms. These are errors in `DOMAIN.md` and the fix is in
`DOMAIN.md`: decide which term owns the word, and say so. Do not resolve it by
deleting the term someone will still use.

**A banned word reached a feature's Gherkin** — reported as a `warn:` by
default and as a FAIL under `fdf validate --strict-domain`:

```
warn: payments/refunds.md: Gherkin uses "store", which DOMAIN.md bans in
      favour of "Venue" (F12)
```

The fix is to reword the scenario, not to weaken the lexicon. But check the
direction first: if the banned word is what the team actually says now, the
lexicon is out of date, and that is a **Context-document change** — propose it
to the user and only edit `DOMAIN.md` on explicit approval, exactly as for the
other four. Never quietly drop an `instead-of` entry to clear a warning.

**The banned word is inside quoted user-facing copy** — `When I tap "Store
settings"`. The lexicon governs the project's internal language (the bundle
and the code's identifiers), not what a person reads on a surface, so the
label is not wrong and must not be changed to clear the warning — nor may
locale files, help text or any other external wording. Reword the step around
the concept — `When the merchant opens the Venue settings` — and let the step
definition or `slug.surface.md` carry the literal label. Where the exact
wording is the thing under test, name the outcome in the scenario and pin the
string in `slug.test.md`.

A rewording that touches a **delivered** feature's Gherkin is a `Change`, not
an edit: scenario names are the join F8 and F10 both check, and renaming one
in place breaks that trail. Report the warning and route to fdf-change.

## Never silence a rule

The bundle exists so the next agent can trust it. These "fixes" all pass
validate and all make the bundle lie:

| Tempting | Why it's a lie |
|---|---|
| Delete a `Scenario:` so F8 stops asking for a test case | You removed documented behavior instead of testing it. |
| Trim `slug.test.md` down to the scenarios you wrote | Same lie, other end. |
| Demote `done` → `implementing` to dodge an open-task error | Only correct if the feature is genuinely unfinished. |
| Drop a task's `resource:` line to clear R1 | The task now has no verifiable target. |
| Write plausible-sounding STACK.md text to clear F9 | Invented context is worse than no context — every later feature is designed against it. |
| Hand-edit `fdf_version` to a supported value | Migration is mechanical; `fdf migrate` exists for this. |
| Delete the file the error names | The error was about the file's content, not its existence. |
| Drop an `instead-of` word so an F12 warning goes away | The synonym drift is the thing the lexicon exists to catch; you deleted the detector, not the problem. |
| Empty a practice's `# Rules` to clear F11 | A practice with no binding statements is a blog post. Write the rules or delete the practice. |
| Set a practice to `active` so `superseded-by` stops being required | It says the old mechanism is still how the project works. It isn't. |
| Mark a debt `resolved` with a vague `# Resolution` to clear F13 | The rule exists so "paid" and "quietly dropped" cannot look the same. Say what closed it, or leave it open. |
| Delete a debt instead of resolving or accepting it | The register then records neither the gap nor the decision. `fdf debt --cleanup` clears **resolved** entries into `debts/LOG.md`; nothing else removes one. |
| Flip a debt to `accepted` because paying it is inconvenient right now | `accepted` means the project has decided to keep the gap, with a reason someone stands behind. "Not today" is still `open`. |
| Edit a Change's `# Scenario changes` to match what you actually did | The declaration is what someone approved. Changing it silently makes F10 check nothing. |
| Turn a Change into a Fix so the spec gate goes away | If the Gherkin changes, someone must decide it. That is the gate, not paperwork. |
| Edit a delivered feature's Gherkin directly to clear a mismatch | Then nothing records why it changed. Open a Change. |

If a rule looks genuinely wrong for a legitimate bundle, say so to the user
and stop — that is a spec or validator bug worth reporting, not something to
work around locally.

## Done

Exit 0, and you can state plainly which rules were failing and what you
changed to satisfy each. Report any remaining `warn:` lines. If you cleared
an error by changing a status, say which status and why it is now true.
