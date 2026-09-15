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
stem-qualified siblings, with four bundle-root Context documents holding the
project's stack, architecture, surfaces and infrastructure. The fdf CLI
scaffolds those bundles, validates them, and teaches AI harnesses the
workflow.

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
      # fill STACK/ARCHITECTURE/SURFACES/INFRA via the fdf-init skill
      fdf new payments/instant-refunds     # a draft feature; write Gherkin
      # add slug.spec.md -> specified, slug.plan.md + slug.test.md -> planned,
      # tasks under slug/ -> implementing -> done
      fdf validate                         # the gate after every bundle edit

EXIT CODES
      0  success / bundle conformant
      1  validation or runtime failure
      2  usage error (bad flag, missing or unexpected argument)
`

var helpTopics = []helpTopic{
	{
		name:  "validate",
		usage: "fdf validate [--root <dir>] [--repo-root <dir>]",
		body: "Check the bundle against the spec version pinned in its root INDEX.md.\n" +
			"Every violation is reported with its rule code — F1-F9 for format\n" +
			"conformance, R1 for repo integrity. Exit 0 means conformant. Run this\n" +
			"after every bundle edit; it is the gate the fdf skills rely on.",
		flags: []string{
			"--root <dir>       bundle root (overrides FDF_ROOT_DIR; default docs/features)",
			"--repo-root <dir>  project root for R1 resource checks (default: auto-detect)",
		},
		examples: []string{
			"fdf validate",
			"fdf validate --root docs/features",
			"FDF_ROOT_DIR=wiki/features fdf validate",
		},
	},
	{
		name:  "init",
		usage: "fdf init [--root <dir>]",
		body: "Scaffold a new bundle at the resolved root: INDEX.md carrying the\n" +
			"fdf_version pin, LOG.md, a vendored copy of the spec at SPEC.md, and the\n" +
			"four Context stubs (STACK.md, ARCHITECTURE.md, SURFACES.md, INFRA.md).\n" +
			"Existing files are never overwritten. Fill the Context stubs with the\n" +
			"fdf-init skill before starting feature work — F9 blocks it otherwise.",
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
		name:  "change",
		usage: "fdf change [--root <dir>] --affects <group>/<slug>[,…] [<group>/]<slug>",
		body: "Scaffold a post-delivery Change: a request to alter what a delivered\n" +
			"feature does. Use it when the fix requires the feature's Gherkin to\n" +
			"change — including when the original document was silent on a case\n" +
			"nobody recognized. A Change carries a design gate (its .spec.md) and\n" +
			"declares, under `# Scenario changes`, the scenarios it will add, modify\n" +
			"or remove; F10 will not let it reach `done` until those landed.\n" +
			"`--affects` may name several features: one Change can span them.",
		flags: []string{
			"--affects <ids>  comma-separated feature ID(s) this touches (required)",
			"--root <dir>     bundle root (default docs/features)",
		},
		examples: []string{
			"fdf change --affects payments/instant-refunds refund-window",
			"fdf change --affects payments/instant-refunds,billing/invoices payments/tax-rounding",
		},
	},
	{
		name:  "fix",
		usage: "fdf fix [--root <dir>] --affects <group>/<slug>[,…] [<group>/]<slug>",
		body: "Scaffold a post-delivery Fix: the feature document was right and the\n" +
			"code drifted from it. No design gate — restoring documented behavior\n" +
			"needs no approval — and no trail files are required, so the floor is a\n" +
			"single file. Declares, under `# Regression cases`, scenarios that\n" +
			"already exist plus the verification for each; the lasting artifact is\n" +
			"the case added to the affected feature's .test.md.",
		flags: []string{
			"--affects <ids>  comma-separated feature ID(s) this touches (required)",
			"--root <dir>     bundle root (default docs/features)",
		},
		examples: []string{
			"fdf fix --affects payments/instant-refunds refund-rounding",
		},
	},
	{
		name:  "history",
		usage: "fdf history [--root <dir>] <group>/<slug>",
		body: "List every Change and Fix that names this feature in its `affects`.\n" +
			"Computed from frontmatter, never from back-links the feature would have\n" +
			"to maintain by hand — a required back-link is a standing invitation to\n" +
			"drift. This is how you answer \"what has happened to this feature since\n" +
			"it shipped?\".",
		flags:    []string{"--root <dir>  bundle root (default docs/features)"},
		examples: []string{"fdf history payments/instant-refunds"},
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
