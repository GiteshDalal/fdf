---
name: fdf-help
description: Use when starting any conversation in a project with an FDF bundle (docs/fdf/, a docs/features/ from before 1.0, or FDF_ROOT_DIR), and before writing any code or bundle document there — including "quick", "tiny" and "just do it" changes. It says which fdf skill the work belongs to.
---

# Using FDF

## What FDF is

FDF (Feature Document Format) documents software features as a directory of
Markdown files — the **bundle**, at `docs/fdf/` in this project. Its root is
closed: it holds the five Context documents below, `INDEX.md` (which pins the
spec version), `LOG.md`, `SPEC.md`, an optional `README.md`, and six
**registers** — `features/`, `changes/`, `practices/`, `debts/`, `bugs/` and
`releases/` — and nothing else. Free-form documentation (guides, ADRs, notes)
belongs outside the bundle, in the rest of `docs/`. Every register but
`releases/` files its documents flat or in groups, and groups nest to any
depth.

Each feature is one Markdown + Gherkin file under `features/` with a
lifecycle `status` in its YAML frontmatter. Its trail lives beside it, in
files named after it:

| Role | Path | Required from |
|---|---|---|
| Spec | `features/…/<slug>.spec.md` | `specified` |
| Plan | `features/…/<slug>.plan.md` | `planned` |
| Test | `features/…/<slug>.test.md` | `planned` |
| Surface | `features/…/<slug>.surface.md` | when the feature has an interface: a screen, an endpoint, a command, an event |
| Log | `features/…/<slug>.log.md` | its first entry (`fdf log`) |
| Tasks | `features/…/<slug>/NN-<name>.md` | `implementing` (fdf-plan writes them with the plan) |

`…/` stands for the groups the feature is filed in: none for a flat feature
(`features/onboarding.md`), one (`features/payments/instant-refunds.md`), or
several (`features/platform/payouts/weekly-payouts.md`). The directory named
after a feature, beside it, is its **task directory**: it holds only task
files, never a trail document.

A capability the software had **before** the bundle existed is an **adopted**
feature (`status: adopted`): documented from the code as it stands, with its
code named in `resource`, and never a spec, plan or tasks, because nothing was
built through FDF. It may start as a *map entry* — a `Feature:` block and no
scenarios — and gain scenarios as work reaches it. fdf-adopt runs that.

Work on a feature after it is delivered lives under `changes/` as a `Change`
(the feature must behave differently) or a `Fix` (the code drifted from what
the feature document already says). Both name the features they touch in
`affects:`.

Five **Context documents** at the bundle root — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md` — hold the project's current stack,
architecture, interface principles for **every** surface (API, UI, CLI,
events — not UI only), build and deployment infrastructure, and domain
language. They are critical: filled once by the fdf-init interview, then
changed only with the user's explicit approval, and every change is logged.
Until filled, each carries a `<!-- fdf:stub -->` marker, and rule F9 blocks
feature work while any is still a stub.

`DOMAIN.md` is the project's vocabulary: one canonical name per concept, and
the words banned in its place. It governs the project's **internal** language
— every document in the bundle, and the identifiers in the code (types,
tables, fields, functions, routes, events). It does **not** govern what a
person reads on a surface — UI labels, button text, locale files, help text,
end-user messages — nor the names an external system fixes at its boundary. A
Venue may be shown to customers as "store" on purpose: that wording is a
surface decision (SURFACES.md), not drift. Never rewrite user-facing copy to
match the lexicon, and never report it as drift.

**Practices** (`practices/`, `type: Practice`) are the project's binding
answers to *how do we do X* for a recurring mechanism: authorization, payment
capture, database access. Before writing code in a path a practice's
`applies-to` covers, read it and follow its `# Rules`. Practices, like the
Context documents, change only with the user's explicit approval.

**Debts** (`debts/`, `type: Debt`) record a known gap between what the project
says and what the code does — work left undone, a rule the code does not
follow everywhere yet — that nobody can observe as wrong behavior.

**Bugs** (`bugs/`, `type: Bug`) record a known defect — the software doing
something observably wrong — that is not repaired yet. The line between the
two: if someone could observe the software misbehaving, it is a bug, even when
a practice violation is the cause. A bug is **never repaired in place**: its
repair is a `Fix` or `Change` that names it in `resolves` — or, while its
feature is still being built, one of that feature's tasks.

