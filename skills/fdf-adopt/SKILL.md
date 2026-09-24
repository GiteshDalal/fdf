---
name: fdf-adopt
description: Use when an FDF project's capabilities already exist in code with no feature document — mapping an existing codebase into the bundle in phases, backfilling scenarios for an adopted feature, or before changing or fixing code that no feature documents yet. Not for new capabilities (fdf-brainstorm).
---

# FDF Adopt

Bring a codebase that predates its bundle into FDF — honestly, and in phases.

New to FDF? Run `fdf spec` for the format rules (see *Adopted features*) and
`fdf help adopt` for the command. The fdf-help skill explains how the skills
fit together.

**Read the Context docs first** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md`. They must be filled (fdf-init) before
any feature exists, adopted ones included; F9 enforces it. Name everything in
`DOMAIN.md`'s vocabulary: F12 reads feature names and Gherkin like any other
document.

## Why adoption is its own path

A capability the software already has was never specified, planned or built
through FDF. Writing it a `slug.spec.md`, a plan and done tasks now would
invent a history that never happened — the one thing an episodic document must
never be — and leaving it out of the bundle means every change to it happens
around the bundle. So it enters as an **adopted** feature:

| | An adopted feature |
|---|---|
| Build trail | **None**: no `slug.spec.md`, no `slug.plan.md`, no task directory — ever |
| `resource` | **Required**: the code it lives in. With no tasks, it is the only link from the document to the code |
| Scenarios | **Zero or more.** A `Feature:` block and `resource` alone make a *map entry* |
| `slug.test.md` | From the first scenario, one case per scenario, pointing at an existing test or an explicit manual procedure |
| Delivered? | Yes: a `Change`, `Fix` or `Bug` can name it in `affects`. It never becomes `done` |
| `version` | None: it shipped before the bundle recorded releases |

Adoption is for code that predates its capability's first document. Code
written last week is not adopted — it goes through fdf-brainstorm, because
skipping the design gate is exactly the drift FDF exists to stop.

## Phases

A large codebase is adopted breadth first, then deepened where work happens.
Run `fdf adopt` (no arguments) at any point: it prints every feature with its
scenario and test counts, and the tracked code no document claims yet, most
unclaimed first. That list is the plan. Files that are no capability — build
manifests, lockfiles, CI and editor configuration — stay unclaimed, and that
is expected: the map is done when every capability is on it, not when the
list is empty.

### Phase 1 — Map

Goal: every capability the software has is on the map — an ID, a group, its
code — before anyone documents how it behaves. It is cheap, and it is what lets
every later piece of work find the capability it is about to touch.

1. **Survey the entry points**, not the modules: routes and RPCs, CLI commands,
   scheduled jobs, event consumers, UI screens. Group them by what a person or
   calling system would call *one thing they can do*. That is a capability.
   Not one per endpoint — "refund a payment" is one capability behind three
   RPCs — and not one per module; a module usually holds several. One
   actor's job is one capability, and its visible effect on others belongs
   to it (an owner closes a date; customers see it closed). Two jobs that
   stand on their own are two capabilities, even on the same data (an owner
   manages a catalog; customers browse it): each is described, and later
   changed, without the other.
2. **Propose a group at a time**: the capability list for one area, each with
   its proposed `<group>/<slug>`, a one-line purpose, and the paths it lives
   in. Groups follow `ARCHITECTURE.md`'s map or the product's own areas. Wait
   for approval before scaffolding — the names become IDs every later
   document links to.
3. **Scaffold each** with
   `fdf adopt --resource <path>[,<path>…] <group>/<slug>`. The paths must
   exist; prefer the directory or file that holds the capability over a whole
   module. A file every capability passes through — a route table, a wiring
   module — goes on each capability it registers, after the capability's own
   code, not instead of it. An entry point whose code you cannot find is still
   mapped, with the file that declares it as its `resource`; say in its prose
   that the code behind it was not found — and if nothing serves it at all,
   that is a defect (below).
4. **Fill the `Feature:` block** truthfully — who uses this, what they do with
   it, why — and the one-line `description`. No scenarios yet. Code rarely
   names its actor: ask when it matters; in a batch the user pre-approved, name
   the most likely actor and say in the prose that it was inferred. Replace
   every scaffold placeholder, in the feature and in the group's `INDEX.md`
   entry — `fdf validate` warns about any left behind.
5. **What else you find, file — do not act on.** Mapping is when defects and
   drift surface, because it is the first time anyone reads the code for what
   it does:
   - a **defect** (the code visibly does something wrong, a route served by
     nothing) → `fdf bug --affects <new-id> --resource <path> <slug>`, with
     `# Symptom` saying it was found by reading and `# Expected` naming the
     open question. Never repaired during mapping, never backfilled.
   - a **banned word in a code name** (a type, function or table) → propose a
     debt (`fdf debt`); renaming code is plain code work for later.
   - a **Context document the code contradicts** → note it for fdf-checkpoint.

   Filing a bug is part of mapping. Proposing a debt waits for approval like
   any other debt. If the user limited the session to mapping alone, list
   each finding with the command that would file it instead.
6. **Log and gate**: one bundle-root `LOG.md` entry per batch, since a batch
   concerns the bundle rather than one feature —
   `fdf log "**Adopted**: 14 capabilities mapped in payments/ and orders/."` —
   link any new group from the root `INDEX.md`, then `fdf validate` exit 0.

### Phase 2 — Backfill on touch

Before a change reaches an adopted capability, its document has to be able to
carry the change:

