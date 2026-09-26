---
name: fdf-validate
description: Use after any edit to a file under an FDF bundle (docs/fdf/), and whenever `fdf validate` exits non-zero or prints FAIL lines — turns rule codes (F1–F14, R1) into the correct fix, and never silences a rule by weakening content.
---

# FDF Validate

`fdf validate` exit 0 is the gate after **every** bundle edit. This skill is
how you clear it honestly.

New to FDF? Run `fdf spec` for the format rules (also vendored in the bundle
at `docs/fdf/SPEC.md`) and `fdf help` for the CLI. The fdf-help skill
explains how the fdf skills fit together.

## When to run

Run `fdf validate` (it respects `--root`/`FDF_ROOT_DIR`):

- After **any** write to a file under the bundle — including a one-word
  status flip, a ticked task checkbox, or a typo fix. Especially then: those
  are the edits made without a workflow skill loaded, where drift goes unseen.
  The one exception is a set of edits that are only valid together — a spec
  and the status flip it enables, a plan and its tasks, a final task and its
  feature's `done`. Make the whole set, then validate; the workflow skills say
  which sets these are.
- Before telling the user a feature, task, or bundle change is done.
- After `fdf init`, `fdf new`, or `fdf migrate`.
- Whenever you are about to reason about bundle state — validate first, so
  you reason about what is actually there.

## Reading the output

```
FAIL: ARCHITECTURE.md: still an unfilled stub; ... (F9)
warn: <advisory>

6 document(s), 1 feature(s), 0 release(s) checked; 4 error(s), 0 warning(s).
Bundle is NOT conformant with FDF v0.7.
```

- Every FAIL line ends with its **rule code** — that code, not the prose, is
  what tells you which invariant broke.
- `warn:` lines are advisory and do **not** fail the run. Report them to the
  user; do not treat them as errors and do not "fix" them by guessing. One
  asks for a decision rather than a repair: a feature with no
  `slug.surface.md` and no `surface: none` is asked whether it has an
  interface. Answer by writing the surface document (fdf-brainstorm,
  *Surface document*) or, when it truly has none, adding `surface: none` —
  never `surface: none` on a feature with a screen, endpoint, command or
  event, just to quiet the warning.
- Exit 0 is the only pass. Fix, re-run, repeat until 0 — do not stop at
  "fewer errors than before".

## Rule triage