The bundle is the source of truth for what the software does. Code that
changes behavior without touching the bundle makes the bundle lie — the
failure FDF exists to prevent, and tiny changes are where it happens.

## Words the skills use

| Word | Meaning |
|---|---|
| **ID** | A document's path from the bundle root without `.md`, register first: `features/payments/instant-refunds`, `changes/refund-window`, `bugs/refund-split-capture`. Below, `<feature-id>`, `<change-id>` and `<bug-id>` are always full IDs. |
| **name in its register** | What a command that *creates* a document takes: `[<group>/…]<slug>` — `fdf new payments/instant-refunds` creates `features/payments/instant-refunds`. |
| **delivered** | A feature whose status is `done` or `adopted`. |
| **trail** | A feature's spec, plan, test, surface and log documents, and its tasks. |
| **living / episodic** | See *Living and episodic documents* below. |
| **join** | A scenario name repeated elsewhere — its `slug.test.md` case, a task's `# Acceptance`, a Change's declaration. Joins match character for character. |
| **maintenance edit** | One of the three edits allowed on any document (see below). |
| **the gate** | `fdf validate` exiting 0. |

## Mechanics every fdf skill relies on

- **Skills are not commands.** `fdf help` lists the commands. There is no
  `fdf plan`, `fdf brainstorm` or `fdf debug`: a skill is a procedure you
  follow. The spec, plan, test, surface and task documents are written by
  hand; only features, adopted features, Changes, Fixes, practices, debts and
  bugs are scaffolded by a command. Not every command has a skill: a Fix is
  scaffolded with `fdf fix` and written through fdf-change.
- **Flags go before the name.** `fdf change --affects <feature-id> <name>`,
  never `fdf change <name> --affects …` (refused).
- **Timestamps.** Every `timestamp:` you write is the output of
  `date -u +%Y-%m-%dT%H:%M:%SZ`, run at the moment of that edit. Never a time
  you estimate, round or reuse, and never a local time with `Z` added. Two
  exceptions: a document restored from git keeps its old `timestamp`
  (`git checkout <commit>^ -- <path>`; never retype it), and a malformed
  `timestamp` already committed is repaired by fdf-validate's F1 — never set
  to now.
- **Frontmatter.** A document you write by hand starts with `type`, `title`,
  `description` and `timestamp`, plus the fields its type needs. `fdf
  validate` warns about a missing one.
- **Placeholders.** A command that scaffolds a document leaves `TODO`,
  `<role>`-style and `Replace me` text in it, and adds a listing line to the
  `INDEX.md` beside it that ends `- TODO.` — and, for each group it creates,
  a `- TODO.` line in the parent `INDEX.md`. Replace all of it with real text
  before you finish; `fdf validate` warns until you do.
- **Log entries.** `fdf log <id> '<entry>'` writes to the log of the document
  the entry is about and creates that log on first use; with no ID it writes
  to the root `LOG.md`, which is only for the bundle as a whole. Use **single
  quotes**: inside double quotes the shell runs anything between backticks and
  the entry silently loses it. Start the entry with a bold label:
  `fdf log features/payments/instant-refunds '**Specified**: design approved by the user.'`
  An apostrophe would end the quotes — write ’ instead.
- **Filing.** A bug found while diagnosing (fdf-debug), mapping (fdf-adopt) or
  interviewing (fdf-init) is filed at once, and the user is told. Every other
  bug, every debt, and every practice or Context-document edit is proposed
  first, and written only once the user approves it.
- **Validate after every bundle edit.** `fdf validate` must exit 0 (use
  fdf-validate when it does not). Also read its `warn:` lines: fix those whose
  repair you know, such as a leftover placeholder; ask the user about those
  that need a decision.
- **Rule codes** (F1–F14, R1) are defined in `docs/fdf/SPEC.md`. It is long:
  read the section you need (`grep -n '^#' docs/fdf/SPEC.md` lists them)
  rather than all of it.

## Living and episodic documents

This distinction decides what you may touch when something changes:

