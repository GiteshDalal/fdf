# FDF — Feature Document Format

**Documentation-as-a-directory for software features.** Each feature is a
Markdown + Gherkin document; its design spec, plan, acceptance tests, and
optional surface/log trail live as **stem-qualified siblings** beside it;
tasks live only under a paired `slug/` directory; post-delivery change
requests and bug fixes live under `changes/`; the recurring mechanisms every
feature must follow the same way live as **practices**, and what the project
has said but not yet done everywhere is written down as **debt**; an
opinionated CLI validates the whole bundle so it can never silently drift.

```
docs/features/
├── INDEX.md                      # bundle root (pins fdf_version)
├── LOG.md
├── SPEC.md                       # the format spec, shipped in the bundle
├── STACK.md                      # Context: technology stack
├── ARCHITECTURE.md               # Context: architecture & principles
├── SURFACES.md                   # Context: interface principles (all surfaces)
├── INFRA.md                      # Context: build & deployment infra
├── DOMAIN.md                     # Context: the project's domain language
├── changes/                      # post-delivery work (flat or grouped)
│   ├── INDEX.md
│   └── refund-rounding.md        # type: Fix — the code drifted from the doc
├── debts/                        # known gaps between doc and code
│   ├── INDEX.md
│   └── authz-legacy-handlers.md  # type: Debt — open | accepted | resolved
├── practices/                    # how recurring mechanisms are done
│   ├── INDEX.md
│   └── permission-checks.md      # type: Practice — binding on all code
└── payments/
    ├── INDEX.md
    ├── instant-refunds.md        # Feature: Gherkin scenarios + status
    ├── instant-refunds.spec.md   # approved design (type: Spec)
    ├── instant-refunds.plan.md   # links every task (type: Plan)
    ├── instant-refunds.test.md   # scenario -> concrete proof (type: Test)
    ├── instant-refunds.surface.md  # optional (type: Surface)
    ├── instant-refunds.log.md    # optional feature decisions
    └── instant-refunds/          # task directory ONLY
        ├── 01-refund-api.md
        └── 02-refund-ui.md
```

**Documents are living or episodic**, and that decides what happens when the
software changes. Living documents describe the system *today* — the feature's
Gherkin, `slug.test.md`, `slug.surface.md`, practices, debts, the Context docs — and are amended
in place. Episodic documents are frozen records of one piece of work —
`slug.spec.md`, `slug.plan.md`, tasks, changes, log entries — and are never
rewritten. So a delivered feature that changes keeps **one** feature document,
whose Gherkin is edited, and gains a **new** episode under `changes/`. Its
original spec and plan stay untouched: they record how it was first built,
which is the context you need to judge the change.

A `Change` alters what a delivered feature does; a `Fix` restores behavior the
feature document already describes. Both name the features they touch in
`affects:` — so one bug spanning several features, or one request spanning
them, is a single document. Both declare the effects they will have, and rule
**F10** refuses to let one reach `done` while the features it claims to alter
still describe the old behavior. That is the drift FDF exists to prevent,
enforced rather than hoped for.

The five **Context documents** (`STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md`, `DOMAIN.md`) are the project's living stack /
architecture / surface-principles / infrastructure / vocabulary snapshot.
**SURFACES.md is always defined** at the bundle root (interface principles for
APIs, UIs, CLIs, events, and inputs — not “UI only”); feature-level
`slug.surface.md` is optional when a feature needs extra surface detail.
`fdf init` scaffolds all five as stubs; the `fdf-init` skill interview fills
them. They are critical and change only with explicit human approval —
accurate context is what makes this agentic engineering, not vibe coding.

**DOMAIN.md** is the project's domain language: one canonical name per concept
and the words banned in its place. The same thing called `Item` in one
document, `Product` in another and `SKU` in a third is three things to every
reader, and that drift reaches schemas and endpoints. It is the project's
*internal* vocabulary — the bundle and the code's identifiers — not the words
a surface shows to people: a UI label or locale string may say "store" for a
Venue on purpose, and the skills treat that as a surface decision, never as
drift. Rule **F12** reports a banned word found in a feature's Gherkin — a
warning by default, an error under `fdf validate --strict-domain`.