| Code | Broke | Usual fix |
|---|---|---|
| **F1** | Frontmatter missing/unterminated, missing `type`, a `timestamp` that is neither a date nor an RFC 3339 time with `Z` or an offset (v0.7), bad log date, unsupported `fdf_version` | Restore the `---` block and required fields. A timestamp with no zone gets the `Z` or offset it was written in; when that is unknown, keep only the date — never invent a time. An unsupported pin means run `fdf migrate` — never hand-edit the pin. |
| **F2** | `status` is not a legal value for that `type` | Use a real status: Feature `draft→specified→planned→implementing→done→retired`, or `adopted→retired` for code that predates its document; Debt and Bug `open`, then `accepted` or `resolved`; Change/Fix `draft→specified→planned→implementing→done`; Task `pending→in-progress→done`; Release `planned→shipped`. Set it to what is **true**, not what clears the error. |
| **F3** | Wrong `type`, wrong position, bad casing, illegal file in a task directory | Move the file to its FDF position. Directories and filenames are lowercase; uppercase is reserved. Task dirs hold **only** `NN-slug.md` — a trail doc nested there belongs at `<group>/<slug>.<role>.md`. Roles are only `spec`, `plan`, `test`, `surface`, `log` under a group — and only `spec`, `plan`, `log` under `changes/`, because `test` and `surface` belong to the affected feature. |
| **F4** | Status ↔ artifact mismatch | The status claims work the trail does not show, or vice versa. A `draft` may have a log and nothing else. `surface` takes only the value `none`, and a feature that says `surface: none` has no `slug.surface.md`: keep whichever is true. See "Which way to fix" below. An `adopted` feature is the exception that runs the other way: it must have **no** spec, plan or task directory and must name its code in `resource` — never write a build trail to satisfy F4 for existing code; if work is being done on it, that work is a Change (fdf-adopt). |
| **F5** | Feature Gherkin malformed | One ```gherkin fence with exactly one `Feature:`, at least one `Scenario:` — an `adopted` feature may have none yet (a map entry), and so may a `retired` one that was never built. A retired feature that was built keeps the scenarios it had. |
| **F6** | Plan ↔ task drift: unlinked task, dead link, missing plan, bad or cyclic `depends-on` | Make `# Tasks` in `slug.plan.md` list exactly the task files that exist. `depends-on` must name sibling tasks and must not cycle. |
| **F7** | Release ↔ `version` linkage, for features **and** changes | Reconcile `releases/*.md` with the `version:` fields the documents carry. `fdf release <version>` derives both lists for you; a shipped release may list only `done` documents, and features retired since they shipped. |
| **F8** | `slug.test.md` missing, or a Gherkin scenario has no test case | Add the case: a `## <scenario name>` heading under `# Test Cases`, the name matched **exactly** (from v0.7, a bullet or table row naming the scenario no longer counts; convert such cases to headings by hand). Fix the test file to match the scenario, not the scenario to match the test. A case that names no scenario is a warning: drop it, or give it the scenario's exact name. |
| **F9** | A Context doc is missing or still an unfilled stub | Stop. Run the **fdf-init** interview. Do not invent STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN content to clear this. |
| **F10** | A Change/Fix under `changes/` does not match reality | See "F10" below. |
| **F11** | A practice under `practices/` is malformed | A practice needs a non-empty `# Rules` section and carries no Gherkin. A `superseded` one must name an existing replacement in `superseded-by`; an `active` one must not carry that field. Its only legal sibling is `<slug>.log.md` — there is no practice spec, plan, test or task directory. |
| **F12** | The domain lexicon is inconsistent, or a banned word reached a document or a name the bundle chose | See "F12" below. |
| **F13** | A debt under `debts/` is malformed | A debt needs a non-empty `# Gap`, concrete enough that someone else could confirm it, and carries no Gherkin. `accepted` requires a `# Rationale` (keeping a gap is a decision); `resolved` requires a `# Resolution` (what closed it). A debt owns no spec, plan, test or tasks — only `<slug>.log.md`. |
| **F14** | A bug under `bugs/` is malformed, or cites a scenario that is not there | A bug needs a non-empty `# Symptom` (the reproduction, or the code path found by reading) and `# Expected`, and carries no Gherkin. Every `# Violates` heading is in `affects`, and while the bug is open every name under it exists **verbatim** — cite the scenario exactly, or drop `# Violates` when no scenario covers the case (the repair is then a Change). `accepted` requires `# Rationale` and forbids `# Violates`: a scenario the project knowingly keeps contradicting makes its feature lie — repair the code (Fix) or change the scenario (Change). `resolved` requires `# Resolution`, and a resolved bug still citing `# Violates` needs the done Fix or Change that `resolves` it: a bug is never repaired in place. While the bug is open, every feature under `# Violates` is delivered (`done` or `adopted`): in a feature still being built the defect is its tasks' to repair, so drop `# Violates` and keep the scenario's name in `# Expected`. |
| **R1** | A `resource:` (task/change/fix/debt/bug, or an adopted feature) or `applies-to:` (practice) path does not exist in the repo | The document points at a path that isn't there. **A task that will create the file:** the path does not belong in `resource:` yet — list the existing directory it goes into, and never create a placeholder file to satisfy R1. **A path that moved under a refactor:** update it to where that code lives now, even on a `done` task — a path is bookkeeping, not the record an episodic document keeps — and note the move in the log. **A path whose code was deleted:** on a done task, Change or Fix, remove the path and log what was removed (a path repair: the record keeps the fact in its log); on a living document the document is out of date — amend it. **A debt:** a vanished path usually means the gap is gone; check, and if so flip it to `resolved` with a `# Resolution`, then `fdf debt --cleanup` — a resolved debt is still checked until cleanup retires it. |

## Which way to fix

An F4/F8 failure has two shapes, and they take opposite fixes:

