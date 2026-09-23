package main

import (
	"fmt"
	"io"
	"strings"
)

// helpTopic is one command's long-form entry. Keeping them as data (rather
// than one prose blob) lets `fdf help <command>` print a single entry.
type helpTopic struct {
	name     string
	usage    string
	body     string
	flags    []string
	examples []string
}

var helpPreamble = banner + `

FDF documents each software feature as a Markdown + Gherkin file whose design
spec, implementation plan, acceptance tests and tasks live beside it as
stem-qualified siblings, with five bundle-root Context documents holding the
project's stack, architecture, surfaces, infrastructure and domain language.
The fdf CLI scaffolds those bundles, validates them, and teaches AI harnesses
the workflow.

BUNDLE ROOT
  Every command resolves the bundle root the same way:

      --root <dir>   >   FDF_ROOT_DIR   >   docs/features

  Relative values resolve against the project root (the topmost enclosing
  .git), so the bundle is found from any subdirectory; absolute values are
  used as-is. A bundle may also be a git submodule mounted at that path.

  Flags must precede positional arguments:

      fdf new --root docs/bundle payments/refunds    # correct
      fdf new payments/refunds --root docs/bundle    # --root is NOT parsed

TYPICAL FLOW
      fdf init                             # scaffold bundle + Context stubs
      # fill STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN via the fdf-init skill
      fdf new payments/instant-refunds     # a draft feature; write Gherkin
      # add slug.spec.md (+ slug.surface.md when it exposes an interface)
      # -> specified, slug.plan.md + slug.test.md -> planned,
      # tasks under slug/ -> implementing -> done
      fdf log payments/instant-refunds "**Specified**: design approved."
      fdf validate                         # the gate after every bundle edit

EXIT CODES
      0  success / bundle conformant
      1  validation or runtime failure
      2  usage error (bad flag, missing or unexpected argument)
`

