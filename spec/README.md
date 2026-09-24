# FDF specification versions

One file per spec version; each is the normative text for bundles pinning that
version. The current version is the highest-numbered file.

- [`0.7.md`](0.7.md) — current. Adds `Bug` documents under `bugs/` (a register
  of known defects that have not been repaired yet) with rule F14, the
  `adopted` feature status (a capability documented from code that predates
  the bundle, with no invented build trail), and *maintenance edits* — the
  lexicon fix, reference repair after a move, and path repair — which are
  allowed on every document. F12 now reaches every document except
  `SPEC.md`, `DOMAIN.md`, `slug.test.md` and `slug.surface.md`, and every
  document name, with `except:` phrases and `strict: true` in `DOMAIN.md`.
  Dates and times are UTC. Additive over 0.6 except that F12 reports more,
  `bugs/` becomes a reserved name, a test case must be a `## <scenario name>`
  heading matched exactly (F8), a `timestamp` or log heading that is not a
  real date or time is an error (F1), and the declarations are checked more
  closely (F10, F14); the file's *Versioning* section lists every
  difference.
- [`0.6.md`](0.6.md) — superseded. Still validated for back-compat. Adds
  `DOMAIN.md` (a fifth Context document
  holding the project's domain language), `Practice` documents under
  `practices/`, `Debt` documents under `debts/`, and rules F11 (practice
  integrity), F12 (domain integrity) and F13 (debt integrity). Additive over
  0.5 except that F9 now requires `DOMAIN.md`. Erratum 2026-09-23: a lexicon
  fix (a banned word replaced by its term) is allowed in every document,
  frozen ones included, and F12 scans declared scenario names rather than
  whole declaration sections (see the file's `# Errata`).
- [`0.5.md`](0.5.md) — superseded. Still validated for back-compat. Adds
  post-delivery documents: `Change` and `Fix` under `changes/` (flat or
  grouped), rule F10 (change integrity), the `retired` feature status,
  feature `depends-on`, and releases that may list changes.
- [`0.4.md`](0.4.md) — superseded. Still validated for back-compat.
  Stem-qualified trail layout
  (`slug.spec.md` / `slug.plan.md` / `slug.test.md` beside the feature; tasks
  only under `slug/`), mandatory root `SURFACES.md` Context document, optional
  `slug.surface.md` (`type: Surface`) and `slug.log.md` (`type: Log`).
- [`0.3.md`](0.3.md) — superseded. Still validated for back-compat; adds
  `Context` documents (STACK/ARCHITECTURE/INFRA), feature-directory `LOG.md`,
  and rule F9. Upgrade a v0.3 bundle with `fdf migrate`.
- [`0.2.md`](0.2.md) — superseded. Still validated for back-compat; upgrade a
  v0.2 bundle with `fdf migrate`.

`fdf init` and `fdf migrate` vendor the pinned version's spec into the bundle
at `docs/features/SPEC.md`, so old-version specs also live inside the bundles
that pin them. Do not edit a released version's file; add a new one instead.
The one exception is an **erratum**: a correction that resolves a
contradiction in the text by relaxing a rule, so that no bundle which
conformed stops conforming. It is recorded, dated, in that file's `# Errata`
section. Anything that could fail a bundle that passed before is a new
version.