- **The trail is behind the status** — e.g. `planned` with no `slug.test.md`,
  or `done` with open tasks. Build the missing artifact: route to
  **fdf-plan** (spec approved, no plan/test) or **fdf-execute** (tasks open).
  Do not demote the status to dodge the rule unless the status was genuinely
  wrong.
- **The status is ahead of reality** — someone marked `done` early. Set the
  status to what is actually true. That is the rule working as intended.

`implementing` with every task done: promote to `done` — once it is true.
For a feature, every `slug.test.md` case has been run and passes (fdf-execute's
completion gate); if one has not, the final task is not really done, so set it
back to `in-progress`. For a Change or Fix, its declared effects must have
landed in the affected features first, or F10 fails next.

## F10

F10 checks that a post-delivery document's **declared effects actually
landed**. The declaration is the contract; the features are the evidence.

| Message | What it means | Fix |
|---|---|---|
| `done, but <feature> has no scenario "X"` | You declared `add:`/`modify: X` but the feature's Gherkin does not have it | Amend the feature's Gherkin. That is the work, not the error. |
| `done, but <feature> still has scenario "X"` | You declared `remove: X` and it is still there | Remove it from the Gherkin, and its case from `slug.test.md`. |
| `... a fix proves scenarios that already exist` | A `Fix` names a scenario the feature does not have | The document was silent on this case, so this is a **Change**, not a Fix. Convert it. |
| `<feature>.test.md has no case for "X"` | The regression case never reached the feature's living test doc | Add it there. That case is the whole point of the Fix. |
| `affects names <feature> with status '<s>'` | The feature is not delivered | Edit it directly through the normal workflow; a change request is for `done`/`adopted`/`retired` features. |
| `requires a # Scenario changes section` (or `# Regression cases`) | Wrong declaration section for the type | A `Change` declares scenario changes; a `Fix` declares regression cases. Never both. |
| `done, but the bug it resolves, bugs/<id>, is still 'open'` | The repair landed and the register still says the defect is there | Flip the bug to `resolved` with a `# Resolution` naming this document. Never drop `resolves` to clear it. |
| `resolves names "…", which is not a bug` | A typo, or the bug was never filed | Point at the real bug ID (`bugs/<slug>` or `bugs/<group>/<slug>`). A bug already cleared into `bugs/LOG.md` is fine once the document is done. |
| `status 'retired' but no done Change ... retires it` | A capability went dark with no reason recorded | Write the retiring `Change` with a `# Rationale`. If it exists but is not `done` yet, land it: its `done` and the feature's `retired` go in one edit (fdf-change). |
| `done, and retires names X whose status is 'done'` | The Change landed but the feature still reads as delivered | Flip the feature to `retired` in the same edit as the Change's `done`. |
| `retires names X ... until this Change is done, the feature it retires is still delivered` | The feature was flipped to `retired` before its Change landed | Put it back to `done` (or `adopted`) until the Change is `done`; then flip both in one edit. |
| `retires names X, which is not in affects` | Retiring a feature affects it | Add it to `affects`, with its `## <feature-id>` heading under `# Scenario changes` (left empty when no scenario changes). |
| `entry "…" is not add/modify/remove` (or `sits under no ## <feature-id> heading`) | A declaration line F10 cannot read, so it would check nothing | Write it as `- add: <scenario>`, `- modify: <scenario>` or `- remove: <scenario>`, lowercase, under the heading of the feature it changes. |
| `replaced-by names unknown feature` | The successor's ID is wrong, or it was never written | Point at the successor's real ID, or drop `replaced-by` when there is none. |

The fix for F10 is almost always **do the work you declared**, never edit the
declaration to match what you happened to do. If the declaration turned out
wrong, say so to the user and change it deliberately — that is a scope
change, not an error to silence.

## F12

Two different failures share this code.

**The lexicon itself is inconsistent** — a duplicate `## <Term>`, a term with
no definition, a word banned by one term that is another term's name, a word
claimed by two terms, an `except:` entry that contains none of its term's
banned words or is a banned word alone, a `strict:` that is not `true` or
`false`. These are errors in `DOMAIN.md`, and the fix is there: decide which
term owns the word, and say so. Do not resolve it by deleting the term someone
will still use.