var helpTopics = []helpTopic{
	{
		name:  "validate",
		usage: "fdf validate [--root <dir>] [--repo-root <dir>] [--strict-domain]",
		body: "Check the bundle against the spec version pinned in its root INDEX.md.\n" +
			"Every violation is reported with its rule code — F1-F14 for format\n" +
			"conformance, R1 for repo integrity. Exit 0 means conformant. Run this\n" +
			"after every bundle edit; it is the gate the fdf skills rely on.",
		flags: []string{
			"--root <dir>       bundle root (overrides FDF_ROOT_DIR; default docs/features)",
			"--repo-root <dir>  project root for R1 resource checks (default: auto-detect)",
			"--strict-domain    report F12 banned domain words as errors, not warnings (DOMAIN.md's `strict: true` does it for every run)",
		},
		examples: []string{
			"fdf validate",
			"fdf validate --root docs/features",
			"fdf validate --strict-domain",
			"FDF_ROOT_DIR=wiki/features fdf validate",
		},
	},
	{
		name:  "init",
		usage: "fdf init [--root <dir>]",
		body: "Scaffold a new bundle at the resolved root: INDEX.md carrying the\n" +
			"fdf_version pin, LOG.md, a vendored copy of the spec at SPEC.md, and the\n" +
			"five Context stubs (STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md,\n" +
			"DOMAIN.md). Existing files are never overwritten. Fill the Context stubs\n" +
			"with the fdf-init skill before starting feature work — F9 blocks it\n" +
			"otherwise.",
		flags:    []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{"fdf init", "fdf init --root docs/features"},
	},
	{
		name:  "new",
		usage: "fdf new [--root <dir>] <group>/<slug>",
		body: "Scaffold a draft feature at <group>/<slug>.md with frontmatter and empty\n" +
			"Gherkin fences, creating the group directory and its INDEX.md if needed.\n" +
			"Group and slug are lowercase [a-z0-9-]. A draft carries no trail siblings\n" +
			"and no task directory; those arrive as the feature advances.",
		flags: []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{
			"fdf new payments/instant-refunds",
			"fdf new --root docs/features auth/passkey-login",
		},
	},
	{
		name:  "adopt",
		usage: "fdf adopt [--root <dir>] [--resource <path>[,…]] [--depth <n>] [<group>/<slug>]",
		body: "Map what a codebase already does (v0.7). With a feature ID it scaffolds\n" +
			"an adopted feature — a capability documented from the code as it stands,\n" +
			"never built through FDF, so it gets no spec, plan or tasks. It starts as\n" +
			"a map entry: a Feature: block and --resource, the code it lives in (it\n" +
			"must exist). Scenarios are backfilled later, each with a .test.md case\n" +
			"that passes against the code as it stands.\n\n" +
			"Without an ID it prints the adoption map: every feature with its status,\n" +
			"scenarios and tested scenarios, then the tracked code (git ls-files) no\n" +
			"feature, task, change or fix claims yet, grouped --depth levels deep,\n" +
			"most unclaimed first — the list a phased adoption works down.",
		flags: []string{
			"--resource <paths>  comma-separated project-relative path(s) of the capability's code (required when mapping one)",
			"--depth <n>         directory levels the unclaimed code is grouped by (default 2)",
			"--root <dir>        bundle root (default docs/features)",
		},
		examples: []string{
			"fdf adopt --resource internal/payments/card.go payments/card-payments",
			"fdf adopt",
			"fdf adopt --depth 3",
		},
	},
	{
		name:  "practice",
		usage: "fdf practice [--root <dir>] [<group>/]<slug>",
		body: "Scaffold a Practice under practices/ — the project's binding answer to\n" +
			"how one recurring mechanism is done (authorization, permission checks,\n" +
			"payment capture, database access). A practice is a living document with\n" +
			"no spec, plan, test or tasks; its only sibling is an optional\n" +
			"<slug>.log.md. Fill `# Rules` with the binding statements and set\n" +
			"`applies-to` to the repo paths it governs — that is how later work is\n" +
			"routed to it, since features never list the practices they follow.\n" +
			"A practice binds all future code: land one only with human approval.\n" +
			"v0.6 bundles and later.",
		flags: []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{
			"fdf practice permission-checks",
			"fdf practice payments/idempotency",
		},
	},
	{
		name:  "debt",
		usage: "fdf debt [--root <dir>] [--open|--accepted|--resolved] [--cleanup [--dry-run] [--no-log]] [[<group>/]<slug>]",
		body: "Read or file the debt register under debts/ — the known gaps between\n" +
			"what the project says and what the code does: work left undone, and\n" +
			"rules the codebase does not follow everywhere yet.\n\n" +
			"With a slug it scaffolds a debt (" + registerStatusList() + "); without one it\n" +
			"prints the register as a table of status, id, filing date and title.\n" +
			"`resource` names the paths carrying the gap — that is how later work\n" +
			"finds the debt, and R1 makes a debt pointing at vanished code loud.\n\n" +
			"--cleanup is the chore that keeps the register worth reading: each\n" +
			"resolved debt is recorded in debts/LOG.md as one line and its file is\n" +
			"removed. Open and accepted debts are never touched. Use --dry-run to\n" +
			"see the plan first, or --no-log to remove without recording.\n" +
			"v0.6 bundles and later.",
		flags: []string{
			"--root <dir>  bundle root (default docs/features)",
			"--open        list only open debts",
			"--accepted    list only accepted debts",
			"--resolved    list only resolved debts",
			"--cleanup     clear resolved debts, recording each in debts/LOG.md",
			"--resource <paths>  when filing: comma-separated path(s) carrying the gap",
			"--dry-run     with --cleanup: show what would be cleared, change nothing",
			"--no-log      with --cleanup: remove without writing debts/LOG.md",
		},
		examples: []string{
			"fdf debt",
			"fdf debt --open",
			"fdf debt --cleanup --dry-run",
			"fdf debt authz-legacy-handlers",
		},
	},
	{
		name:  "bug",
		usage: "fdf bug [--root <dir>] [--open|--accepted|--resolved] [--cleanup [--dry-run]] [--affects <ids>] [[<group>/]<slug>]",
		body: "Read or file the bug register under bugs/ (v0.7) — known defects, the\n" +
			"software doing something wrong that someone could observe, that nobody\n" +
			"is repairing yet: undiagnosed, waiting on a decision, in code no feature\n" +
			"documents, or deferred. A gap nothing observable shows is a debt.\n\n" +
			"With a slug it scaffolds a bug (" + registerStatusList() + ") with `# Symptom` and\n" +
			"`# Expected`. When a scenario already promises the expected behavior,\n" +
			"cite it under `# Violates` and the repair is a Fix: `fdf fix --from\n" +
			"bugs/<id> …`; when none does, the repair is a Change that decides it:\n" +
			"`fdf change --from bugs/<id> …`. A bug is never resolved in place — the\n" +
			"Fix or Change names it in `resolves`. Without a slug it prints the\n" +
			"register; --cleanup folds resolved bugs into bugs/LOG.md as\n" +
			"`fdf debt --cleanup` does, and always logs them: a done Fix or Change that\n" +
			"`resolves` a cleared bug is checked against bugs/LOG.md (F10).",
		flags: []string{
			"--affects <ids>     when filing: comma-separated feature ID(s) the defect shows up in",
			"--resource <paths>  when filing: comma-separated path(s) carrying it",
			"--open           list only open bugs (also --accepted, --resolved)",
			"--cleanup        clear resolved bugs, recording each in bugs/LOG.md",
			"--dry-run        with --cleanup: show what would be cleared, change nothing",
			"--root <dir>     bundle root (default docs/features)",
		},
		examples: []string{
			"fdf bug --open",
			"fdf bug --affects payments/instant-refunds --resource internal/payments/refund.go refund-split-capture",
			"fdf bug --cleanup --dry-run",
		},
	},
	{
		name:  "change",
		usage: "fdf change [--root <dir>] [--from bugs/<id>] --affects <group>/<slug>[,…] [<group>/]<slug>",
		body: "Scaffold a post-delivery Change: a request to alter what a delivered\n" +
			"feature does. Use it when the fix requires the feature's Gherkin to\n" +
			"change — including when the original document was silent on a case\n" +
			"nobody recognized. A Change carries a design gate (its .spec.md) and\n" +
			"declares, under `# Scenario changes`, the scenarios it will add, modify\n" +
			"or remove; F10 will not let it reach `done` until those landed.\n" +
			"`--affects` may name several features: one Change can span them.\n" +
			"--from bugs/<id> repairs a bug nobody documented the answer to: it\n" +
			"copies the bug's analysis into `# Problem` and writes `resolves`.",
		flags: []string{
			"--affects <ids>  comma-separated feature ID(s) this touches (required unless --from supplies them)",
			"--from <bug>     bugs/<id> this repairs: copies its analysis, writes `resolves`",
			"--root <dir>     bundle root (default docs/features)",
		},
		examples: []string{
			"fdf change --affects payments/instant-refunds refund-window",
			"fdf change --from bugs/export-drops-last-row export-keeps-every-row",
			"fdf change --affects payments/instant-refunds,billing/invoices payments/tax-rounding",
		},
	},
	{
		name:  "fix",
		usage: "fdf fix [--root <dir>] [--from bugs/<id>] --affects <group>/<slug>[,…] [<group>/]<slug>",
		body: "Scaffold a post-delivery Fix: the feature document was right and the\n" +
			"code drifted from it. No design gate — restoring documented behavior\n" +
			"needs no approval — and no trail files are required, so the floor is a\n" +
			"single file. Declares, under `# Regression cases`, scenarios that\n" +
			"already exist plus the verification for each; the lasting artifact is\n" +
			"the case added to the affected feature's .test.md. --from bugs/<id>\n" +
			"takes over a bug's `# Symptom` and `# Root cause`, turns its `# Violates`\n" +
			"scenarios into regression cases, and writes `resolves`.",
		flags: []string{
			"--affects <ids>  comma-separated feature ID(s) this touches (required unless --from supplies them)",
			"--from <bug>     bugs/<id> this repairs: copies its analysis, writes `resolves`",
			"--root <dir>     bundle root (default docs/features)",
		},
		examples: []string{
			"fdf fix --affects payments/instant-refunds refund-rounding",
			"fdf fix --from bugs/refund-split-capture refund-split-capture",
		},
	},
	{
		name:  "history",
		usage: "fdf history [--root <dir>] <group>/<slug>",
		body: "List every Change and Fix that names this feature in its `affects`,\n" +
			"the bugs each resolves, and the bugs on the register it shows up in.\n" +
			"Computed from frontmatter, never from back-links the feature would have\n" +
			"to maintain by hand — a required back-link is a standing invitation to\n" +
			"drift. This is how you answer \"what has happened to this feature since\n" +
			"it shipped?\".",
		flags:    []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{"fdf history payments/instant-refunds"},
	},
	{
		name:  "mv",
		usage: "fdf mv [--root <dir>] [--dry-run] <from-id> <to-id>",
		body: "Move or rename a document with everything it owns — a feature with its\n" +
			"spec, plan, test, surface, log and task directory; a change or fix; a\n" +
			"practice, debt or bug; a task within its directory; or a whole group —\n" +
			"and repair every reference to it across the bundle, frozen documents\n" +
			"included (v0.7's reference repair): links, `affects`, `depends-on`,\n" +
			"`replaced-by`, `retires`, `superseded-by`, `resolves`, declaration\n" +
			"headings, index listings and ID mentions. Logs keep their words and get\n" +
			"their links repaired. A debt and a bug can be re-filed as each other.\n" +
			"The move is logged in LOG.md and validated; references outside the\n" +
			"bundle are listed, never edited. It never overwrites.",
		flags: []string{
			"--dry-run     print what would move and what would be repaired",
			"--root <dir>  bundle root (default docs/features)",
		},
		examples: []string{
			"fdf mv payments/store-hours venues/opening-hours",
			"fdf mv --dry-run payments billing",
			"fdf mv debts/export-drops-rows bugs/export-drops-rows",
			"fdf mv payments/refunds/02-ui payments/refunds/03-ui",
		},
	},
	{
		name:  "lexicon",
		usage: "fdf lexicon [--root <dir>] [--term <Term>] [--all] [--fix [--dry-run]]",
		body: "Report every banned word F12 sees (v0.7) — file:line:col and the line\n" +
			"around it, grouped by word — and every name using one, with a suggested\n" +
			"`fdf mv`. Triage first: a word used in another sense is qualified and\n" +
			"listed under the term's `except:`; a mention of a word goes in a code\n" +
			"span. Then --fix replaces the rest with the term, one term at a time:\n" +
			"plurals, capitals and a/an are kept right, and a scenario name is renamed\n" +
			"everywhere it is a join, slug.test.md included. Italic mentions, labels\n" +
			"quoted in Gherkin steps, table cells and names are left for a person and\n" +
			"listed. The sweep is logged in LOG.md and validated.",
		flags: []string{
			"--term <Term>  only that term's banned words (required with --fix)",
			"--all          list every occurrence, not the first few per word",
			"--fix          replace them with the term (a lexicon fix)",
			"--dry-run      with --fix: print the diff, change nothing",
			"--root <dir>   bundle root (default docs/features)",
		},
		examples: []string{
			"fdf lexicon",
			"fdf lexicon --term Venue --all",
			"fdf lexicon --term Venue --fix --dry-run",
			"fdf lexicon --term Venue --fix",
		},
	},
	{
		name:  "log",
		usage: "fdf log [--root <dir>] [<id>] \"<entry>\"",
		body: "Add one entry to the log it belongs in, under today's `## YYYY-MM-DD`\n" +
			"heading, newest first. An entry goes in the log of the one document it\n" +
			"is about: a feature, change, fix, practice, debt or bug ID puts it in\n" +
			"that document's <slug>.log.md, created on first use, and a task or trail\n" +
			"document's entry goes in its owner's log. A group ID puts it in the\n" +
			"group's LOG.md. With no ID, or for a Context document or a release, it\n" +
			"goes in the bundle-root LOG.md, which is for the bundle as a whole. A\n" +
			"draft feature has no log yet: its first entry comes with its approved\n" +
			"spec. Start an entry with a bold label naming the event (**Specified**,\n" +
			"**Decision**, **Done**), then say what happened and, for a decision, why.",
		flags: []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{
			"fdf log payments/instant-refunds \"**Specified**: design approved; synchronous PSP call, reconciled from the webhook.\"",
			"fdf log changes/payments/refund-window \"**Decision**: refunds already in flight keep the 180-day window.\"",
			"fdf log \"**Checkpoint**: Context documents re-read against the code; STACK.md updated.\"",
		},
	},
	{
		name:  "release",
		usage: "fdf release [--root <dir>] [--date <YYYY-MM-DD>] [--ship] <version>",
		body: "Create or refresh releases/<version>.md from the `version:` fields\n" +
			"already carried by features, changes and fixes. Those lists are the one\n" +
			"thing in FDF that is wholly derived, so hand-maintaining them is pure\n" +
			"drift surface. Idempotent, and `# Notes` is human prose that is\n" +
			"preserved untouched. What it never derives is membership: which work\n" +
			"ships is a human decision recorded as `version:` on each document.\n" +
			"--ship refuses while any listed document is still open.",
		flags: []string{
			"--date <YYYY-MM-DD>  target date while planned, actual date once shipped",
			"--ship               flip to shipped once every listed document is done",
			"--root <dir>         bundle root (default docs/features)",
		},
		examples: []string{
			"fdf release 1.2.0",
			"fdf release --date 2026-10-01 1.2.0",
			"fdf release --ship 1.2.0",
		},
	},
	{
		name:  "install",
		usage: "fdf install [--project] [--root <dir>] <claude-code|codex|opencode>",
		body: "Install or upgrade the FDF skills and that harness's adapter, plus a\n" +
			"'## Feature Document Format' primer in its instruction file. Idempotent:\n" +
			"re-running upgrades the managed blocks and never clobbers your edits.\n" +
			"Installs into your home directory by default; --project installs into the\n" +
			"current git project (nearest .git) so the setup is committed with the code.\n" +
			"Re-run it after `fdf migrate` so the skills teach the new layout.",
		flags: []string{
			"--project     install into the current git project instead of the home directory",
			"--root <dir>  bundle root to bake into the installed skills",
		},
		examples: []string{
			"fdf install claude-code",
			"fdf install --project claude-code",
			"fdf install --project --root docs/features codex",
		},
	},
	{
		name:  "serve",
		usage: "fdf serve [--root <dir>]",
		body: "Serve the bundle in a browser by wrapping `bun x mdts`, so the markdown\n" +
			"renders with working cross-links. Requires bun (https://bun.sh).",
		flags:    []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{"fdf serve", "fdf serve --root docs/features"},
	},
	{
		name:  "migrate",
		usage: "fdf migrate [--root <dir>]",
		body: "Upgrade a bundle to the spec version this binary ships: rewrite the\n" +
			"fdf_version pin, lift nested trail files to stem-qualified siblings,\n" +
			"re-vendor SPEC.md, scaffold any missing Context stubs, then validate.\n" +
			"A pre-flight refuses to start on content the new layout cannot hold and\n" +
			"leaves the bundle untouched, so a refused run is always safe to retry\n" +
			"after fixing what it names. Freshly scaffolded Context stubs are warnings\n" +
			"here, not errors — a plain `fdf validate` still fails F9 until you fill\n" +
			"them. Re-run `fdf install` afterwards.",
		flags:    []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{"fdf migrate", "fdf migrate --root docs/features"},
	},
	{
		name:  "spec",
		usage: "fdf spec [-v <version>] [--list]",
		body: "Print the format specification, straight from the copy embedded in this\n" +
			"binary — no bundle, no network, and no checkout of the fdf repository\n" +
			"needed. Defaults to the current version; -v prints an older one, which is\n" +
			"what a bundle still pinning that version must satisfy.",
		flags: []string{
			"-v <version>  spec version to print (default: the current version)",
			"--list        list the spec versions embedded in this binary",
		},
		examples: []string{"fdf spec", "fdf spec -v 0.3", "fdf spec --list", "fdf spec | less"},
	},
	{
		name:     "version",
		usage:    "fdf version",
		body:     "Print the CLI version.",
		examples: []string{"fdf version"},
	},
	{
		name:     "help",
		usage:    "fdf help [<command>]",
		body:     "Print this help. With a command name, print only that command's entry.",
		examples: []string{"fdf help", "fdf help migrate"},
	},
}

