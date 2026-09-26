---
name: fdf-validate
description: Use after any edit to a file under an FDF bundle (docs/fdf/), and whenever `fdf validate` exits non-zero or prints FAIL or warn lines — before reporting the work done, and before any fix that deletes, demotes or weakens something.
---

# FDF Validate

`fdf validate` exit 0 is the gate after **every** bundle edit. This skill is
how you clear it honestly.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md` (rule codes F1–F14 and R1 are defined there); `fdf help`
documents every command.

## When to run

Run `fdf validate` (it respects `--root` and `FDF_ROOT_DIR`):

- after **any** write to a file under the bundle — including a one-word status
  flip, a ticked checkbox or a typo fix. Especially then: those are the edits
  made without a workflow skill loaded, where drift goes unseen. The one
  exception is a set of edits that are only valid together — a spec and the
  status flip it enables, a plan and its tasks, a final task and its feature's
  `done`. Make the whole set, then validate; the workflow skills name these
  sets;
- before telling the user a feature, task or bundle change is done;
- after `fdf init`, `fdf new` and `fdf migrate`;
- before you reason about the bundle's state — so you reason about what is
  actually there.

## Mechanics

- A document you write, and a fix that changes a document's substance — a
  status, a section — sets its `timestamp:` to the output of
  `date -u +%Y-%m-%dT%H:%M:%SZ`, run at the moment of the edit — never a time
  you estimate or round, and never your local time with `Z` added. A
  maintenance edit (a lexicon fix, a reference or path repair) and a pure
  format repair leave `timestamp` as it is. Two more exceptions: a document
  restored from git keeps its old `timestamp`
  (`git checkout <commit>^ -- <path>`; never retype it), and a malformed
  `timestamp` already committed is repaired as *F1* below says — never set to
  now.
- **Undoing a mistaken edit** — a deleted document, a demoted heading, a
  reformatted value — restores the earlier bytes from git, `timestamp`
  included: a restore is not new work. `git log -p -- <path>` finds the
  commit that made the mistake; `git checkout <commit>^ -- <path>` restores
  the file as it was just before it, and `git show <commit>^:<path>` prints
  it, when only part of the file is to be restored.
- Log with single quotes — `fdf log <id> '**Decision**: …'` — because inside
  double quotes the shell runs anything between backticks.
- **Ask the user** when the repair needs a decision the documents cannot make
  for you: whether a feature has an interface, which of two contradicting
  documents is right, whether a status is really true. Say what you found and
  what each answer would change.

## Reading the output

```
FAIL: ARCHITECTURE.md: still an unfilled stub; ... (F9)
warn: features/payments/refunds.md: missing recommended `description`

6 document(s), 1 feature(s), 0 release(s) checked; 4 error(s), 1 warning(s).
Bundle is NOT conformant with FDF v1.0.
```

- **Read the whole line.** The rule code — usually at the end, sometimes
  mid-line, as in `frontmatter is not parseable (F1): …` — picks the section
  below; the prose usually names the fix ("did you mean features/…?",
  "promote to 'done'").
- **Exit 0 is the only pass.** Fix, re-run, repeat until 0 — never stop at
  "fewer errors than before".
- **`warn:` lines do not fail the run**, but each says something is off. Fix
  the ones whose repair you know; ask about the ones that need a decision (see
  *Warnings*). Report whatever warnings remain when you finish.

## Rule triage

| Code | What broke | Section or fix |
|---|---|---|
| **F1** | Frontmatter, a `timestamp`, a log's dates, or the `fdf_version` pin | *F1* below |
| **F2** | `status` is not a legal value for the document's `type` | Set the status that is **true**. The legal ones: Feature `draft → specified → planned → implementing → done → retired`, or `adopted → retired`; Change and Fix `draft → specified → planned → implementing → done`; Task `pending → in-progress → done`; Debt and Bug `open`, `accepted`, `resolved`; Practice `active`, `superseded`; Release `planned`, `shipped`. |
| **F3** | A file or directory is in the wrong place, or wrongly named | *F3* below |
| **F4** | A status and its trail disagree | *Which way to fix* below |
| **F5** | A feature's Gherkin is malformed | Across the feature's ```` ```gherkin ```` fences there is exactly one `Feature:` and at least one `Scenario:` (an `adopted` feature may have none yet, and so may a `retired` one that never had any). Every fence starts with a Gherkin keyword (`Feature:`, `Scenario:`, `Scenario Outline:`, `Background:`, `Rule:`) or a tag (`@…`). |
| **F6** | A plan and its tasks disagree | The plan's `# Tasks` links exactly the task files that exist — each as a Markdown link, `[01-x.md](<slug>/01-x.md)`. `depends-on` names sibling tasks by file name without `.md`, one as a bare value and two or more as a YAML list (`[01-x, 02-y]`), with no cycle. |
| **F7** | A release and the `version:` fields disagree | `fdf release <version>` writes the release's lists from the `version:` fields. A shipped release may list only `done` documents (and features retired since). |
| **F8** | A scenario has no test case | *F8* below |
| **F9** | A Context document is missing or still a stub | Stop, and run the **fdf-init** interview. Never invent Context content to clear it. |
| **F10** | A Change or Fix does not match reality | *F10* below |
| **F11** | A practice is malformed | A practice has a non-empty `# Rules` and no Gherkin. A `superseded` one names an existing replacement in `superseded-by`; an `active` one has no `superseded-by`. Its only sibling may be `<slug>.log.md`. |
| **F12** | The lexicon is inconsistent, or a banned word reached the bundle | *F12* below |
| **F13** | A debt is malformed | A debt has a non-empty `# Gap` concrete enough for someone else to confirm (files and counts), and no Gherkin. `accepted` needs `# Rationale`; `resolved` needs `# Resolution`. Its only sibling may be `<slug>.log.md`. |
| **F14** | A bug is malformed, or cites a scenario that is not there | *F14* below |
| **R1** | A `resource` or `applies-to` path does not exist | *R1* below |

