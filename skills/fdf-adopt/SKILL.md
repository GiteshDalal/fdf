---
name: fdf-adopt
description: Use when an FDF project's capabilities already exist in code with no feature document — mapping an existing codebase into the bundle in phases, backfilling scenarios for an adopted feature, or before changing, extending or fixing a command, route, screen or job that no feature documents yet. Not for capabilities with no code yet (fdf-brainstorm).
---

# FDF Adopt

Bring a codebase that predates its bundle into FDF — honestly, and in phases.

New to FDF? The fdf-help skill explains how the fdf skills fit together and
the mechanics they share. The format rules are in the bundle at
`docs/fdf/SPEC.md` (see *Adopted features*); `fdf help adopt` documents the
command.

**Read the Context documents first** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md`. They must be filled (fdf-init) before
any feature exists, adopted ones included (F9). Name everything in
`DOMAIN.md`'s vocabulary: F12 reads feature names and Gherkin like any other
document.

Came here straight, not through the fdf-help skill? Do these first:

1. Run `fdf validate`.
2. Run `fdf debt --open` and `fdf bug --open`, and
   `grep -rn -A3 -e '^applies-to:' -e '^resource:' docs/fdf/practices docs/fdf/debts docs/fdf/bugs`;
   name every entry whose paths overlap the code you will map.
3. Say "Using fdf-adopt — <capability> is not mapped."

## Mechanics

- `<feature-id>` is a feature's full ID, such as
  `features/payments/card-payments`; paths are relative to the bundle root,
  `docs/fdf/`.
- A `timestamp:` you set is the output of `date -u +%Y-%m-%dT%H:%M:%SZ`, run
  at the moment of the edit — never a time you estimate, round or reuse, and
  never your local time with `Z` added. A restored document keeps its old
  one; a malformed one already committed is repaired by fdf-validate's F1,
  never set to now.
- Log with single quotes — `fdf log '**Adopted**: …'` — because inside double
  quotes the shell runs anything between backticks. An apostrophe would end
  the quotes: write ’ instead.
- `fdf adopt --resource …` scaffolds a map entry with placeholders and a
  `timestamp` (keep it), adds a listing line with a `TODO.` description to
  the `INDEX.md` beside it, and — when it creates a group — a `TODO.` line for
  the group in its parent's `INDEX.md`: replace every one with real text.

## Why adoption is its own path

A capability the software already has was never specified, planned or built
through FDF. Writing it a `slug.spec.md`, a plan and done tasks now would invent
a history that never happened — the one thing an episodic document must never
be — and leaving it out of the bundle means every change to it happens around
the bundle. So it enters as an **adopted** feature:

| | An adopted feature |
|---|---|
| Build trail | **None**: no `slug.spec.md`, no `slug.plan.md`, no task directory — ever |
| `resource` | **Required**: the code it lives in. With no tasks, it is the only link from the document to the code |
| Scenarios | **Zero or more.** A `Feature:` block and `resource` alone make a *map entry* |
| `slug.test.md` | From its first scenario: one case per scenario, pointing at an existing test or an explicit manual procedure |
| Surface | From its first scenario: `slug.surface.md`, or `surface: none` |
| Delivered? | Yes: a `Change`, `Fix` or `Bug` can name it in `affects`. It never becomes `done` |
| `version` | None: it shipped before the bundle recorded releases |

Adoption is for code that predates its capability's first document. Code
written last week is not adopted — it goes through fdf-brainstorm, because
skipping the design gate is exactly the drift FDF exists to stop.

## Phases

A large codebase is adopted breadth first, then deepened where work happens.

### Phase 1 — Map

Goal: every capability the software has is on the map — an ID, a group, its
code — before anyone documents how it behaves. It is cheap, and it lets every
later piece of work find the capability it is about to touch. Map entries
carry no scenarios yet; backfilling is Phase 2 and 3.

1. **Survey the entry points**, not the modules: routes and RPCs, CLI commands
   and subcommands, scheduled jobs, event consumers, UI screens. List every
   one. Then run `fdf adopt` (no arguments): it prints every feature with its
   scenario and test counts, and the tracked code no feature, task, Change or
   Fix names in `resource` yet. Use that list as a hint, never as the plan: a
   single broad `resource` (a task that names `internal`) claims every file
   below it, so code can look claimed when no feature documents it. An entry
   point no feature describes is unmapped, whatever the list says. Files that
   are no capability — build manifests, lockfiles, CI and editor
   configuration — stay unclaimed, and that is expected: the map is done when
   every capability is on it, not when the list is empty.
2. **Group the entry points into capabilities** — what a person or calling
   system would call *one thing they can do*. Not one per endpoint ("refund a
   payment" is one capability behind three RPCs) and not one per module (a
   module usually holds several). One actor's job is one capability, and its
   visible effect on others belongs to it (an owner closes a date; customers
   see it closed). Two jobs that stand on their own are two capabilities, even
   on the same data (an owner manages a catalog; customers browse it).
3. **Propose a group at a time** to the user: the capabilities of one area,
   each with its proposed `[<group>/…]<slug>`, a one-line purpose, and the
   paths it lives in. Groups follow `ARCHITECTURE.md`'s map or the product's
   own areas, and nest where an area is large. **Wait for approval** before
   scaffolding: the names become IDs every later document links to.
4. **Scaffold each approved capability**:
   `fdf adopt --resource <path>[,<path>…] [<group>/…]<slug>` —
   `fdf adopt --resource internal/payments/card.go payments/card-payments`
   writes `features/payments/card-payments.md`, ID
   `features/payments/card-payments`. Every path must exist. Prefer the
   directory or file that holds the capability over a whole module. A file
   every capability passes through — a route table, a wiring module — goes on
   each capability it registers, after the capability's own code. An entry
   point whose code you cannot find is still mapped, with the file that
   declares it as its `resource`; say in its prose that the code behind it was
   not found — and if nothing serves it at all, that is a defect (step 6).
5. **Fill it in truthfully**: the `Feature:` block (who uses this, what they do
   with it, why) and the one-sentence `description`, and the same description
   in its `INDEX.md` listing line. No scenarios yet. Take the actor from a
   sibling feature's `As a` line, else `SURFACES.md`, else the code's
   comments — in `DOMAIN.md`'s term where the source uses a banned word —
   and use the same words in the description; when none of them names one,
   ask. Say in the prose where the actor came from.
6. **Read each capability's code once for defects and gaps**, before you log
   the batch, and tell the user the result of each of the five checks below,
   even when it finds nothing. Mapping is the first time anyone reads the code
   for what it does, so this is when defects and drift surface. File what you
   find; do not act on it:
   - (a) a **defect** — the code does something observably wrong (a refund that
     can exceed its payment, a route served by nothing). Behavior no document
     decides still counts: if someone could lose by it — money, access, data,
     a promise made to them — it is a defect. Whether it was intended is the
     user's call, so `# Expected` asks. Reproduce it if you can, then file
     it:
     `fdf bug --affects <feature-id> --resource <path> [<group>/…]<slug>`.
     `--resource` names the paths the defect runs through, so work that
     touches them finds the bug. Fill it in: `# Symptom` — the commands, their
     output and `Reproduced.`, or the code path at `<file>:<line>`, the input
     that reaches it and `Found by reading.`; `# Expected` — first the
     sentence `No scenario decides this, so its repair is a Change.`, then
     the open question, written as a question, never as your answer; delete
     the scaffold's `<!-- # Violates … -->` comment (a map entry has no
     scenario to cite); keep the scaffold's `timestamp`. Tell the user each
     bug you file. Never repair it during mapping, and never backfill it;
   - (b) **tests** — nothing exercises the capability → check
     `fdf debt --open`: if a debt names it, say so; otherwise propose one;
   - (c) **practices** — compare the code with the `# Rules` of each practice
     whose paths overlap it;
   - (d) **conventions** — reopen `ARCHITECTURE.md` and `SURFACES.md` and
     check the code against every convention they state, bullet by bullet,
     naming each in your report. Code that breaks a convention the rest of
     the code keeps is a debt: find every place that breaks it (search for
     the pattern) and name them all; check `fdf debt --open` first, and
     propose one only if none names it. A Context document the code as a
     whole no longer matches → tell the user, and suggest fdf-checkpoint;
   - (e) a **banned word in a code name** (a type, function or table) →
     propose a debt (`fdf debt`); renaming code is plain code work for later.

   A debt is filed only once the user approves it. If the user limited the
   session to mapping alone, list each finding with the command that would
   file it instead.