**Practices** (`practices/<slug>.md`, `type: Practice`) are the project's
binding answers to *how do we do X* for one recurring mechanism —
authorization, permission checks, payment capture, database access. They exist
so the answer is written once and followed, instead of being re-derived
differently in every feature spec, and so `ARCHITECTURE.md` does not have to
swallow every mechanism to keep them written down. A practice is living and
has no episodic trail; its `applies-to` paths are how later work finds it,
since a feature never lists the practices it follows. Rule **F11** keeps them
honest: a non-empty `# Rules`, no Gherkin, and a `superseded` practice that
names its replacement.

**Debt** (`debts/<slug>.md`, `type: Debt`) is the register of known gaps
between what the project says and what the code does: work deliberately left
undone, and rules the codebase does not follow everywhere yet. That second
case is what lets a practice be written honestly for an existing codebase —
the practice states the rule, the debt names the files that have not caught up,
and neither document has to lie for the other to exist. A debt is a register
entry, not a unit of work: the work that closes it is a `Change`, a `Fix`, or
plain code work. Rule **F13** means it cannot be closed without saying what
closed it (`# Resolution`), cannot be kept without saying why (`# Rationale`
on `accepted`), and — through R1 on its `resource` paths — cannot go on naming
code that no longer exists.

## Why

Agents made writing code cheap. What broke is the record of *why* the code
exists: the design lived in a chat scrollback that is gone, the plan was
never written down, and the tests prove whatever the code happens to do.
That is vibe coding, and it compounds — every future change starts from
archaeology.

FDF is the opposite bet: the documentation is the interface between humans
and agents. Intent, design, plan, and proof live in the repo beside the
code; a feature's status must say what is actually true of it; and
`fdf validate` turns drift into a failing build instead of a discovery six
months later. Humans approve specs and context; agents do the mechanical
work in between. Built for **agentic engineering, not vibe coding**.

## Install