## F1

- **A `timestamp` that is neither a date nor a time with a zone**
  (`2026-02-14 09:30`, `2026-02-14T09:30:00`, or a template's
  `<output of date -u …>` copied as it is):
  - it is already committed — `git show HEAD:<path> | grep '^timestamp:'`
    prints this same value → **never set it to now**. Restore the value git
    history holds from before it broke (`git log -p -- <file>`); otherwise
    **keep only the date**: `2026-02-14`; and when it holds no real date
    (`yesterday`, `2026-02-30`), the date of the commit that wrote it:
    `git log -1 --format=%as -S'<value>' -- <file>`. Never add a zone or a
    time yourself — a `Z` you guessed claims a moment nobody recorded. Only
    when the user tells you the zone it was written in, write the time with
    it: `2026-02-14T09:30:00+01:00`;
  - otherwise — `git show HEAD:<path>` fails (a new file) or prints another
    value: you wrote it in this session, or copied a template's placeholder →
    set it to the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run now.
- **Missing or unterminated frontmatter** → restore the `---` block at the top
  of the file, with its required fields (`type`, and the `status` its type
  needs).
- **`frontmatter is not parseable (F1): line N`** → frontmatter takes one
  `key: value` per line; a list is written `[a, b]`; no `|` or `>` blocks, no
  nested keys, and no `# comment` after a value (it becomes part of the
  value). Fix it first and validate again: F3, F10 or F14 errors about the
  same document — "has no sibling feature document", "names unknown feature" —
  are knock-on effects of the frontmatter nobody could read, and vanish with
  it.
- **A log date that is not `## YYYY-MM-DD`** → correct the heading.
- **The `fdf_version` pin** in the root `INDEX.md`. First decide which version
  the bundle was written for: **1.0 when it has `features/INDEX.md`**,
  otherwise the 0.x version it was written for.
  - it pins `0.x` → the bundle predates 1.0: its upgrade is the user's
    decision (fdf-help, *A bundle from before 1.0*). Never edit a 0.x pin to
    `"1.0"` by hand.
  - it is missing, or is not a `MAJOR.MINOR` version (`1.0.0`, `v1.0`,
    `0.7.1`), or is a 0.x version FDF never had (after 0.7) → correct it to
    the version the bundle was written for: `"1.0"`, or its 0.x version
    (`0.7.1` → `"0.7"`) — and a bundle written for 0.x is then upgraded
    (fdf-help, *A bundle from before 1.0*). When you cannot tell which 0.x
    version, `fdf migrate --dry-run` works from any 0.x layout: show the user
    its plan.
  - it is newer than this fdf supports → upgrade fdf.

## F3