7. **Log and gate**: one root `LOG.md` entry per batch, since a batch concerns
   the bundle rather than one feature —
   `fdf log '**Adopted**: 3 capabilities mapped in features/payments/: card payments, refunds, payouts.'`
   — then `fdf validate` exit 0.

### Phase 2 — Backfill on touch

Before a Change or Fix reaches an adopted capability, its document has to be
able to carry the change. In order:

1. **Map it first** if it has no map entry (Phase 1, one capability, the ID
   approved).
2. **A defect?** Then see *Defects in adopted and undocumented code* below: a
   defect is never backfilled.
3. **Backfill what the Change will alter.** Ask it of the code, not of the
   document: what does the capability do *today* that will be different, or
   gone, once the Change is done? Backfill each such behavior first, as a
   scenario describing it as it is (*Backfilling one scenario*, below), so the
   Change can `modify:` or `remove:` it and the bundle records what changed
   rather than only what is new. A map entry has no scenarios, so this
   question decides, not the scenario list: "hide out-of-stock products"
   alters what the list shows today, so "An out-of-stock product appears in
   the product list" is backfilled first. A Change that only adds behavior the
   code does not have needs no backfill.
4. **Record the interface the Change will alter.** When the Change touches an
   endpoint, a screen, a command or an event, write the capability's
   `slug.surface.md` from the code as it stands first (*Recording an adopted
   capability's surface*), so the Change amends it.
5. **Route the work to fdf-change.**

### Phase 3 — Backfill by risk

With the map complete, deepen where being wrong costs most: money, access
control, data loss and privacy first; then the paths agents change most often
(`git log --format= --name-only | sort | uniq -c | sort -rn` ranks them).
There is no quota. A capability with three scenarios that matter beats one with
thirty nobody checks.

### Phase 4 — Steady state

New capabilities go through the normal lifecycle (fdf-brainstorm).
Post-delivery work on adopted ones goes through fdf-change. `fdf adopt` keeps
showing what is still unclaimed.

## Backfilling one scenario

A backfilled scenario is a **promise about what the code already does** — and
the proof that it already does it is part of the same edit.

1. **Read the code and its existing tests.** Find the behavior, and the test
   that already exercises it if there is one.
2. **Write the scenario** in the feature document, in declarative Gherkin and
   `DOMAIN.md`'s terms, describing what the software does **today** — never
   what it should do. A map entry has no `# Scenarios` heading yet: add one,
   with one ```` ```gherkin ```` fence per scenario.
3. **Write its test case.** The feature's first scenario also creates
   `<feature-id>.test.md`:

   ```markdown
   ---
   type: Test
   title: Card payments — test cases
   description: How each card-payment scenario is proven.
   timestamp: <output of date -u +%Y-%m-%dT%H:%M:%SZ, run now for this file>
   ---

   # Test Cases

   ## A declined card is not charged

   `go test ./internal/payments -run TestCharge_Declined` passes.
   ```

   One `## <scenario name>` heading per scenario, the name exactly as in the
   Gherkin (F8), then the existing test's command or path, or an explicit
   manual procedure, and what passing looks like.
4. **Run it against the code as it stands.** No code change in this edit.
5. **It passes** → keep both. Set the feature's and the test document's
   `timestamp`. At the feature's first scenario, also write its
   `slug.surface.md` or add `surface: none` (*Recording an adopted
   capability's surface*). Run `fdf validate`, and log the batch in the
   feature's own log:
   `fdf log <feature-id> '**Backfilled**: 3 scenarios, each proven by an existing test.'`
6. **It fails** → stop. Either you misread the code (rewrite the scenario to
   what the code actually does, and run it again), or you found a defect. A
   defect is never backfilled — not as it is, which would promise it, and not
   as it should be, whose test fails. Remove the scenario and its test case
   again, then file the defect
   (`fdf bug --affects <feature-id> --resource <path> …`) with its `# Symptom`
   and `# Expected`.

Backfill only **adds**. Once written, a scenario is a promise like any other:
modifying or removing one is a `Change` with its design gate, exactly as for a
`done` feature.

## Recording an adopted capability's surface

When the capability has an interface — an endpoint, a screen, a command, an
event — its `slug.surface.md` (frontmatter `type: Surface`, `title`,
`description`, `timestamp`; one `#` heading per interface) records that
interface as the code has it today: the request and response shapes and
codes, the screen's states and copy, the flags and exit codes, the event's
fields. Read it from the code and the existing API descriptions; never design
it. Like a backfilled scenario, it describes what is, and anything wrong it
shows is a bug to file, not a surface to correct. Write it when backfilling
reaches the capability, or before a Change alters its interface. A capability
with no such interface says `surface: none` in its frontmatter. Once an
adopted feature has a scenario, `fdf validate` warns until it has one or the
other; a map entry is not asked yet.

## Defects in adopted and undocumented code

Start with fdf-debug: root cause first. Then:

- **The code is on the map** — the repair is a `Change` when no scenario
  covers the case (the usual one: nobody wrote the promise down, so someone
  decides it now — for an obvious case the design gate is a one-line
  approval), or a `Fix` when a backfilled scenario already says otherwise.
- **The code is not on the map** — map the capability first (Phase 1, one
  entry), add it to the bug's `affects`, then repair it as above. A defect is
  never repaired around the bundle because its code was undocumented: that
  repair would leave no scenario and no test behind, and the defect would
  return unnoticed.
- **Not repairing it now** — it is a `Bug` on the register, with everything
  the investigation learned.

## Red flags — STOP

| Thought | Reality |
|---|---|
| "I'll write a quick spec and plan so it can be `done`" | That invents a history. Adopted features have no build trail, by design. |
| "The code clearly should do X — I'll backfill X" | Backfill documents what the code does. If it does not do X, that is a Bug or a Change, never a backfilled scenario. |
| "I'll fix this small thing while backfilling" | A backfill edit changes no code. The fix is its own work, routed through fdf-debug. |
| "Map every endpoint as its own feature" | Capabilities, not endpoints. A map nobody can read is not a map. |
| "Let me backfill all the scenarios before anything else" | Breadth first, depth where work happens. Full backfill of a large codebase is a project nobody finishes. |
| "`fdf adopt` shows nothing unclaimed, so everything is mapped" | The list is a heuristic: one broad `resource` of an adopted feature, a task, a Change or a Fix (`internal`, `src`) hides everything below it — `grep -rn -A3 '^resource:' docs/fdf/features docs/fdf/changes` shows which. Check every entry point against the features. |
| "This new code has no document yet — I'll adopt it" | Adoption is for code that predates its document. New work goes through fdf-brainstorm and its design gate. |
| "The backfilled scenario was wrong; I'll edit it" | If the code does something else, it was a misread only before it landed. Once written, changing it is a Change. |
| "It's only the mapping phase, so the bug I just spotted can wait" | File it — `fdf bug`, found by reading. A defect known only to a chat log is found again from zero. |
| "It's probably a design choice" | Nobody wrote that down. File it; its `# Expected` asks the user. |
| "I know roughly what time it is" | You don't. `date -u +%Y-%m-%dT%H:%M:%SZ` — and a timestamp a scaffold wrote stays as it is. |
| "The name the code uses is fine for the feature ID" | IDs are names the bundle chooses: they follow `DOMAIN.md`, and F12 reads them. A code name that uses a banned word is a debt to propose, not a precedent. |

## Rules

- An adopted feature never gets a spec, plan or task directory; work on it gets
  its own Change.
- `resource` names code that exists; `fdf adopt` refuses a path that does not.
- A backfilled scenario describes today's behavior and arrives with a test case
  that passes against unchanged code.
- Backfill adds; modifying or removing a scenario is a Change.
- Map in batches the user approves; IDs are hard to take back.
- `fdf validate` exit 0 after every bundle edit.