**A banned word reached a document or a name the bundle chose.** From v0.7 F12
reads every document except `SPEC.md`, `DOMAIN.md`, `slug.test.md` and
`slug.surface.md` — features, specs, plans, tasks, changes, practices, debts,
bugs, logs, indexes, Context documents, finished work included — plus every
group, slug and task name. It skips what quotes rather than chooses: code spans,
non-Gherkin code blocks, link targets, URLs, HTML comments, double-quoted text
outside Gherkin, and a declaration's `## <feature-id>` headings and
verifications. Findings are `warn:` lines, one per document, and FAIL once
`DOMAIN.md` sets `strict: true` or `--strict-domain` is passed:

```
warn: payments/refunds.spec.md: uses banned "store" ×3 → "Venue" (F12)
warn: stores/: name uses banned "store" → "Venue" — rename it with `fdf mv` (F12)
warn: and 37 more document(s) or name(s) use banned words — `fdf lexicon` lists every occurrence (F12)
```

First check the direction: if the banned word is what the team actually says
now, the lexicon is out of date, and that is a **Context-document change** —
propose it and edit `DOMAIN.md` only on explicit approval. Never quietly drop
an `instead-of` entry to clear a warning.

**Sweep it — in this order.** A sweep touches finished work, so it is a lexicon
fix (a maintenance edit): it changes words and nothing else, needs no Change,
and is logged. It leaves every `timestamp` as it is — F10 orders finished work
by timestamp, and a sweep is not new work.

1. **Report**: `fdf lexicon` lists every occurrence with `file:line:col`,
   grouped by word; `--term <Term> --all` shows one term's in full.
2. **Triage the other senses.** A banned word often has an ordinary second
   meaning — *branch* in git, *integration* tests, a *log line*, an *AI agent*.
   Say which thing you mean: qualify it ("git branch", "integration test") and
   propose adding the qualified phrase to the term's `except:` in `DOMAIN.md`
   — a lexicon edit, so it needs approval like any other. A phrase, never the
   word alone.
3. **Mark the mentions.** Text *about* a word — a log entry recording that
   `store` was renamed to Venue — is not a use of it. Put the word in a code
   span; `fdf lexicon` flags italic ones, which are usually mentions.
4. **Fix one term at a time**: `fdf lexicon --term <Term> --fix --dry-run`,
   read the diff, then `--fix`. It keeps plurals, capitals and "a"/"an"
   right, renames a scenario name everywhere it is a join — Gherkin,
   `slug.test.md`, task acceptance, declarations — logs the sweep, and
   validates. What it cannot decide it lists and leaves: italic mentions, a
   label quoted in a Gherkin step, table cells, names.
5. **Names are moves**: `fdf mv <old-id> <new-id>` for each name the report
   lists, choosing the new name yourself — a mechanical one reads badly
   (`02-menu-from-catalog` → `02-catalog-from-catalog`).
6. Once the report is empty, propose `strict: true` in `DOMAIN.md`.

Rules for any lexicon fix, by hand or by tool: replace the banned word with its
term where it names the concept, and nothing else. A scenario name is a join —
change every copy in one edit, or F8 and F10 fail. A rename recorded before the
fix (`remove: Store owner…` with `add: Venue owner…`) ends up removing and
adding one name; F10 reads that as a replacement — leave it.

**The banned word is quoted user-facing copy** — `When I tap "Store
settings"`. The lexicon covers the project's internal language — the bundle and
the code's identifiers — not the words a person sees on a surface. So:

- Write the step around the concept, not the label text: `When the merchant
  opens the Venue settings`. In a delivered feature that is a lexicon fix too:
  the behavior is unchanged, so it is made in place.
- Keep the exact label outside the Gherkin: in the step definition, or in
  `slug.surface.md` when the wording is a surface decision. In prose, quote a
  label in a code span (`` `Store` ``).
- When the exact wording is what is under test — an error message, a URL, a
  button label — name the outcome in the scenario (`Then the merchant is told
  the Venue is closed`) and put the literal string in `slug.test.md`, in the
  case that checks it.
- Never change the label itself to clear the warning — nor locale files, help
  text, or any other wording people read. The surface may say "store" for a
  Venue on purpose.