- **A file the bundle has no place for** — a stray Markdown file at the root,
  a trail document inside a task directory, a directory beside a practice,
  debt or bug — has no ID, so `fdf mv` refuses it. Move it with `git mv` (or
  `mv` outside git): free-form documentation leaves the bundle, into the rest
  of `docs/`; a trail document goes beside its feature,
  `features/…/<slug>.<role>.md`. Then fix the links `fdf validate` reports as
  `broken cross-link`.
- **A document whose `type:` does not match its register** — a `type: Feature`
  file under `practices/` — has an ID, but `fdf mv` moves documents across
  registers only between `debts/` and `bugs/` (`fdf mv <debt-id>
  bugs/[<group>/…]<slug>` re-files a debt as a bug, and back). Otherwise: if
  the content belongs in this register, correct its `type:`; if not, move it
  with `git mv` into the right register and fix the links `fdf validate`
  reports as `broken cross-link`. Within one register, `fdf mv <id> <new-id>`
  renames or regroups a document and repairs every reference to it.
- **Casing** → directories and file names are lowercase; uppercase is only for
  the reserved files (`INDEX.md`, `LOG.md`, `SPEC.md`, `README.md`, the
  Context documents). No document is named `index.md` or `log.md`.
- **Roles**: beside a feature, `spec`, `plan`, `test`, `surface` and `log`;
  beside a Change or Fix only `spec`, `plan` and `log` (its `test` and
  `surface` belong to the affected feature); beside a practice, debt or bug
  only `log`. A task directory holds only `NN-<name>.md` task files.

## Which way to fix (F4)

A status and its trail disagree in one of three ways, and each takes a
different fix:

- **The trail is behind the status** — `planned` with no `slug.test.md`, `done`
  with a task still open. Build what is missing: fdf-plan for a missing plan or
  test document; fdf-execute for open tasks. For `done` with an open task: if
  the task's work is really finished, flip the task to `done`; if not, the
  feature is not done — set it back to `implementing` and continue with
  fdf-execute. Never demote a status just to dodge the rule.
- **The trail is ahead of the status** — a `slug.spec.md` (or any sibling but
  a log) beside a `draft`. If the step that wrote it is really complete (the
  spec was approved), flip the status in the same edit. If it is not, finish
  that step through its skill.
- **The status is ahead of reality** — someone marked `done` early. Set the
  status to what is actually true. That is the rule working as intended.

