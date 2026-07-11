# Feature Document Format (FDF) — specification

FDF is defined by **versioned specification documents** under [`spec/`](spec/).
Each is normative for the bundles that pin its version; a bundle vendors a copy
of its pinned version's spec at its own `docs/features/SPEC.md`, so a bundle is
always self-describing.

| Version | Document | Status |
|---------|----------|--------|
| **0.4** | [spec/0.4.md](spec/0.4.md) | **Current** |
| 0.3     | [spec/0.3.md](spec/0.3.md) | Superseded (validated for back-compat; migrate with `fdf migrate`) |
| 0.2     | [spec/0.2.md](spec/0.2.md) | Superseded (validated for back-compat; migrate with `fdf migrate`) |

The `fdf` CLI validates supported versions and offers mechanical migration
between adjacent ones. `testdata/` fixtures are the executable conformance
contract. MIT licensed.