- One-liner (macOS/Linux, no Go needed):
```bash
curl -fsSL https://raw.githubusercontent.com/GiteshDalal/fdf/main/install.sh | bash
```
- Binaries on the [releases page](https://github.com/GiteshDalal/fdf/releases)
- With [mise](https://mise.jdx.dev)
```
mise use ubi:GiteshDalal/fdf
```
- With [Go](https://go.dev/)
```
go install github.com/GiteshDalal/fdf/cli/cmd/fdf@latest
```

## Use

```bash
fdf init                     # scaffold docs/features/ + SPEC.md + context stubs (or FDF_ROOT_DIR / --root)
                             #   then run the fdf-init skill to fill STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN
fdf new payments/instant-refunds
fdf validate                 # F1-F13 + R1; exit 1 on any violation
fdf validate --strict-domain # …and F12 banned words become errors, not warnings
fdf practice permission-checks  # scaffold a Practice under practices/

fdf debt                     # the register: status, id, filing date, title
fdf debt --open              # …only what is outstanding (also --accepted, --resolved)
fdf debt authz-legacy-handlers  # file a new debt
fdf debt --cleanup           # fold resolved debts into debts/LOG.md and clear them
fdf debt --cleanup --dry-run # …show the plan first, change nothing
fdf spec                     # print the format spec (-v 0.4 for an older one)
fdf help                     # every command with examples
fdf serve                    # browse the bundle (bun x mdts)

# after a feature ships
fdf change --affects payments/instant-refunds refund-window   # behavior should differ
fdf fix    --affects payments/instant-refunds refund-rounding # code drifted from the doc
fdf history payments/instant-refunds                          # what happened since it shipped
fdf release 1.2.0            # derive the release doc from `version:` fields; --ship to close it
fdf install claude-code      # user-level skills + "## Feature Document Format" primer
fdf install codex            #   (primer skipped if the heading is already present)
fdf install opencode
fdf install --project claude-code   # project-level: skills under .claude/, primer in ./CLAUDE.md
fdf migrate                  # mechanical upgrade to the current spec version
```

`fdf install` defaults to **user-level** (under your home directory). Prefer
`fdf install --project <harness>` for repos that actually carry an FDF bundle:
skills and the primer stay out of unrelated projects, and a stale primer after
`fdf migrate` is a per-repo refresh instead of a machine-global one. User-level
and project-level installs coexist and upgrade independently (each destination
has its own `.fdf-version` markers). A machine with both will carry two primers
— harnesses merge memory files, so that is fine.

| Scope | Harness | Skills | Instruction file |
|---|---|---|---|
| user | claude-code | `~/.claude/skills/` | `~/.claude/CLAUDE.md` |
| user | codex | `~/.codex/skills/` | `~/.codex/AGENTS.md` |
| user | opencode | `~/.config/opencode/skills/` | `~/.config/opencode/AGENTS.md` |
| project | claude-code | `<proj>/.claude/skills/` | `<proj>/CLAUDE.md` |
| project | codex | `<proj>/.codex/skills/` | `<proj>/AGENTS.md` |
| project | opencode | `<proj>/.opencode/skills/` | `<proj>/AGENTS.md` |

The agent-facing surface is **skills only** — [nine of them](#skills),
identical across harnesses. Earlier versions also shipped Claude Code slash
commands; those wrapped skills the model can reach directly, so `fdf install`
now removes them (leaving any commands you wrote yourself alone).

`--project` outside a git working tree is a usage error (exit 2).
`fdf install --root <dir>` (or `FDF_ROOT_DIR`) bakes a non-default bundle
location into the installed skills, which otherwise reference
`docs/features/`; it composes with `--project` unchanged. Agents need no prior
FDF knowledge: the spec copy at the bundle root is the reference the skills
and primer point them to.

Works the same everywhere: the bundle may be a plain directory or a git
submodule mounted at the same path — `resource:` paths always verify against
the **project** root.

### Upgrading to v0.6

1. Run `fdf migrate` on the bundle. From **v0.5** nothing moves: the pin
   moves, `SPEC.md` is re-vendored, and `practices/INDEX.md`, `debts/INDEX.md`
   and a `DOMAIN.md` stub appear. One thing does change, and it is the point of the version —
   `DOMAIN.md` is a fifth Context document, so **F9 holds the bundle to filling
   it before the next feature**. `fdf migrate` itself never fails on that
   (fresh stubs are advisory); the next plain `fdf validate` does, until the
   `fdf-init` interview fills the lexicon. From **v0.4** it also scaffolds
   `changes/INDEX.md`; from **v0.2/v0.3** it additionally lifts nested trail
   files to stem-qualified siblings and scaffolds `SURFACES.md`.
2. **Re-run `fdf install`** for each harness you use. Skills and primers
   written under an older version do not know about `practices/`, `debts/`,
   the `Practice` and `Debt` types, or `DOMAIN.md`, and stay stale in
   CLAUDE.md/AGENTS.md
   until refreshed. `fdf install` upgrades skills automatically when the
   version marker changes; if your primer heading already exists with old
   wording, edit or replace that section.

## Skills

`fdf install` places nine skills into your harness. Five walk a feature
through its lifecycle; the other four are not stages at all — a router, a
diagnostic front door, a periodic audit, and a gate, each of which applies at
every stage. An agent needs no prior FDF knowledge: `fdf-help` routes, and the
spec is vendored in the bundle it is working in.

| Skill | What it is for |
|---|---|
| **`fdf-help`** | The router. Finds the feature and its status, then sends the work to the right skill — *before* any code is written, including for "tiny" changes. |
| **`fdf-init`** | The project-context interview. Fills `STACK` / `ARCHITECTURE` / `SURFACES` / `INFRA` / `DOMAIN` with your approval, so every later feature is designed against what is actually true. |
| **`fdf-brainstorm`** | Idea → feature document. Questions the idea one at a time, writes the Gherkin, and gets your explicit design approval before writing `slug.spec.md`. |
| **`fdf-plan`** | Approved spec → executable plan. Tasks a zero-context implementer could run, plus `slug.test.md`: how each scenario will be *proven*. Ends with a ready-to-paste prompt, so execution can start from a fresh context. |
| **`fdf-execute`** | Plan → working code. Delegates tasks to subagents in parallel batches — a fast model for mechanical tasks, the most capable one for tasks that need judgment — or works them serially; keeps statuses truthful, and will not call a feature done on an unrun test case. |
| **`fdf-change`** | Post-delivery writing. A `Change` when a delivered feature must behave differently; a `Fix` when the code drifted from what its document already says. |
| **`fdf-debug`** | Something is broken and nobody knows why yet. Finds the root cause first, then routes the repair — Fix, Change, new feature, or task work. |
| **`fdf-checkpoint`** | The periodic audit. Checks the Context docs, the vendored `SPEC.md`, and `CLAUDE.md` / `AGENTS.md` against the code and against each other — stale, repeated, or contradictory — and proposes each fix for your approval. Run it before a release and after dependency or infrastructure work. |
| **`fdf-validate`** | The gate after every bundle edit. Turns `fdf validate`'s rule codes into the *honest* fix, and refuses the fake ones (deleting a scenario so F8 stops asking). |

The three human gates are: the Context interview (`fdf-init`), the design
approval (`fdf-brainstorm`, and `fdf-change` for a `Change`), and any edit to
a Context document. Everything between them is mechanical work an agent does
on its own.

### Common flows

**Build a new capability**

`fdf-help` → `fdf-brainstorm` → `fdf-plan` → `fdf-execute`

The spine. Brainstorm stops at "Do you approve this design?" and writes
nothing before you say yes; plan turns the approved spec into tasks and the
test document; execute works the tasks, flipping statuses as it goes, and
reports the actual command output per test case. It ends by asking whether
the work made any Context doc stale.

**Fix a bug in a delivered feature**

`fdf-debug` → *(root cause)* → `fdf-change` → `fdf fix` or `fdf change`

Debug first, because the root cause decides the document. Code contradicts a
scenario that already exists → a **`Fix`** (no design gate; the lasting
artifact is the regression case added to the feature's `slug.test.md`). No
scenario covers the case, or the scenario itself is wrong → a **`Change`**,
because someone now has to decide what should happen, and that needs the
design gate.

**Fix a bug in a feature still being built**

`fdf-debug` → `fdf-execute`

A feature that has not shipped is repaired in place as task work. `Change`
and `Fix` are only for `done` / `retired` features — F10 rejects anything
else.

**Change what a delivered feature does**

`fdf-help` → `fdf-change`

A request, not a defect, so there is nothing to diagnose. The change declares
its effects up front (`add:` / `modify:` / `remove:` per scenario), and F10
will not let it reach `done` until the feature's Gherkin actually says so.

**Retire a capability**

`fdf-change` → a `Change` with `retires:` and a `# Rationale`

The feature document is never deleted — it records behavior the software once
had. The feature goes to `retired`, with `replaced-by:` when a successor
exists.

**Update ARCHITECTURE.md / INFRA.md after work that changed reality**

`fdf-execute` / `fdf-change` → Context-doc review → your approval → LOG entry

Not a separate flow you start — it is the last step of the work that made the
doc stale. The agent *proposes* the specific edit; nothing is written until
you approve it, and the change is logged. A Context doc can also go stale with
no feature in flight — a dependency upgrade, a CI move, a refactor. That is
what `fdf-checkpoint` is for: it audits the Context docs, `SPEC.md` and
`CLAUDE.md` / `AGENTS.md` against the code and each other, and proposes each
correction the same way.

**Migrate infrastructure** (new datastore, new deployment target)

First question: **does anything a user or a calling system can observe
change?** — limits, error shapes, endpoints, auth, guarantees.

- **Yes** → every feature that promises that behavior gets a `Change`
  (`fdf-change`), or a new feature via `fdf-brainstorm` if the migration adds
  a capability. Then the `INFRA.md` / `STACK.md` update, approved and logged.
- **No** → the work is behavior-neutral, so no feature document changes. The
  only bundle artifact is the approved `INFRA.md` / `STACK.md` edit plus its
  LOG entry — which `fdf-checkpoint` proposes if nobody did at the time.

FDF has no document type for behavior-neutral engineering work, and
deliberately so: the bundle records what the software *does*, not every task
performed on it. Track the migration itself in your issue tracker.

**Start a project, or adopt FDF in an existing one**

`fdf init` → `fdf-init` → `fdf-brainstorm`

The interview is the most leveraged conversation in the whole workflow, and
F9 blocks feature work until it is done — accurate context is what makes an
agent build *this* project's way instead of guessing.

## Spec

FDF is defined by versioned specs under [`spec/`](spec/) — current
[v0.6](spec/0.6.md), prior [v0.5](spec/0.5.md) / [v0.4](spec/0.4.md) / [v0.3](spec/0.3.md) /
[v0.2](spec/0.2.md);
[SPEC.md](SPEC.md) indexes them. Each is normative for the bundles pinning
its version, and every bundle vendors its pinned version's spec at
`docs/features/SPEC.md`. `testdata/` fixtures are the executable conformance
contract. MIT licensed.
