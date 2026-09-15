---
name: fdf-help
description: Use when starting any conversation in a project with an FDF bundle (docs/features/ or FDF_ROOT_DIR) — establishes how to route work to the fdf skills by feature status BEFORE writing any code, including "quick", "tiny", and "just do it" changes.
---

# Using FDF

## What is FDF

FDF (Feature Document Format) documents software features as a directory of
markdown files — the **bundle**, at `docs/features/` in this project. Each
feature is one Markdown + Gherkin file with a lifecycle `status` in its YAML
frontmatter. Its implementation trail lives as **stem-qualified siblings**
beside it at the group level:

| Role | Path | Required from |
|---|---|---|
| Spec | `<group>/<slug>.spec.md` | `specified` |
| Plan | `<group>/<slug>.plan.md` | `planned` |
| Test | `<group>/<slug>.test.md` | `planned` |
| Surface | `<group>/<slug>.surface.md` | optional (`type: Surface`) |
| Log | `<group>/<slug>.log.md` | optional |
| Tasks | `<group>/<slug>/NN-….md` | task directory only |

Position and stem are the link — no frontmatter pointers. The task directory
holds **only** ordered task files; trail documents never nest inside it.

Work that arrives **after** a feature ships lives under `changes/` as a
`Change` (alters what the feature does) or a `Fix` (the code drifted from what
the feature document already says). Both may be filed flat or in groups, both
take the same `slug.spec.md`/`slug.plan.md` trail and task directory, and both
name the features they touch in an `affects:` list.

Four bundle-root **Context documents** — `STACK.md`, `ARCHITECTURE.md`,
`SURFACES.md`, `INFRA.md` — hold the project's current stack, architecture,
interface principles for **all surfaces** (API/UI/CLI/events/inputs — not UI
only), and build/deployment infrastructure.

You are not expected to know FDF. Two commands tell you everything:

| Question | Command |
|---|---|
| What are the exact format rules? | `fdf spec` (or read `docs/features/SPEC.md`) |
| What can the CLI do, with examples? | `fdf help` (or `fdf help <command>`) |

The CLI keeps the bundle honest: `fdf validate` must exit 0 after any bundle
edit. Scaffold with `fdf new <group>/<slug>`, `fdf change`, `fdf fix`. The
rules the fdf skills cite by number (F1–F10, R1) are defined in the spec.

The bundle is the source of truth for what the software does. Code that
changes behavior without touching the bundle makes the bundle lie — that is
the failure FDF exists to prevent, and tiny changes are where it happens.

**Context documents are critical.** STACK/ARCHITECTURE/SURFACES/INFRA are the
project's living context; accurate, they let you build the project's way
instead of guessing — agentic engineering, not vibe coding. They are filled
once by the fdf-init interview and changed only with explicit user approval
(the post-feature step in fdf-execute), each change logged. Never edit them
casually. Until filled they carry a `<!-- fdf:stub -->` stub marker, and F9
blocks feature work while any stays a stub.

## Living and episodic documents

This is the distinction that decides what you touch when something changes:

| | Documents | When the software changes |
|---|---|---|
| **Living** — the system **today** | Feature Gherkin, `slug.test.md`, `slug.surface.md`, Context docs | **Amend in place.** A living document describing behavior the software no longer has makes the bundle lie. |
| **Episodic** — a record of **one piece of work** | `slug.spec.md`, `slug.plan.md`, tasks, Change/Fix, log entries | **Never rewrite.** New work gets a new episode; the old one records why things were done that way. |

So a delivered feature that changes keeps **one** feature document, whose
Gherkin you edit, and gains a **new** Change document with its own spec, plan
and tasks. Never fork a second feature document for the same capability: two
documents describing one capability is exactly the drift FDF exists to stop.

## The Rule

Before writing any code — and before answering "how should we build X" —
find the bundle state and route by **feature status**. Status is the dispatch
key: not the verb the user used, not the size of the change.

| Bundle state | Skill |
|---|---|
| Context docs missing or still `<!-- fdf:stub -->` | fdf-init (fill them first) |
| No feature file for this capability | fdf-brainstorm |
| `draft` | fdf-brainstorm (finish the spec) |
| `specified` | fdf-plan |
| `planned` or `implementing` | fdf-execute |
| `done` or `retired` | Delivered — post-delivery work → fdf-change |
| *(any bundle file just edited, or `fdf validate` failing)* | fdf-validate |

