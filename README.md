# FDF — Feature Document Format

**Documentation-as-a-directory for software features.** Each feature is a
Markdown + Gherkin document; its design spec, plan, acceptance tests, and
tasks live in a paired directory beside it; an opinionated CLI validates the
whole bundle so it can never silently drift.

```
docs/features/
├── INDEX.md                      # bundle root (pins fdf_version)
├── LOG.md
└── payments/
    ├── INDEX.md
    ├── instant-refunds.md        # Feature: Gherkin scenarios + status
    └── instant-refunds/
        ├── SPEC.md               # approved design
        ├── PLAN.md               # links every task
        ├── TEST.md               # scenario -> concrete proof
        ├── 01-refund-api.md      # Task (status, depends-on, resource)
        └── 02-refund-ui.md
```

## Install

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
fdf init                     # scaffold docs/features/ (or FDF_ROOT_DIR / --root)
fdf new payments/instant-refunds
fdf validate                 # F1-F8 + R1; exit 1 on any violation
fdf serve                    # browse the bundle (bun x mdts)
fdf install claude-code      # agent skills: brainstorm -> plan -> execute
fdf migrate                  # mechanical upgrade between spec versions
```

Works the same everywhere: the bundle may be a plain directory or a git
submodule mounted at the same path — `resource:` paths always verify against
the **project** root.

## Spec

[SPEC.md](SPEC.md) is normative; `testdata/` fixtures are the executable
conformance contract. MIT licensed.