| | Documents | When the software changes |
|---|---|---|
| **Living** — the system **today** | Feature Gherkin, `slug.test.md`, `slug.surface.md`, practices, debts, bugs, Context documents | **Amend in place** — for a delivered feature's Gherkin, test and surface documents, inside the Change, Fix or backfill that alters them. A living document describing behavior the software no longer has makes the bundle lie. |
| **Episodic** — a record of **one piece of work** | `slug.spec.md`, `slug.plan.md`, tasks, Changes, Fixes, log entries | Written and updated while the work is in flight; **frozen once its feature, Change or Fix is `done`**. After that, only a maintenance edit touches it, and new work gets a new episode. A log entry is never rewritten. |

So a delivered feature that changes keeps **one** feature document, whose
Gherkin is amended, and gains a **new** Change with its own spec. Never fork a
second feature document for the same capability.

Three **maintenance edits** may touch any document, living or episodic, at any
status, because each only puts one name in place of another. None needs a
Change or Fix; each is logged; none changes a `timestamp`:

- a **lexicon fix** — a word `DOMAIN.md` bans replaced by its term
  (`fdf lexicon --term <Term> --fix`; fdf-validate, F12), or put in a code
  span where the text is about the word itself — a log entry included;
- a **reference repair** — a document moved or renamed and every reference to
  it updated. `fdf mv` does the whole thing; never rename a bundle file by
  hand;
- a **path repair** — a `resource` or `applies-to` path updated to where the
  code now lives, or removed because the code was deleted. `fdf mv` moves
  bundle documents only; code moves with `git mv`, and its paths are
  repaired by hand (*Route the work*, step 3).

## Route the work — in this order

Status is the dispatch key: not the verb the user used, not the size of the
change. Before writing any code, and before answering "how should we build X",
go through these steps in order. Steps 1–5 choose the route: stop at the
first that decides it. Steps 6 and 7 then apply to **every** route — a
step-2 debug and a step-3 refactor included.

1. **Check the bundle.** Run `fdf validate`.
   - A `FAIL` about the `fdf_version` pin → fdf-validate's *F1* says what the
     bundle was written for. A bundle written for 0.x is upgraded first (*A
     bundle from before 1.0*, below): nothing else proceeds until it is.
   - A `still an unfilled stub` line (`FAIL` or `warn:`) about a Context
     document → **fdf-init**. Feature work waits until they are filled.
   - Any other `FAIL` → tell the user, and clear it with **fdf-validate**
     before other bundle work: every later step needs the gate.
