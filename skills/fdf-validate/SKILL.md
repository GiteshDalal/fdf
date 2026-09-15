---
name: fdf-validate
description: Use after any edit to a file under an FDF bundle (docs/features/), and whenever `fdf validate` exits non-zero or prints FAIL lines — turns rule codes (F1–F10, R1) into the correct fix, and never silences a rule by weakening content.
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
Bundle is NOT conformant with FDF v0.5.
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
| **F9** | A Context doc is missing or still an unfilled stub | Stop. Run the **fdf-init** interview. Do not invent STACK/ARCHITECTURE/SURFACES/INFRA content to clear this. |
| **F10** | A Change/Fix under `changes/` does not match reality | See "F10" below. |
| **R1** | A task `resource:` path does not exist in the repo | The plan points at a file that isn't there. Correct the path, or create the file if the task's work is what creates it. |

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
