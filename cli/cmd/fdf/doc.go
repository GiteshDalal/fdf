// Fdf is the command-line tool for FDF (Feature Document Format), which keeps
// a project's features as validated Markdown and Gherkin documents in the
// repository, beside the code they describe.
//
// Each feature is one document under features/: its Gherkin scenarios, and a
// status that must be true of it — draft, specified, planned, implementing or
// done, adopted for code that predates the bundle, retired once it is gone.
// Its design spec, implementation plan and acceptance tests sit beside it,
// and its tasks in a directory of the same name. Five Context documents at the
// bundle root hold the project's stack, architecture, surfaces,
// infrastructure and domain language, so an AI agent reads what is true of
// the project instead of guessing it from the code. fdf scaffolds the bundle,
// validates it, keeps its names and references consistent, and installs the
// skills that teach an agent the workflow.
//
// The [project page] explains why the format exists and how it compares with
// other approaches; this page is the command's reference.
//
// # Install
//
// On macOS or Linux, the install script downloads the latest release,
// verifies its checksum and puts fdf in ~/.local/bin:
//
//	curl -fsSL https://giteshdalal.com/fdf.sh | bash
//
// FDF_VERSION=vX.Y.Z pins a release, and FDF_INSTALL_DIR chooses another
// directory:
//
//	curl -fsSL https://giteshdalal.com/fdf.sh | FDF_INSTALL_DIR="$HOME/bin" bash
//
// With Go 1.26 or later:
//
//	go install github.com/GiteshDalal/fdf/cli/cmd/fdf@latest
//
// Binaries for macOS, Linux and Windows are on the [releases] page, and
// [mise] pins a version per project: mise use ubi:GiteshDalal/fdf.
//
// # Quick start
//
// In the repository of the project you want to document:
//
//	fdf init                           # docs/fdf/: INDEX.md, SPEC.md, the Context stubs, the registers
//	fdf install --project claude-code  # the skills and primer for your agent (or codex, opencode)
//	fdf new payments/instant-refunds   # a draft feature: features/payments/instant-refunds.md
//	fdf validate                       # check the bundle; exit 1 on any violation
//
// fdf init leaves the five Context documents as stubs. Fill them first — ask
// your agent to, and the fdf-init skill interviews you for each — because
// once a feature exists, validation fails while one is still a stub. From
// there the skills route the work by each feature's status: fdf-brainstorm
// writes the Gherkin and asks you to approve the design, fdf-plan turns the
// approved spec into tasks and test cases, and fdf-execute builds it.
//
// For a codebase that predates the bundle, fdf adopt with no arguments maps
// what is documented and which tracked code no document claims yet, and
// fdf adopt --resource <path> <slug> documents one capability as it stands.
//
// # A bundle
//
//	docs/fdf/
//	├── INDEX.md          pins the spec version the bundle follows
//	├── LOG.md            what happened to the bundle as a whole
//	├── SPEC.md           that version's spec, so the bundle describes itself
//	├── SPEC.examples.md  one example of each document
//	├── STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md, DOMAIN.md
//	├── TAGS.md           the tag dictionary
//	├── features/
//	│   └── payments/                     a group; groups nest to any depth
//	│       ├── instant-refunds.md        the feature: Gherkin scenarios and a status
//	│       ├── instant-refunds.spec.md   its approved design
//	│       ├── instant-refunds.plan.md   its implementation plan
//	│       ├── instant-refunds.test.md   how each scenario is proven
//	│       └── instant-refunds/          its tasks
//	├── changes/          Changes and Fixes to delivered features
//	├── practices/        how the project does each recurring mechanism
//	├── debts/            known gaps between the documents and the code
//	├── bugs/             known defects not repaired yet
//	└── releases/         what shipped in each version
//
// A document's ID is its path from the bundle root without .md, such as
// features/payments/instant-refunds. A delivered feature changes only
// through a Change, when it must behave differently, or a Fix, when the code
// drifted from what it says; validation will not let either be done while
// the feature still describes the old behavior.
//
// # Commands
//
// Every command, as fdf lists them when run on its own:
//
//	Set up
//	  init       Create a bundle: INDEX.md, SPEC.md, Context documents, registers
//	  install    Install the FDF skills for claude-code, codex or opencode
//	  migrate    Upgrade a 0.x, 1.0 or 1.1 bundle to spec 1.2
//
//	Features
//	  new        Start a feature to build: fdf new [<group>/…]<slug>
//	  adopt      Map code that predates the bundle, or show what is still unmapped
//	  change     Start a Change: a delivered feature must behave differently
//	  fix        Start a Fix: the code drifted from what its feature says
//	  history    Show a feature's Changes, Fixes and known bugs
//
//	Registers and practices
//	  bug        List the bug register (known defects), or file a bug
//	  debt       List the debt register (known gaps), or file a debt
//	  practice   Write down the one way the project does a recurring mechanism
//
//	Check and maintain
//	  validate   Check the bundle against its pinned spec (F1-F16, R1)
//	  log        Add an entry to the log it belongs in: fdf log [<id>] '<entry>'
//	  mv         Move or rename a document and repair every reference to it
//	  lexicon    List banned domain words (F12), or replace them with their term
//	  tag        List the tag dictionary (TAGS.md), add a tag, or merge two
//	  release    Write releases/<version>.md from the documents' version: fields
//
//	Reference
//	  spec       Print the format spec: fdf spec [-v <version>] [--examples|--full]
//	  serve      Browse the bundle in a web browser (needs bun)
//	  help       Every command in full, with examples: fdf help [<command>]
//	  version    Print the fdf version
//
// fdf help prints every command in full, with its flags and examples, and
// fdf help <command> prints one. Flags go before arguments. The bundle root
// is --root, else FDF_ROOT_DIR, else docs/fdf (or docs/features, where a
// bundle lived before 1.0); a relative root resolves against the project
// root, so fdf works from any subdirectory.
//
// # Validation in CI
//
// fdf validate exits 0 when the bundle conforms (warnings never fail it), 1
// on any violation and 2 on a usage error. Each error ends with the code of
// the rule it breaks, F1 to F16 or R1, which the bundle's own SPEC.md defines.
// One step keeps the documents honest on every pull request; in GitHub
// Actions:
//
//	steps:
//	  - uses: actions/checkout@v4
//	  - run: curl -fsSL https://giteshdalal.com/fdf.sh | bash
//	  - run: ~/.local/bin/fdf validate
//
// # Agent skills
//
// fdf install places these skills, the same for Claude Code, Codex and
// OpenCode, and a short "Feature Document Format" primer in the harness's
// instruction file (CLAUDE.md or AGENTS.md). It installs into your home
// directory, or with --project into the repository, to be committed with it.
//
//   - fdf-help routes each request to the skill its feature's status calls for.
//   - fdf-init interviews you to fill the five Context documents.
//   - fdf-adopt maps code that predates the bundle, capability by capability.
//   - fdf-brainstorm turns an idea into a feature and an approved design.
//   - fdf-plan turns an approved design into tasks and test cases.
//   - fdf-execute works the tasks and keeps every status true.
//   - fdf-change writes the Change or Fix for a delivered feature.
//   - fdf-debug finds a defect's root cause before any fix, and routes the repair.
//   - fdf-checkpoint audits the Context documents and instruction files against the code.
//   - fdf-validate turns a validation failure into its honest fix.
//
// # Links
//
//   - [project page]: what FDF is for, and how it differs from other approaches
//   - [format spec]: the rules fdf validate enforces; fdf spec prints them too
//   - [source]: the repository, its README and issues
//   - [module]: the Go module, with the README's guide to upgrading a bundle
//
// [project page]: https://giteshdalal.com/projects/fdf/
// [format spec]: https://github.com/GiteshDalal/fdf/blob/main/SPEC.md
// [source]: https://github.com/GiteshDalal/fdf
// [module]: https://pkg.go.dev/github.com/GiteshDalal/fdf
// [releases]: https://github.com/GiteshDalal/fdf/releases
// [mise]: https://mise.jdx.dev
package main