2. **Is something broken?** A report that the software does something it
   should not — a bug report, a failing test, a crash, a regression, output
   that is wrong by what the documents say → **fdf-debug**, whatever the
   feature's status. fdf-debug finds the root cause, then tells you which
   route below the repair takes. (A test failing inside the task you are
   implementing is still that task's work: stay in fdf-execute.) A request to
   make the software behave *differently* ("show refunds in the order history
   too") is not a report of something broken: go on to step 3.
3. **Does the work change what the software does?** A pure refactor, a typo,
   tooling, a dependency bump, a CI change — nothing a user or another system
   can observe → no feature document. Renaming or moving code (a package, a
   directory, a file) is a refactor too: it is not a lexicon matter unless
   `DOMAIN.md` bans the old name. Do the work — move code with `git mv`:
   `fdf mv` moves bundle documents only — then:
   - repair any `resource` or `applies-to` path that names code you moved or
     deleted (a path repair). Find them with
     `grep -rn -A3 -e '^resource:' -e '^applies-to:' docs/fdf | grep '<old path>'`,
     edit each path by hand — never a log entry — run `fdf validate` (R1
     names any you missed), and log it. A debt or bug that names that code in
     its text is amended to match;
     a practice's prose is proposed to the user, like a Context document; an
     episodic document keeps the old path in its prose, as the record of that
     time;
   - if the work made a Context document untrue — a version in `STACK.md`, a
     directory in `ARCHITECTURE.md`, a command in `INFRA.md` — propose that
     edit to the user.

   No feature skill follows (steps 6 and 7 still apply). Repair the paths
   now, not at the next fdf-checkpoint.
4. **Find the feature the work is about.**
   - `fdf adopt` (with no arguments) prints every feature's ID and status;
     the list of unclaimed code below it is not needed here.
   - Search for the concept by its canonical name in `DOMAIN.md` — the user
     may be using a banned word:
     `grep -ril '<term>' docs/fdf/features --include='*.md'`.
   - Open the feature and read `status:` in its frontmatter.
   - No feature matches? Search the code as well: does the code already have
     the command, route, screen or job this work is about, with no feature
     documenting it?
5. **Route by what you found:**

   | What you found | Skill |
   |---|---|
   | No feature, and the work alters or extends a command, route, screen or job that **no feature documents** — even when the behavior asked for is new | **fdf-adopt** — map it, then route again by its new status (usually fdf-change). Never brainstorm existing behavior as if it were new |
   | No feature, and the work extends no undocumented command, route, screen or job — even when it sits beside code a feature documents (name that feature in `depends-on`) | **fdf-brainstorm** — a new feature |
   | `draft` | **fdf-brainstorm** — finish the design |
   | `specified` | **fdf-plan** |
   | `planned` or `implementing` | **fdf-execute** |
   | `done` or `adopted`, and its behavior must change, be repaired, or be removed | **fdf-change** — see *A delivered feature: Fix or Change?* |
   | `adopted`, and you only add scenarios describing what it already does | **fdf-adopt** (*Backfilling one scenario*) |
   | `retired`, and the capability is coming back | **fdf-brainstorm** — a new feature, which may name the retired one in `depends-on` |

   Other work routes here too:
   - the project's context may have drifted — periodically, before a release,
     after dependency or infrastructure work → **fdf-checkpoint**;
   - a bundle file was just edited, or `fdf validate` fails → **fdf-validate**;
   - a release → *Cutting a release*, below.
6. **Read what binds the work** before you write anything: `DOMAIN.md` (use its
   canonical names), and every practice, debt and bug whose paths overlap the
   paths you will touch. List their paths with:

   ```bash
   grep -rn -A3 -e '^applies-to:' -e '^resource:' docs/fdf/practices docs/fdf/debts docs/fdf/bugs
   ```

   (A bare `applies-to:` or `resource:` lists its paths on the `- ` lines
   below it.)

   A path overlaps when it is the same as, a parent of, or inside one you
   will touch. Follow the `# Rules` of each `active` practice that overlaps. A
   debt or bug that overlaps is already known: read it before you diagnose or
   design anything.
7. **Announce**: "Using fdf-<skill> — <feature-id> is <status>." For code no
   feature documents: "Using fdf-adopt — <capability> is not mapped."

A request may span stages: "get it implemented" on a `specified` feature means
fdf-plan, then fdf-execute. Chain them in order, and keep every stage's gates
and hand-offs — fdf-brainstorm's design approval, and fdf-plan's question
before execution starts. Never enter a later stage because the user named it.

## A delivered feature: Fix or Change?

When the work alters a `done` or `adopted` feature, one question decides which
document it needs. If the report is only a symptom ("it's broken"), you cannot
answer it yet: run fdf-debug first, and answer it with the root cause.

> **Does a scenario of the feature already say what should happen, and the
> code does something else?**

- **Yes** — the document was right, and the code drifted. That is a `Fix`:
  scaffolded with `fdf fix`, written through fdf-change. No design gate; the smallest Fix is one file. Its lasting
  artefact is the regression case in the feature's `slug.test.md`.
- **No** — someone is deciding what the software should do, including when the
  document was silent or wrong about the case. That is a `Change`:
  scaffolded with `fdf change`, written through fdf-change, and it needs the
  design gate. This includes a change to an interface detail no scenario
  names — a message's wording, a flag's name, an error code: how existing
  behavior is shown or invoked — which lives in `slug.surface.md`: the Change
  lists the feature under `# Scenario changes` with its `## <feature-id>`
  heading and no entries, and amends `slug.surface.md`. An option that makes
  the software do something new (another output format, a filter) is
  behavior: the Change adds its scenario.

A Change or Fix may name several features in `affects:` — one defect can show
in several features, and one request can span them.

**A new feature instead of a Change?** If the new behavior reads as its own
`Feature:` block — its own As a / I want / So that — it is a new feature
(fdf-brainstorm), which names the feature it builds on in `depends-on`. If it
alters what an existing capability visibly does, it is a Change.

**Repairing a bug on the register** is never plain code work. When the bug
cites a scenario under `# Violates`, its repair is a Fix:
`fdf fix --from <bug-id> <name>`. When it cites none, the repair is a Change:
`fdf change --from <bug-id> <name>`. Both run through fdf-change. A defect in
a feature still being built (`draft` to `implementing`) is that feature's own
task work, not a Fix or Change: the task that repairs a filed bug sets it to
`resolved`, with a `# Resolution` naming the task. In code no feature documents, adopt the
capability first (fdf-adopt).

**Paying down a debt** routes like any other work — usually plain code work,
since bringing code into line with a practice changes nothing a user sees.
When it lands, flip the debt to `resolved` with a `# Resolution` saying what
closed it.

## Does the work need a bundle document?

Ask: does it change what the software does for a user or another system, or
make the software do what the bundle already promises? Either way, yes:

- a new capability → a feature (fdf-brainstorm);
- altering a delivered capability → a `Change` (fdf-change);
- making the code match a delivered feature's documented behavior → a `Fix`
  (fdf-change). This is **not** an exemption: the regression case is what
  stops the bug coming back.

Only work that changes no behavior is exempt — pure refactors, typos,
tooling, dependency bumps. Exempt from a *feature* document, that is — not
from the Context documents: a dependency bump or a CI move can make `STACK.md`
or `INFRA.md` wrong. Propose that edit when you make the change, or run
fdf-checkpoint.

## Cutting a release

A release is bookkeeping, not a stage:

1. Run fdf-checkpoint.
2. Set `version: <version>` on each feature, Change and Fix that will ship in
   it — never on an `adopted` feature.
3. Run `fdf release <version>` straight away: until `releases/<version>.md`
   exists, `fdf validate` fails F7 on every `version:` that names it.
4. Once everything listed is `done`, run `fdf release --ship <version>`.

## A bundle from before 1.0

fdf 1.x works on spec 1.x bundles only. A bundle that pins 0.x — often still
at `docs/features/`, where bundles lived before 1.0 — fails `fdf validate`
with F1, and every other command that works on a bundle refuses it, pointing
at `fdf migrate`. The upgrade moves the bundle's documents and rewrites
references to them across the project, so it is **the user's decision**:
propose it, say what it does, and wait.

- **If the user declines**, bundle work waits: the project keeps fdf 0.7.x and
  its skills until it is ready (a version manager such as mise pins one per
  project). Never move files into 1.0's layout, or edit the pin, by hand.
- **If the user agrees:**
  1. **Start clean.** Commit first: `fdf migrate` runs only on a clean tree,
     and git is its undo.
  2. **`fdf migrate --dry-run`**, and read the plan with the user: what moves
     where, the files outside the bundle it rewrites, and each mention of the
     old path it leaves for a person to decide on. A file that must keep its
     bytes, such as an applied SQL migration a tool checksums, is left as it
     is with `--skip '<glob>'`. A refusal names what to fix first; nothing has
     been written.
  3. **`fdf migrate`**, with the dry run's `--skip` flags. It validates the
     result and exits with the validator's code. A bundle from before 0.7 may
     fail rules added since its version — a test case is a
     `## <scenario name>` heading, a `timestamp` carries its zone: clear them
     through fdf-validate. If migrate says a Context document was written as a
     stub, fill it with fdf-init.
  4. **`fdf install` again**, once for each agent the project uses, naming it
     and the bundle root: `fdf install --project --root docs/fdf claude-code`
     (or `codex`, `opencode`; leave out `--project` for a copy installed in
     your home directory). The skills and the primer then teach 1.0.
  5. **Review** `git diff -M --stat` and what migrate listed, then commit. Its
     report ends with the commands that back the migration out.
  6. **Run fdf-checkpoint.** An instruction file may still name a feature by
     its old ID: outside the bundle, migrate cannot tell a bare ID from a code
     path, and leaves it.

## "It's tiny, just do it"

Size never routes around FDF. The smallest compliant path is cheap:

- a **new** capability: `fdf new [<group>/…]<slug>` with one `Feature:` block
  and one `Scenario:`, a short spec the user approves, and a one-task plan with
  its one test case — minutes, and each skill scales down to match;
- a **delivered** capability: a one-file `fdf fix`, or a `fdf change` with a
  short spec.

State that cost once. If the user then explicitly opts out, their instruction
wins: do the work, then say plainly that the bundle now lacks this change. The
violation is the *silent* skip — and so is doing it first and asking later.

## Red flags — STOP, you are rationalizing

| Thought | Reality |
|---|---|
| "It's a one-line change" | Size doesn't route; status does. One line that changes behavior gets a document. |
| "The command exists but not this behavior, so it's a new feature" | When no feature documents that command, fdf-adopt maps it first; fdf-change then adds the behavior. |
| "It's only the message text — I'll just edit the surface document" | Wording a user sees is behavior. On a delivered feature it is a Change: its `## <feature-id>` heading with no entries, and the amended `slug.surface.md`. |
| "I'll use `fdf mv` to move this package" | `fdf mv` moves bundle documents only. Move code with `git mv`, then repair the paths that name it (*Route the work*, step 3). |
| "It's just a bug, no doc needed" | A bug's repair is a `Fix` or `Change` under `changes/`. The regression case it leaves is what stops the bug coming back. |
| "It's a bug, so it's a Fix" | Only when a scenario already says otherwise. A bug in a case the document never covered is a `Change` — fdf-debug decides, after the root cause. |
| "This feature is done, I'll make a v2 feature doc" | Two documents for one capability is the drift FDF exists to stop. Amend the feature; record the work as a Change. |
| "The feature is done, I'll just edit its Gherkin" | Then nothing records why it changed, and F10 never checks that the code followed. Open a Change — unless the edit is only a lexicon fix. |
| "The user said just do it" | Offer the smallest path first. Only an explicit opt-out after that counts. |
| "I'll backfill the docs later" | The bundle lies the whole time in between. Document first, code second. |
| "The full pipeline is process theater" | The trail is what the next agent trusts. Scale it down, don't skip it. |
| "User said 'implement', so fdf-execute" | Verb ≠ status. Read the frontmatter, and route by it. |
| "The user asked for it to be built, so I'll go straight from the plan into fdf-execute" | fdf-plan ends with a question: continue here, or in a fresh session? Ask it, and wait for the answer. |
| "I'll run `fdf plan` / `fdf debug`" | Skills are not commands. `fdf help` lists the commands; the plan, spec, test and task documents are written by hand. |
| "I'll flip statuses in a batch at the end" | Statuses reflect reality *now*: `in-progress` before working, task by task. |
| "Skip validate just this once" | `fdf validate` exit 0 is the gate after every bundle edit (use fdf-validate). The only wait is inside a set of edits that are valid only together, which the skills name. |
| "Validate failed, I'll just delete the scenario" | Never weaken content to silence a rule. fdf-validate has the honest fix for each code. |
| "This already exists in code — I'll write it up as a `done` feature" | That invents a spec, a plan and tasks that never happened. It is an `adopted` feature — fdf-adopt. |
| "I'll rename this feature file and fix the links myself" | A hand rename misses an `affects`, a heading or a link somewhere. `fdf mv` repairs every reference and logs the move. |
| "It's a known bug, so I'll patch it and close the bug" | A bug is never repaired in place. The repair is a Fix or Change that names it in `resolves` — that is what leaves the regression case. |
| "I'll add a back-link on the feature" | Don't. `affects:` is the whole link; `fdf history <feature-id>` computes the rest. |
| "I'll keep this design note in the bundle, at its root" | The root is closed (F3). A document of an FDF type goes in its register; a free-form note goes outside the bundle, in the rest of `docs/`. |
| "The bundle pins 0.7 — I'll just set the pin to 1.0" | 1.0 moved every feature into `features/` and gave every feature ID its register. `fdf migrate` does that in one run; a hand-edited pin claims a layout the bundle does not have. |
| "I know roughly what time it is" | You don't. `date -u +%Y-%m-%dT%H:%M:%SZ`. |

## Precedence

Explicit user instructions outrank skills. But "quick", "we're late" and
"don't waste tokens" are pressure, not opt-outs — the only opt-out is the user
declining the smallest path after you have named it.