Check the Context docs first: if `fdf validate` warns or fails on unfilled
STACK/ARCHITECTURE/SURFACES/INFRA stubs, route to fdf-init before any feature
work — F9 will block it otherwise.

fdf-validate is not a stage — it is the gate that closes every one of them.
Use it after any edit to a bundle file, including edits made outside a
workflow skill (a status flip, a ticked checkbox, a typo fix), and whenever
`fdf validate` exits non-zero.

Then announce: "Using fdf-<skill> — <feature> is <status>."

For a **delivered** feature, one question routes the work:

> **Does this require the feature's Gherkin to change?**

- **No** — the document was right, the code drifted. That is a `Fix`
  (`fdf fix`). No design gate; the floor is a single file. Its lasting
  artifact is the regression case added to the feature's `slug.test.md`.
- **Yes** — someone is deciding what the software should do, *including* when
  the original document was silent or wrong on a case nobody recognized. That
  is a `Change` (`fdf change`), and it needs the design gate.

A Change or Fix may name several features in `affects:` — one bug can surface
across features, and one request can span them.

A request may span stages: "get it implemented" on a `specified` feature
means fdf-plan **then** fdf-execute. Chain them in order; never enter a later
stage because the user named it.

**Does this change need a bundle document?** Test: does it change what the
software does for a user, or make the software match what the bundle already
promises? Either way, yes:

- New capability → a feature (fdf-brainstorm).
- Altering a delivered capability → a `Change` (fdf-change).
- Making code match a delivered feature's documented behavior → a `Fix`
  (fdf-change). This is **not** an exemption: the regression case is what
  stops the bug coming back.

Only genuinely bundle-neutral work is exempt — pure refactors with no
behavior change, typos, tooling, dependency bumps.

**When is it a new feature instead of a Change?** If the new behavior reads as
its own `Feature:` block with its own As-a / I-want / So-that, it is a new
feature, and it may record its lineage with `depends-on`. If it alters an
existing capability's observable behavior, it is a Change.

## "It's tiny, just do it"

Size never routes around FDF. The minimal compliant path is cheap: `fdf new
<group>/<slug>`, one `Feature:` fence, one `Scenario:` — minutes, and the
skills scale down to match. State that cost once. If the user still
explicitly opts out, their instruction wins — do the work, then say plainly
that the bundle now lacks this change. The violation is the *silent* skip,
and so is the shortcut of doing it first and asking later.

## Red Flags — STOP, you are rationalizing

| Thought | Reality |
|---|---|
| "It's a one-line change" | Size doesn't route; status does. One line that changes behavior gets a document. |
| "It's just a bug, no doc needed" | A bug fix is a `Fix` under `changes/`. Ten lines of markdown, and the regression case is what stops it recurring. |
| "This feature is done, I'll make a v2 feature doc" | Two documents for one capability is the drift FDF exists to stop. Amend the feature; record the work as a Change. |
| "The feature is done, I'll just edit its Gherkin" | Then nothing records why it changed, and F10 never checked that the code followed. Open a Change. |
| "The user said just do it" | Offer the two-minute path first. Only an explicit opt-out after that counts. |
| "I'll backfill the docs later" | The bundle lies the whole interim. Draft first, code second. |
| "The full pipeline is process theater" | The trail is what the next agent trusts. Scale it down, don't skip it. |
| "User said 'implement', so fdf-execute" | Verb ≠ status. Read the frontmatter, route by it. |
| "I'll flip statuses in a batch at the end" | Statuses reflect reality *now* — `in-progress` before working, per task. |
| "Skip validate just this once" | `fdf validate` exit 0 is the gate after every bundle edit. No exceptions — use fdf-validate. |
| "Validate failed, I'll just delete the scenario" | Never weaken content to silence a rule. fdf-validate has the honest fix for each code. |
| "I'll add the back-link on the feature" | Don't. `affects:` is the whole link; `fdf history <feature>` computes the rest. A hand-written back-link drifts. |

## Precedence

Explicit user instructions outrank skills. But "quick", "we're late", and
"don't waste tokens" are pressure, not opt-outs — the only opt-out is the
user declining the minimal path after you've named it.
