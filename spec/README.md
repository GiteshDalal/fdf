# FDF specification versions

One file per spec version; each is the normative text for bundles pinning that
version. The current version is the highest-numbered file.

- [`0.3.md`](0.3.md) — current. Adds `Context` documents (STACK/ARCHITECTURE/
  INFRA), feature-directory `LOG.md`, and rule F9.
- [`0.2.md`](0.2.md) — superseded. Still validated for back-compat; upgrade a
  v0.2 bundle with `fdf migrate`.

`fdf init` and `fdf migrate` vendor the pinned version's spec into the bundle
at `docs/features/SPEC.md`, so old-version specs also live inside the bundles
that pin them. Do not edit a released version's file; add a new one instead.