- **Backfill what the Change will alter.** Ask it of the code, not of the
  document: what does the capability do *today* that will be different, or
  gone, once the Change is done? Backfill each such behavior first, as a
  scenario describing it as it is, so the Change can `modify:` or `remove:`
  it and the bundle records what changed rather than only what is new. A
  map entry has no scenarios, so this question decides, not the scenario
  list: "hide out-of-stock products" alters what the list shows today, so
  "An out-of-stock product appears in the product list" is backfilled first.
  A Change that adds behavior the code does not have at all needs no
  backfill, and neither does a defect the Change repairs: wrong behavior is
  never backfilled (see *Backfilling one scenario*). Then route the work to
  fdf-change.
- **Record the interface the Change will alter.** When the Change touches an
  endpoint, a screen, a command or an event, write the capability's
  `slug.surface.md` from the code as it stands first (see *Recording an
  adopted capability's surface*), so the Change amends it.
- A capability with no map entry is mapped first (Phase 1, one capability).
- A defect: see *Defects in adopted and undocumented code* below.

### Phase 3 — Backfill by risk

With the map complete, deepen where being wrong costs most: money, access
control, data loss and privacy first; then the paths agents change most often
(`git log --format= --name-only | sort | uniq -c | sort -rn` ranks them).
There is no quota. A capability with three scenarios that matter beats one with
thirty nobody checks.

### Phase 4 — Steady state

New capabilities go through the normal lifecycle (fdf-brainstorm).
Post-delivery work on adopted ones goes through fdf-change. The map keeps
itself honest: `fdf adopt` shows what is still unclaimed.

## Backfilling one scenario

A backfilled scenario is a **promise about what the code already does** — and
the proof that it already does it is part of the same edit.

1. **Read the code and its existing tests.** Find the behavior, and the test
   that already exercises it if there is one.
2. **Write the scenario** in declarative Gherkin, in `DOMAIN.md`'s terms,
   describing what the software does **today** — never what it should do.
3. **Write its `slug.test.md` case** — a `## <scenario name>` heading under
   `# Test Cases`, the name exactly as in the Gherkin (F8 matches it
   exactly), then the existing test's command or path, or an explicit manual
   procedure.
4. **Run it against the code as it stands.** No code change in this edit.
5. **It passes** → keep both, set the feature's `timestamp` (and the test
   document's) to now, in UTC, and `fdf validate`. Log the batch in the
   feature's own log: `fdf log <group>/<slug> "**Backfilled**: 3 scenarios, each proven by an existing test."`
6. **It fails** → stop. Either you misread the code (rewrite the scenario to
   what the code actually does, and run again), or you found a defect. A defect
   is never backfilled — not as it is, which would promise it, and not as it
   should be, whose test fails. File it (`fdf bug --affects <group>/<slug> …`)
   with its `# Symptom` and `# Expected`.

Backfill only **adds**. Once written, a scenario is a promise like any other:
modifying or removing one is a `Change` with its design gate, exactly as for a
`done` feature.

## Recording an adopted capability's surface

When the capability has an interface — an endpoint, a screen, a command, an
event — its `slug.surface.md` records that interface as the code has it today:
the request and response shapes and codes, the screen's states and copy, the
flags and exit codes, the event's fields. Read it from the code and the
existing API descriptions; never design it. Like a backfilled scenario, it
describes what is, and anything wrong it shows is a bug to file, not a surface
to correct. Write it when backfilling reaches the capability, or before a
Change alters its interface — that Change then amends what is already there.
A capability with no such interface says `surface: none` in its frontmatter.
Once an adopted feature has a scenario, `fdf validate` warns until it has
one or the other; a map entry is not asked yet.

## Defects in adopted and undocumented code

Start with fdf-debug: root cause first. Then:

- **The code is on the map** — the defect's repair is a `Change` when no
  scenario covers the case (the usual one: nobody wrote the promise down, so
  someone decides it now — for an obvious case the design gate is a one-line
  approval), or a `Fix` when a backfilled scenario already says otherwise.
- **The code is not on the map** — map the capability first (Phase 1, one
  entry), add it to the bug's `affects`, then repair it as above. A defect is
  never repaired around the bundle because its code was undocumented: that
  repair would leave no scenario and no test behind, and the defect would
  return unnoticed.
- **Not repairing it now** — it is a `Bug` on the register, with everything the
  investigation learned.

## Red flags — STOP

| Thought | Reality |
|---|---|
| "I'll write a quick spec and plan so it can be `done`" | That invents a history. Adopted features have no build trail, by design. |
| "The code clearly should do X — I'll backfill X" | Backfill documents what the code does. If it does not do X, that is a Bug or a Change, never a backfilled scenario. |
| "I'll fix this small thing while backfilling" | A backfill edit changes no code. The fix is its own work, routed through fdf-debug. |
| "Map every endpoint as its own feature" | Capabilities, not endpoints. A map nobody can read is not a map. |
| "Let me backfill all the scenarios before anything else" | Breadth first, depth where work happens. Full backfill of a large codebase is a project nobody finishes. |
| "This new code has no document yet — I'll adopt it" | Adoption is for code that predates its document. New work goes through fdf-brainstorm and its design gate. |
| "The backfilled scenario was wrong; I'll edit it" | If the code does something else, it was a misread only before it landed. Once written, changing it is a Change. |
| "It's only the mapping phase, so the bug I just spotted can wait" | File it — `fdf bug`, found by reading. Mapping is when defects surface, and one known only to a chat log is found again from zero. |
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