## Never silence a rule

The bundle exists so the next agent can trust it. These "fixes" all pass
validate and all make the bundle lie:

| Tempting | Why it's a lie |
|---|---|
| Delete a `Scenario:` so F8 stops asking for a test case | You removed documented behavior instead of testing it. |
| Trim `slug.test.md` down to the scenarios you wrote | Same lie, other end. |
| Put `surface: none` on a feature that has an interface | The next person to change that endpoint or screen finds no record of what it is. Write the surface document. |
| Demote `done` → `implementing` to dodge an open-task error | Only correct if the feature is genuinely unfinished. |
| Drop an existing path from a task's `resource:` to clear R1 | The task now has no verifiable target. (A path the task will *create* never belonged there — list its existing directory instead.) |
| Create an empty placeholder file so a `resource:` path exists | The bundle now claims work that has not happened. |
| Write plausible-sounding STACK.md text to clear F9 | Invented context is worse than no context — every later feature is designed against it. |
| Hand-edit `fdf_version` to a supported value | Migration is mechanical; `fdf migrate` exists for this. |
| Delete the file the error names | The error was about the file's content, not its existence. |
| Drop an `instead-of` word so an F12 warning goes away | The synonym drift is the thing the lexicon exists to catch; you deleted the detector, not the problem. |
| Change a UI label, locale string or help text so a scenario stops tripping F12 | The lexicon covers internal language only; the surface may use that word on purpose. Reword the step around the concept, and keep the literal in the step definition or `slug.test.md`. |
| Slip a behavior change into a lexicon fix | A lexicon fix changes wording only. A scenario whose meaning changes is a Change, with its design gate, whatever else the edit renames. |
| Empty a practice's `# Rules` to clear F11 | A practice with no binding statements is a blog post. Write the rules or delete the practice. |
| Set a practice to `active` so `superseded-by` stops being required | It says the old mechanism is still how the project works. It isn't. |
| Mark a debt `resolved` with a vague `# Resolution` to clear F13 | The rule exists so "paid" and "quietly dropped" cannot look the same. Say what closed it, or leave it open. |
| Hide a banned word in quotes, italics or a code span so F12 stops seeing it | Code spans and quotes mark a *mention* or a quoted label. Using one to disguise the word the document actually chose is the drift the check exists to catch. |
| Add a bare banned word to `except:` | That un-bans it everywhere. `except:` takes phrases that say which other thing is meant. |
| Rename a bundle file by hand | Some `affects`, heading or link is missed, and the next validate fails far from the cause. `fdf mv` repairs every reference and logs the move. |
| Flip a bug to `resolved` after patching the code directly | A bug is never repaired in place. The repair is a Fix or Change that `resolves` it — that is what leaves the regression case. |
| Accept a bug that still cites a scenario, so F14 stops asking for a repair | Then the feature promises what the project has decided the software will not do. Repair the code, or change the scenario. |
| Write a spec, plan and done tasks for an existing capability so it can be `done` | That invents a history. It is an `adopted` feature, with no build trail. |
| Delete a debt instead of resolving or accepting it | The register then records neither the gap nor the decision. `fdf debt --cleanup` clears **resolved** entries into `debts/LOG.md`; nothing else removes one. |
| Flip a debt to `accepted` because paying it is inconvenient right now | `accepted` means the project has decided to keep the gap, with a reason someone stands behind. "Not today" is still `open`. |
| Edit a Change's `# Scenario changes` to match what you actually did | The declaration is what someone approved. Changing it silently makes F10 check nothing. |
| Turn a Change into a Fix so the spec gate goes away | If the Gherkin changes, someone must decide it. That is the gate, not paperwork. |
| Edit a delivered feature's Gherkin directly to clear a mismatch | Then nothing records why it changed. Open a Change. (A lexicon fix is the one exception: it changes words, not behavior.) |

If a rule looks genuinely wrong for a legitimate bundle, say so to the user
and stop — that is a spec or validator bug worth reporting, not something to
work around locally.

## Done

Exit 0, and you can state plainly which rules were failing and what you
changed to satisfy each. Report any remaining `warn:` lines. If you cleared
an error by changing a status, say which status and why it is now true.
