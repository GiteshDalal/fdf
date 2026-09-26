# Feature Document Format (FDF) — specification

FDF is defined by **versioned specification documents** under [`spec/`](spec/).
Each is normative for the bundles that pin its version; a bundle vendors a copy
of its pinned version's spec at its own `docs/fdf/SPEC.md`, so a bundle is
always self-describing.

| Version | Document | Status |
|---------|----------|--------|
| **1.0** | [spec/1.0.md](spec/1.0.md) | **Current** |
| 0.7     | [spec/0.7.md](spec/0.7.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |
| 0.6     | [spec/0.6.md](spec/0.6.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |
| 0.5     | [spec/0.5.md](spec/0.5.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |
| 0.4     | [spec/0.4.md](spec/0.4.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |
| 0.3     | [spec/0.3.md](spec/0.3.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |
| 0.2     | [spec/0.2.md](spec/0.2.md) | Not validated by fdf 1.x; upgrade with `fdf migrate` |

The `fdf` CLI validates the versions of its own major — fdf 1.x validates spec
1.x — and `fdf migrate` upgrades a bundle at any 0.x version to 1.0 in one run.
`testdata/` fixtures are the executable conformance contract. MIT licensed.