`implementing` with every task `done`: promote it to `done` — once that is
true. For a feature, every `slug.test.md` case has been run and passes
(fdf-execute's *Completion*); if one has not, the final task is not really
done, so set it back to `in-progress`. For a Change or Fix, its declared
effects must have landed in the affected features first, or F10 fails next.

F4 also covers an **adopted** feature: it must have **no** spec, plan or task
directory, and must name its code in `resource`. Never write a build trail to
satisfy F4 for code that predates its document; work on it is a Change.

`surface:` takes only the value `none`, and a feature with `surface: none` has
no `slug.surface.md`: keep whichever is true.

## F8

A scenario with no test case: add the case to the feature's `slug.test.md` —
a `## <scenario name>` heading under `# Test Cases`, the name copied exactly
as the Gherkin spells it (a bullet or a table row naming the scenario does not
count; a bundle upgraded from before 0.7 may still have them — turn each into
a heading), followed by its verification. Fix the test document to match the
scenario, never the scenario to match the test. A case that names no scenario
is a warning: if it is a second check of an existing scenario, move it under
that scenario's heading; if it checks behavior no scenario promises, the
scenario is missing — in a feature still being built, add it (fdf-plan,
step 3; ask when nobody decided it), and in a delivered one that is a Change.
Remove a case only when its scenario was removed, and never move a check out
of `# Test Cases` to silence the warning: outside it, F8 checks nothing.

## F10

F10 checks that a Change or Fix's **declared effects actually landed**. The
declaration is the contract; the features are the evidence. Find the message
below — **read the Fix rows first**: a message about a Fix is never a reason to
edit the feature's Gherkin.

| Message | What it means | Fix |
|---|---|---|
| `done, but <feature> has no scenario "X" — a fix proves scenarios that already exist` | A **Fix** names a scenario the feature does not have | Usually a typo: copy the scenario name character for character from the Gherkin. If no scenario covers the case, the document was silent: this is a **Change**, not a Fix. Convert it: set `type: Change` and `status: draft`, replace `# Regression cases` with `# Scenario changes` (`- add: <scenario>`), and take it through fdf-change's design gate before the scenario is added. Never add the scenario to the delivered feature to satisfy a Fix. |
| `<feature>.test.md has no case for "X"` | A Fix's regression case never reached the feature's living test document | If the row above names the same "X", correct the name there and this clears too. Otherwise add the case under the scenario's exact name — that case is the whole point of the Fix. |
| `done, but <feature> has no scenario "X"` (a **Change**) | The Change declared `add:` or `modify: X`, and the Gherkin does not have it | Amend the feature's Gherkin as declared — that is the work, not the error. When "X" starts `TODO —`, it is a scaffold line left behind: delete it, and any `- remove: TODO — …` line too (that one passes silently). |
| `done, but <feature> still has scenario "X"` | The Change declared `remove: X`, and it is still there | Remove it from the Gherkin, and its case from `slug.test.md`. |
| `affects names <feature> with status '<s>'` | The feature is not delivered | A Change or Fix is only for `done`, `adopted` or `retired` features. Delete this document and its `INDEX.md` listing line, and do the work as the feature's own tasks (fdf-execute) or design (fdf-brainstorm). |
| `` `affects` names unknown feature "…" — did you mean features/…? `` | A feature ID written the 0.7 way: every 1.0 ID starts with its register | Write the full ID the hint gives — in `affects`, `retires`, `replaced-by`, `depends-on` and the `## <feature-id>` headings alike. |
| `requires a # Scenario changes section` (or `# Regression cases`) | The wrong declaration section for the type | A Change declares `# Scenario changes`; a Fix declares `# Regression cases`. Never both. |
| `entry "…" is not add/modify/remove` (or `sits under no ## <feature-id> heading`) | A declaration line F10 cannot read, so it checks nothing | Write it as `- add: <scenario>`, `- modify: <scenario>` or `- remove: <scenario>`, lowercase, under the heading of the feature it changes. |
| `done, but the bug it resolves, bugs/<id>, is still 'open'` | The repair landed, and the register still says the defect is there | Flip the bug to `resolved` with a `# Resolution` naming this document. Never drop `resolves` to clear it. |
| `resolves names "…", which is not a bug` | A typo, or the bug was never filed | Point at the real bug ID (`bugs/<slug>`, or `bugs/<group>/<slug>`). A bug already cleared into `bugs/LOG.md` is fine once the document is done. |
| `status 'retired' but no done Change ... retires it` | A capability went dark with no reason recorded | Write the retiring Change, with a `# Rationale` (fdf-change, *Retiring a feature*). If it exists but is not `done`, land it: its `done` and the feature's `retired` go in one edit. |
| `done, and retires names X whose status is 'done'` | The Change landed, and the feature still reads as delivered | Flip the feature to `retired` in the same edit as the Change's `done`. |
| `retires names X ... until this Change is done, the feature it retires is still delivered` | The feature was flipped to `retired` before its Change landed | Put it back to `done` (or `adopted`) until the Change is `done`; then flip both in one edit. |
| `retires names X, which is not in affects` | Retiring a feature affects it | Add it to `affects`, with its `## <feature-id>` heading under `# Scenario changes` (left empty when no scenario changes). |
| `replaced-by names unknown feature` | The successor's ID is wrong, or it was never written | Point at the successor's real ID, or drop `replaced-by` when there is none. |

The fix for F10 is almost always **do the work you declared** — never edit the
declaration to match what you happened to do. If the declaration turned out
wrong, say so to the user and change it deliberately: that is a scope change,
not an error to silence.

## F12

Two different failures share this code.

**The lexicon itself is inconsistent** — a duplicate `## <Term>`, a term with
no definition, a word banned by one term that is another term's name, a word
claimed by two terms, an `except:` entry that contains none of its term's
banned words or is a banned word alone, a `strict:` that is not `true` or
`false`. These are errors in `DOMAIN.md`, and the fix is there: decide which
term owns the word, and say so. `DOMAIN.md` is a Context document, so the edit
needs the user's approval. Do not resolve it by deleting a term someone still
uses.

**A banned word reached a document or a name the bundle chose.** F12 reads
every document except `SPEC.md`, `DOMAIN.md`, `slug.test.md` and
`slug.surface.md` — features, specs, plans, tasks, changes, practices, debts,
bugs, logs, indexes, Context documents, finished work included — plus every
group, slug and task name. It skips what quotes rather than chooses: code
spans, non-Gherkin code blocks, link targets, URLs, HTML comments,
double-quoted text outside Gherkin, and a declaration's `## <feature-id>`
headings and verifications. Findings are `warn:` lines, one per document —
FAIL once `DOMAIN.md` sets `strict: true` or `--strict-domain` is passed:

```
warn: features/payments/refunds.spec.md: uses banned "store" ×3 → "Venue" (F12)
warn: features/stores/: name uses banned "store" → "Venue" — rename it with `fdf mv` (F12)
warn: and 37 more document(s) or name(s) use banned words — `fdf lexicon` lists every occurrence (F12)
```

First check the direction: if the banned word is what the team actually says
now, the lexicon is out of date, and that is a **Context-document change** —
propose it, and edit `DOMAIN.md` only on explicit approval. Never quietly drop
an `instead-of` entry to clear a warning.

**Sweep it, in this order.** A sweep is a lexicon fix (a maintenance edit): it
changes words and nothing else, needs no Change, is logged, and leaves every
`timestamp` as it is — F10 orders finished work by timestamp, and a sweep is
not new work.

1. **Report**: `fdf lexicon --all` lists every occurrence with
   `file:line:col`, grouped by word (without `--all` it shows a few per word).
2. **Triage the other senses.** A banned word often has an ordinary second
   meaning — *branch* in git, *integration* tests, a *log line*. Say which thing you mean: rephrase, or qualify it ("git branch") and
   propose adding the qualified phrase to the term's `except:` in `DOMAIN.md` —
   a lexicon edit, so it needs approval. An `except:` entry is a phrase, never
   the word alone.
3. **Mark the mentions.** Text *about* a word — a log entry recording that
   `store` was renamed to Venue — is not a use of it. Put the word in a code
   span; `fdf lexicon` flags italic ones, which are usually mentions.
4. **Fix one term at a time**: `fdf lexicon --term <Term> --fix --dry-run`,
   read the diff, then `fdf lexicon --term <Term> --fix`. It keeps plurals,
   capitals and "a"/"an" right, renames a scenario everywhere it is joined —
   Gherkin, `slug.test.md`, task acceptance, declarations — logs the sweep, and
   validates. What it cannot decide it lists and leaves: italic mentions, a
   label quoted in a Gherkin step, table cells, names.
5. **Names are moves**: `fdf mv <old-id> <new-id>` for each name the report
   lists, choosing the new name yourself — a mechanical one reads badly
   (`02-menu-from-catalog` → `02-catalog-from-catalog`).
6. Once the report is empty, propose `strict: true` in `DOMAIN.md`.

Rules for any lexicon fix, by hand or by tool: replace the banned word with its
term where it names the concept, and nothing else. A scenario name is a join —
change every copy of it in one edit, or F8 and F10 fail. A rename recorded
before the fix (`remove: Store owner…` with `add: Venue owner…`) ends up
removing and adding one name; F10 reads that as a replacement — leave it.

**The banned word is quoted user-facing copy** — `When I tap "Store
settings"`. The lexicon covers the project's internal language, not the words
a person sees on a surface. So:

- Write the step around the concept, not the label: `When the merchant opens
  the Venue settings`. In a delivered feature that is a lexicon fix too: the
  behavior is unchanged, so it is made in place.
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

## F14

- A bug needs a non-empty `# Symptom` (the reproduction, or the code path
  found by reading) and `# Expected`, and no Gherkin.
- `# Violates` cites scenarios the bug contradicts: every `## <feature-id>`
  heading under it is in `affects`, and while the bug is `open`, every name
  under it exists in that feature **character for character**. Cite the
  scenario exactly — or, when no scenario covers the case, delete
  `# Violates` (its repair is then a Change).
- While the bug is open, every feature under `# Violates` is delivered
  (`done` or `adopted`). In a feature still being built, the defect is its
  tasks' to repair: drop `# Violates`, and keep the scenario's name in
  `# Expected`.
- `accepted` requires `# Rationale` and forbids `# Violates`: a scenario the
  project knowingly keeps contradicting makes its feature lie — repair the
  code (a Fix) or change the scenario (a Change).
- `resolved` requires `# Resolution`. A resolved bug that still cites
  `# Violates` needs the done Fix or Change that `resolves` it: a bug is never
  repaired in place.
- A feature named the 0.7 way, in `affects` or a heading, gets the hint
  `did you mean features/…?`: write the full ID it gives.

## R1

A `resource:` (on a task, Change, Fix, debt, bug or adopted feature) or
`applies-to:` (on a practice) path does not exist. Find out **why** first:

- **The task will create the file** → the path does not belong in `resource:`
  yet. List the nearest directory that already exists instead:
  `src/refunds/policy.ts`, in a project with no `src/refunds/` yet → `src`. Never create a placeholder file or directory to satisfy R1.
  Leave `resource:` out when that directory would be the project root —
  never write `.`.
- **The code moved in a refactor** → update the path to where it lives now,
  even on a `done` task: a path is bookkeeping, not the record an episodic
  document keeps. This is a path repair (a maintenance edit): log it.
- **The code was deleted** → on a finished task, Change or Fix, remove the
  path (a path repair), and log what was removed. On a living document — a
  practice, an adopted feature — the document is out of date: amend it.
- **A debt's path vanished** → the gap is probably gone. Check; if so, flip
  the debt to `resolved` with a `# Resolution`, then `fdf debt --cleanup`: a
  resolved debt is still checked until cleanup retires it.

## Warnings

| `warn:` says | Do |
|---|---|
| `still holds a scaffold's placeholder text` | Replace every `TODO`, `<role>`-style and `Replace me` text with the real content; delete what does not apply. If you do not know the content, ask. |
| `missing recommended …` (a field name) | Add the field (a `timestamp` from `date -u +%Y-%m-%dT%H:%M:%SZ`). |
| `has no <slug>.surface.md — write one if …` | A decision, not a repair. (1) **Deleted by mistake?** Run `git log --oneline --diff-filter=D -- <path of the surface document>`; if it names a commit, ask whether the deletion was a mistake. If so, restore it with `git checkout <commit>^ -- <path>` — its old `timestamp` included, never retyped — then run `git show --stat <commit>` and restore every other part of that commit the user confirms was the same mistake, such as the feature's link to the document — those lines only, read from `git show <commit>^:<path>`. (2) **No such commit, or the deletion was intended**, and the feature's spec, tasks or Gherkin show a screen, an endpoint, a command or an event → write `slug.surface.md` (the layout is in fdf-brainstorm, *Surface document*), its `timestamp:` the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run now. (3) You cannot tell whether it has an interface → ask the user. Only a feature with no interface gets `surface: none`. |
| `case … names no scenario` | A second check of an existing scenario → move it under that scenario's heading. Behavior no scenario promises → add the scenario while the feature is being built (fdf-plan, step 3), or open a Change once it is delivered. Remove the case only when its scenario was removed. |
| `case … appears N times` | Merge them into one case. |
| `<slug>.test.md: … predates <change> …, which declared scenarios of … — the case it needed may be missing` | Check that the test document has, for each scenario that Change or Fix declared, the case it needed (for a Fix, the strengthened regression case). Add or update it, then set the test document's `timestamp`. When the test the case names was changed and now fails without the fix, and the case's text was already right, setting the `timestamp` is the whole repair to the document. If no test changed, the case never caught the defect: strengthen its test first (fdf-change, step 7). The warning compares timestamps, nothing else. |
| `broken cross-link -> <target>` | Point the link at where the document is now. A move made with `fdf mv` repairs links itself. |
| `open debt has no …resource` / `open bug has no …resource` | Add the paths that carry it, so work touching them finds it. |
| `active practice has no …applies-to` | Propose the paths it governs to the user — a practice edit needs approval. |
| `still an unfilled stub` / `freshly scaffolded stub` / `recommended context document is missing` | fdf-init. |
| `log entries are not in newest-first order` | Move the misplaced entry to its date, newest first, changing no words. |
| `uses banned "…"` (F12) | *F12* above. |
| `standalone bundle (no project root): skipping R1` | Nothing to do: R1 needs a project around the bundle. |

## Never silence a rule

The bundle exists so the next agent can trust it. These "fixes" all pass
validate, and all make the bundle lie:

| Tempting | Why it's a lie |
|---|---|
| Delete a `Scenario:` so F8 stops asking for a test case | You removed documented behavior instead of testing it. |
| Trim `slug.test.md` down to the scenarios you wrote | Same lie, other end. |
| Put `surface: none` on a feature that has an interface | The next person to change that endpoint or screen finds no record of what it is. Write the surface document. |
| Demote `done` → `implementing` to dodge an open-task error | Only correct if the feature is genuinely unfinished. |
| Drop a path from `resource:` without finding out why it is missing | The task now has no verifiable target. (A path the task will *create* never belonged there — list its nearest existing directory instead.) |
| Create an empty placeholder file so a `resource:` path exists | The bundle now claims work that has not happened. |
| Invent a zone or a time for a timestamp | It now claims a moment nobody recorded. Keep the date. |
| Bump a test document's `timestamp` with its case untouched, to clear the "predates" warning | It claims the case now catches the defect. First make the test fail without the fix. |
| Write plausible-sounding STACK.md text to clear F9 | Invented context is worse than none — every later feature is designed against it. |
| Hand-edit a 0.x `fdf_version` pin to `"1.0"` | The pin says which rules the bundle follows. `fdf migrate` moves a bundle into 1.0's layout and then pins it; a hand-edited pin claims a layout the bundle does not have. |
| Give a free-form note an FDF type so the closed root stops rejecting it | It becomes a document the bundle vouches for. Move free-form documentation out of the bundle. |
| Delete the file the error names | The error was about the file's content, not its existence — unless the file should not exist at all (a Change against an undelivered feature). |
| Drop an `instead-of` word so an F12 warning goes away | The synonym drift is the thing the lexicon exists to catch; you deleted the detector, not the problem. |
| Change a UI label, locale string or help text so a scenario stops tripping F12 | The lexicon covers internal language only; the surface may use that word on purpose. Reword the step around the concept. |
| Slip a behavior change into a lexicon fix | A lexicon fix changes wording only. A scenario whose meaning changes is a Change, with its design gate. |
| Empty a practice's `# Rules` to clear F11 | A practice with no binding statements is a blog post. Write the rules or delete the practice. |
| Set a practice to `active` so `superseded-by` stops being required | It says the old mechanism is still how the project works. It isn't. |
| Mark a debt `resolved` with a vague `# Resolution` to clear F13 | The rule exists so "paid" and "quietly dropped" cannot look the same. Say what closed it, or leave it open. |
| Hide a banned word in quotes, italics or a code span | Code spans and quotes mark a *mention* or a quoted label. Using one to disguise the word the document chose is the drift the check exists to catch. |
| Add a bare banned word to `except:` | That un-bans it everywhere. `except:` takes phrases that say which other thing is meant. |
| Rename a bundle file by hand | Some `affects`, heading or link is missed. `fdf mv` repairs every reference and logs the move. |
| Flip a bug to `resolved` after patching the code directly | A bug is never repaired in place. The repair is a Fix or Change that `resolves` it — that is what leaves the regression case. |
| Accept a bug that still cites a scenario, so F14 stops asking for a repair | Then the feature promises what the project has decided the software will not do. |
| Write a spec, plan and done tasks for an existing capability so it can be `done` | That invents a history. It is an `adopted` feature, with no build trail. |
| Delete a debt instead of resolving or accepting it | The register then records neither the gap nor the decision. `fdf debt --cleanup` clears **resolved** entries into `debts/LOG.md`; nothing else removes one. |
| Flip a debt to `accepted` because paying it is inconvenient now | `accepted` means the project decided to keep the gap, with a reason someone stands behind. "Not today" is still `open`. |
| Edit a Change's `# Scenario changes` to match what you actually did | The declaration is what someone approved. Changing it silently makes F10 check nothing. |
| Turn a Change into a Fix so the spec gate goes away | If the Gherkin changes, someone must decide it. That is the gate, not paperwork. |
| Add a scenario to a delivered feature so a Fix's regression case has something to name | A Fix names scenarios that already exist. A missing one means the Fix is a Change. |
| Edit a delivered feature's Gherkin directly to clear a mismatch | Then nothing records why it changed. Open a Change. (A lexicon fix is the exception: it changes words, not behavior.) |

If a rule looks genuinely wrong for a legitimate bundle, say so to the user and
stop — that is a spec or validator bug worth reporting, not something to work
around locally.

## Done

Exit 0, and you can state plainly which rules were failing and what you
changed for each. Report any remaining `warn:` lines. If you cleared an error
by changing a status, say which status and why it is now true.
