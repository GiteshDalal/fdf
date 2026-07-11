---
name: fdf-init
description: Use right after `fdf init` on a new FDF bundle, or whenever STACK.md / ARCHITECTURE.md / INFRA.md are still unfilled stubs — runs the project-context interview that fills the three critical Context documents before any feature work.
---

# FDF Init

Interview the user about the project, then write the three bundle-root
**Context documents** — `STACK.md`, `ARCHITECTURE.md`, `INFRA.md` — that every
later feature relies on. This is the difference between agentic engineering
and vibe coding: with accurate context an agent builds *this* project's way;
without it, it guesses. Do this thoroughly and slowly. It is the most
leveraged conversation in the whole workflow.

New to FDF? The format is defined in the bundle at `docs/features/SPEC.md`.
The three Context docs live at the bundle root beside it; `fdf init` writes
them as stubs carrying `<!-- fdf:stub -->` and a ⚠️ banner. Your job is to
replace each stub with real content. Validation rule F9 rejects any bundle
that has features while a Context doc is still a stub — so this interview
comes first.

## The three documents

| File | Captures |
|---|---|
| `STACK.md` | Languages, runtimes, frameworks, libraries, data stores — and their versions. What the code is written in and with. |
| `ARCHITECTURE.md` | The architecture style, how code is organized, the design principles and conventions contributors follow, and the load-bearing decisions behind them. |
| `INFRA.md` | How the project is built, tested, packaged, deployed; its environments and runtime targets; the operational dependencies (CI, hosting, queues, caches, secrets). |

Each is a **current snapshot**, not a roadmap and not a changelog. Write only
what is true now (or true for the project being started).

## Process

1. **Locate the bundle and read the stubs.** Run `fdf validate` (respects
   `--root`/`FDF_ROOT_DIR`). Read the three stub files so you match their
   heading structure. If they don't exist yet, run `fdf init` first.
2. **Survey what already exists.** If there's code, read enough to ground your
   questions — `README`, manifests (`package.json`, `go.mod`, `Cargo.toml`,
   `pyproject.toml`, `pom.xml`), lockfiles, CI config, Dockerfiles, IaC. Come
   to the interview with informed guesses to confirm, not a blank slate. For a
   greenfield project there's nothing to read — the interview *is* the design.
3. **Interview — one question at a time.** Prefer multiple-choice with a
   recommended default; let the user redirect. Do not dump a questionnaire.
   Cover the areas below, skipping what genuinely doesn't apply (and say why
   you're skipping). Chase vague answers ("scalable", "modern", "cloud") into
   specifics. Ask follow-ups when an answer implies more (chose microservices
   → how do services communicate? chose a SPA → what backend serves it?).
4. **Draft each document and get approval section by section.** Present a
   draft, take edits, confirm. These become immutable-without-approval the
   moment they're written — so the user must genuinely agree now.
5. **Write the three files.** Replace the entire stub (remove the
   `<!-- fdf:stub -->` sentinel and the ⚠️ banner — their presence is what
   F9 treats as "unfilled"). Keep `type: Context` and the frontmatter; update
   `timestamp`.
6. **Log and gate.** Add a LOG.md entry noting the interview and key decisions.
   Run `fdf validate` — exit 0 (F9 now satisfied) before you're done.
7. **Hand off the responsibility.** Tell the user plainly (see Closing).

## What to ask

**Purpose & scope**
- What is this project, in one sentence? Who uses it and to do what?
- What are the primary use cases? What is explicitly out of scope?
- Greenfield or existing? Any hard constraints (compliance, offline, latency,
  budget, team size/skills)?

**Stack → STACK.md**
- Primary language(s) and runtime/version. Why these?
- Frameworks and major libraries (web, ORM, UI, testing, etc.).
- Data stores: relational / document / key-value / cache / search / blob —
  which engines, and what each holds.
- External services and APIs the project depends on.
- Version floors that matter (language edition, framework major).

**Architecture → ARCHITECTURE.md**
- Shape: modular monolith? microservices? serverless / cloud functions?
  library/SDK? CLI? A frontend, a backend, or fullstack? A mix?
- If multiple services/modules: how do they communicate (HTTP/gRPC/queue/
  events)? Where are the boundaries?
- Code organization: by layer, by feature/domain, by service? Monorepo or
  multi-repo? Where does new code go?
- Design principles and conventions to follow (e.g. DDD, hexagonal, DI,
  functional core, error-handling style, API design rules).
- State & data flow: where does state live, how does data move through the
  system?
- The few decisions a new contributor must not silently violate.

**Infrastructure → INFRA.md**
- Build & test: how is it built and tested? What commands? What CI?
- Packaging & artifacts: binaries, containers, packages, bundles?
- Environments: local, staging, production — how do they differ?
- Deployment: where does it run (cloud provider, k8s, serverless, on-prem,
  app stores, package registries)? How does a change reach production?
- Operational dependencies: managed DBs, caches, queues, CDNs, secrets,
  observability. What has to exist for the system to run?
- Targets: OS/arch, browsers, mobile platforms, runtime versions.

## Closing (say this explicitly)

Tell the user, in your own words:

> STACK.md, ARCHITECTURE.md, and INFRA.md are written and validated. Treat
> them as **critical, living documents**: from here on I will not change them
> without your explicit approval, and after each feature I'll ask whether any
> of them needs updating (a new dependency, a new pattern, new infrastructure)
> and only edit on your say-so, logging the change. Keeping them accurate is
> what makes this agentic engineering rather than vibe coding — stale context
> produces confidently wrong work. They're yours to own; I'll help maintain
> them.

## Rules

- One question at a time; confirm each document before writing it.
- Write only what's true now — snapshots, not aspirations or history.
- Removing the stub sentinel/banner is required; a Context doc that still
  contains `<!-- fdf:stub -->` counts as unfilled and fails F9 once features
  exist.
- These three files are edited ONLY here and, later, via the post-feature
  update step in fdf-execute — always with explicit user approval, always
  logged. Never edit them casually mid-implementation.