func (t helpTopic) render(w io.Writer) {
	fmt.Fprintf(w, "  %s\n", t.usage)
	for _, line := range strings.Split(t.body, "\n") {
		fmt.Fprintf(w, "      %s\n", line)
	}
	if len(t.flags) > 0 {
		fmt.Fprintln(w)
		for _, f := range t.flags {
			fmt.Fprintf(w, "      %s\n", f)
		}
	}
	if len(t.examples) > 0 {
		fmt.Fprintln(w)
		for _, e := range t.examples {
			fmt.Fprintf(w, "      $ %s\n", e)
		}
	}
	fmt.Fprintln(w)
}

// runHelp prints the long-form help: the whole document, or one command's
// entry when named. `fdf` with no arguments still prints the short usage.
func runHelp(args []string, stdout io.Writer) int {
	fs := newFlagSet("help", stdout)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) > 1 {
		fmt.Fprintln(stdout, "usage: fdf help [<command>]")
		return 2
	}
	if len(rest) == 1 {
		for _, t := range helpTopics {
			if t.name == rest[0] {
				fmt.Fprintln(stdout)
				t.render(stdout)
				return 0
			}
		}
		var names []string
		for _, t := range helpTopics {
			names = append(names, t.name)
		}
		fmt.Fprintf(stdout, "fdf help: unknown command %q\ncommands: %s\n", rest[0], strings.Join(names, ", "))
		return 2
	}
	fmt.Fprint(stdout, helpPreamble)
	fmt.Fprintln(stdout, "\nCOMMANDS")
	for _, t := range helpTopics {
		t.render(stdout)
	}
	return 0
}
