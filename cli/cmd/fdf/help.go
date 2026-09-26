package main

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/GiteshDalal/fdf/cli/internal/install"
)

// helpTopic is one command's entry. The topics are the one source of what fdf
// says about its commands: the overview `fdf` prints on its own, `fdf help`,
// `fdf <command> --help`, and the usage line of every usage error.
type helpTopic struct {
	name     string
	group    string // the overview section it is listed under
	summary  string // its line in the overview
	usage    string
	body     string // wrapped at 74 columns
	flags    []flagDoc
	examples []string
}

// flagDoc is one row of a topic's flag table.
type flagDoc struct{ flag, text string }

// helpWidth is the widest line the help prints.
const helpWidth = 80

var rootFlagDoc = flagDoc{"--root <dir>", "bundle root (default docs/fdf, or a docs/features from before 1.0; FDF_ROOT_DIR overrides it)"}

var helpPreamble = banner + `

FDF documents each software feature as a Markdown + Gherkin file. Its design
spec, plan and test cases sit beside it as siblings named after it, and its
tasks in a directory of the same name. Five Context documents at the bundle
root hold the project's stack, architecture, surfaces, infrastructure and
domain language. Changes and Fixes record work on a delivered feature;
practices say how a recurring mechanism is always done; the debt and bug
registers hold what is left undone and what is broken. fdf scaffolds and
checks the bundle, keeps its names and references consistent, and installs
the skills that teach an AI agent the workflow.

IDS
  A document's ID is its path from the bundle root without .md, so it starts
  with its register: features/payments/instant-refunds,
  bugs/refund-split-capture, changes/refund-window. Commands that name an
  existing document take its ID: log, mv, history, --affects, --from. The
  commands that create one (new, adopt, practice, debt, bug, change, fix)
  take its name inside its register, [<group>/…]<slug>, with groups nested
  to any depth: fdf new payments/instant-refunds files
  features/payments/instant-refunds, and fdf debt authz-legacy-handlers
  files debts/authz-legacy-handlers, as does fdf debt
  debts/authz-legacy-handlers.

BUNDLE ROOT
  Every command resolves the bundle root the same way:

      --root <dir>   >   FDF_ROOT_DIR   >   docs/fdf

  With neither set, fdf also looks in docs/features, where a bundle lived
  before 1.0, and takes it when docs/fdf holds no bundle; when both do, it
  warns. A bundle there is an INDEX.md that pins fdf_version, so a
  documentation site's docs/features is left alone. Relative values
  resolve against the project root (the topmost enclosing git repository,
  or the linked worktree you are in), so the bundle is found from any
  subdirectory; absolute values are used as-is. A bundle may also be a git
  submodule mounted at that path.

  Flags go before arguments. A flag after an argument is refused, and fdf
  prints the command as it should have been typed:

      fdf new --root docs/bundle payments/refunds    # correct
      fdf new payments/refunds --root docs/bundle    # refused

TYPICAL FLOW
  A new capability:
      fdf init                             # bundle + Context stubs
      # fill STACK/ARCHITECTURE/SURFACES/INFRA/DOMAIN with the fdf-init skill
      fdf new payments/instant-refunds     # a draft feature; write its Gherkin
      # slug.spec.md approved, plus slug.surface.md or ` + "`surface: none`" + `
      #   -> specified; slug.plan.md + slug.test.md, one
      #   ` + "`## <scenario name>`" + ` case per scenario -> planned;
      #   tasks under slug/ -> implementing -> done
      fdf log features/payments/instant-refunds "**Specified**: approved."
      fdf validate                         # the gate after every bundle edit

  Code that predates the bundle:
      fdf adopt                            # what is mapped, and what is not
      fdf adopt --resource internal/payments/card.go payments/card-payments

  After delivery:
      fdf change --affects features/payments/instant-refunds refund-window
      fdf fix --affects features/payments/instant-refunds refund-rounding
      fdf bug --affects features/payments/instant-refunds refund-split-capture

  Every date and time fdf writes is UTC.

EXIT CODES
      0  success / bundle conformant (warnings never fail it)
      1  validation or runtime failure
      2  usage error (bad flag, missing or unexpected argument)
`

// helpTopics are in overview order: grouped, and within a group in the order
// the work usually meets them.
var helpTopics = []helpTopic{
	{
		name:    "init",
		group:   "Set up",
		summary: "Create a bundle: INDEX.md, SPEC.md, Context documents, registers",
		usage:   "fdf init [--root <dir>]",
		body: "Scaffold a new bundle at the resolved root, docs/fdf by default: INDEX.md\n" +
			"carrying the fdf_version pin and listing the registers, LOG.md, the spec\n" +
			"at SPEC.md, the five Context stubs (STACK.md, ARCHITECTURE.md,\n" +
			"SURFACES.md, INFRA.md, DOMAIN.md), and the indexes of features/,\n" +
			"changes/, practices/, debts/ and bugs/. It never overwrites: on a bundle\n" +
			"already at this version it only adds what is missing, and on an older\n" +
			"one it points you to `fdf migrate`. A directory inside a bundle, one of\n" +
			"its registers or groups, is refused, and so is one that holds Markdown\n" +
			"but no INDEX.md, such as a bundle from before 1.0 that never had one,\n" +
			"which is `fdf migrate`'s. Fill the Context stubs with the fdf-init skill\n" +
			"before feature work: once a feature exists, F9 fails while any of them\n" +
			"is still a stub.",
		flags:    []flagDoc{rootFlagDoc},
		examples: []string{"fdf init", "fdf init --root wiki/fdf"},
	},
	{
		name:    "install",
		group:   "Set up",
		summary: "Install the FDF skills for claude-code, codex or opencode",
		usage:   "fdf install [--project] [--root <dir>] <claude-code|codex|opencode>",
		body: "Install or upgrade the " + countWord(len(install.SkillNames())) + " FDF skills for an AI agent, plus a\n" +
			"'## Feature Document Format' primer in its instruction file (CLAUDE.md or\n" +
			"AGENTS.md). Idempotent: re-running upgrades the skills and refreshes the\n" +
			"primer whenever this build's text differs from the installed one, even at\n" +
			"the same version, unless you edited the primer — an edited primer is left\n" +
			"as it is, with a note. Installs into your home directory by default;\n" +
			"--project installs into the current git project (nearest .git) so the\n" +
			"setup is committed with the code. Re-run it after `fdf migrate` so the\n" +
			"skills teach the new layout.",
		flags: []flagDoc{
			{"--project", "install into the current git project instead of your home directory"},
			{"--root <dir>", "bundle root the skills name (default docs/features, or FDF_ROOT_DIR)"},
		},
		examples: []string{
			"fdf install claude-code",
			"fdf install --project claude-code",
			"fdf install --project --root docs/features codex",
		},
	},
	{
		name:    "migrate",
		group:   "Set up",
		summary: "Upgrade a 0.x bundle to spec 1.0 in one run",
		usage:   "fdf migrate [--root <dir>] [--dry-run] [--to <dir>] [--skip <glob>]",
		body: "Upgrade a bundle at any 0.x pin to spec 1.0, then validate it. It works\n" +
			"out the whole migration and prints it before it writes anything; with\n" +
			"--dry-run it stops there. Older layouts are made 0.7-shaped first (v0.1's\n" +
			"renames, v0.3's trail lift, the status tags older tools wrote after index\n" +
			"listings). Then every feature group moves into features/, and every\n" +
			"mention of a feature's ID gains features/, in frozen documents too; logs\n" +
			"keep their words. Every link is repaired, features/INDEX.md takes the\n" +
			"groups' listings from INDEX.md, the other registers get their indexes, and\n" +
			"the pin, the vendored SPEC.md and any missing Context stub follow. The\n" +
			"migration is logged in LOG.md. A bundle at docs/features then moves to\n" +
			"docs/fdf beside it, or where --to says; a submodule moves with git mv. In\n" +
			"a git repository migrate also rewrites the rest of the project's\n" +
			"git-tracked text files: each Markdown link into the bundle, and each\n" +
			"mention of its path. Each mention of the old path it leaves for a person\n" +
			"to decide on — inside a URL, after a longer path, or in a link that leads\n" +
			"elsewhere — is listed, and so is a symbolic link the move breaks that\n" +
			"migrate cannot name again: an absolute one, or one outside the bundle.\n" +
			"Those in logs, which keep their words, are counted, and so are those in\n" +
			"what `fdf install` manages, which is left to it.\n" +
			"--skip leaves the files a glob names outside the bundle as they are, such\n" +
			"as applied SQL migrations whose checksums a tool verifies, and lists each\n" +
			"mention of the old path in them; a glob reads from the project root, as\n" +
			"git reads one: * within a directory, ** across them.\n" +
			"Migrate starts only from a clean tree, in the repository that tracks the\n" +
			"bundle, refuses to put a file where git would ignore it, and marks the\n" +
			"files it writes with `git add -N`, so that `git diff -M` shows each move.\n" +
			"It prints the commands that put everything back should it stop partway,\n" +
			"and, once done, those that back it out, starting with the `git reset` of\n" +
			"those marks that git stash and git clean would trip on. A pre-flight\n" +
			"refuses what 1.0 has no place for, and what migrate cannot move safely — a\n" +
			"stray Markdown file at the root, a document named index.md or log.md, a\n" +
			"practice, debt or bug beside a directory of Markdown, a register that is a\n" +
			"symbolic link or whose place a file takes, a v0.1 rename onto a file that\n" +
			"is there — and leaves the bundle untouched, so a refused run is safe to\n" +
			"retry after fixing what it names. A bundle already at 1.0 moves nothing:\n" +
			"its spec copy, indexes and Context stubs are restored, never through a\n" +
			"symbolic link. An unfilled Context stub is only a warning here; once the\n" +
			"bundle has a feature, a plain `fdf validate` fails F9 until it is filled.\n" +
			"Re-run `fdf install` afterwards.",
		flags: []flagDoc{
			{"--dry-run", "print the plan, and change nothing"},
			{"--to <dir>", "where the bundle goes (default docs/fdf beside a docs/features bundle, otherwise where it is)"},
			{"--skip <glob>", "leave the files the glob names outside the bundle as they are, listing what they say of it; repeatable"},
			rootFlagDoc,
		},
		examples: []string{"fdf migrate --dry-run", "fdf migrate", "fdf migrate --root docs/features --to docs/fdf", "fdf migrate --skip 'db/migrations/**'"},
	},
	{
		name:    "new",
		group:   "Features",
		summary: "Start a feature to build: fdf new [<group>/…]<slug>",
		usage:   "fdf new [--root <dir>] [<group>/…]<slug>",
		body: "Scaffold a draft feature at features/[<group>/…]<slug>.md with\n" +
			"frontmatter and placeholder Gherkin, listed in the index beside it. A\n" +
			"feature is filed flat or in groups nested to any depth; a new group gets\n" +
			"its directory and INDEX.md, and is listed in its parent's. Names are\n" +
			"lowercase [a-z0-9-], and none is reserved inside features/. A name that\n" +
			"clashes with what is beside it is refused: a feature named like a group\n" +
			"would make that group its task directory. A draft may have a log but no\n" +
			"other sibling and no task directory; the rest arrive as it advances.",
		flags: []flagDoc{rootFlagDoc},
		examples: []string{
			"fdf new payments/instant-refunds",
			"fdf new onboarding",
			"fdf new platform/payouts/weekly-payouts",
		},
	},
	{
		name:    "adopt",
		group:   "Features",
		summary: "Map code that predates the bundle, or show what is still unmapped",
		usage:   "fdf adopt [--root <dir>] [--resource <paths>] [--depth <n>] [[<group>/…]<slug>]",
		body: "Map what a codebase already does. With a name it scaffolds an adopted\n" +
			"feature at features/[<group>/…]<slug>.md: a capability documented from\n" +
			"the code as it stands, never built through FDF, so it has no spec, plan\n" +
			"or tasks. It starts as a map entry: a `Feature:` block, and `resource`\n" +
			"naming the code it lives in (which must exist). Scenarios are backfilled\n" +
			"later, each with its case in slug.test.md, passing against the code as\n" +
			"it stands.\n" +
			"\n" +
			"Without an ID it prints the adoption map: every feature with its status,\n" +
			"scenario count and tested count, then the tracked code (git ls-files) that\n" +
			"no feature, task, Change or Fix names in `resource` yet, grouped --depth\n" +
			"directory levels deep, most unclaimed first. Work down that list.",
		flags: []flagDoc{
			{"--resource <paths>", "comma-separated project paths of the capability's code (required with an ID)"},
			{"--depth <n>", "directory levels the unclaimed code is grouped by (default 2)"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf adopt --resource internal/payments/card.go payments/card-payments",
			"fdf adopt",
			"fdf adopt --depth 3",
		},
	},
	{
		name:    "change",
		group:   "Features",
		summary: "Start a Change: a delivered feature must behave differently",
		usage:   "fdf change [--root <dir>] [--from bugs/<id>] [--affects <feature-id>[,…]] [<group>/…]<slug>",
		body: "Scaffold a Change under changes/: a request to alter what a delivered\n" +
			"(done or adopted) feature does. Use it when the feature's Gherkin has to\n" +
			"change, including when its document was silent on a case nobody foresaw.\n" +
			"A Change passes the design gate — an approved <slug>.spec.md — and\n" +
			"declares under `# Scenario changes` the scenarios it will add, modify or\n" +
			"remove; F10 will not let it reach `done` until they have. --affects\n" +
			"takes features' full IDs (features/…), and may name several: one Change\n" +
			"can span them. --from bugs/<id> starts from a filed bug that no scenario\n" +
			"covers yet: it copies the bug's `# Symptom` and `# Expected` into\n" +
			"`# Problem`, takes its `affects`, and names it in `resolves`.",
		flags: []flagDoc{
			{"--affects <ids>", "comma-separated feature IDs this touches (required unless --from supplies them)"},
			{"--from bugs/<id>", "the bug this repairs: copies its analysis, writes `resolves`"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf change --affects features/payments/instant-refunds refund-window",
			"fdf change --from bugs/refund-to-closed-card refund-to-closed-card",
			"fdf change --affects features/payments/refunds,features/payments/cards payments/partial-refunds",
		},
	},
	{
		name:    "fix",
		group:   "Features",
		summary: "Start a Fix: the code drifted from what its feature says",
		usage:   "fdf fix [--root <dir>] [--from bugs/<id>] [--affects <feature-id>[,…]] [<group>/…]<slug>",
		body: "Scaffold a Fix under changes/: the feature document was right and the\n" +
			"code drifted from it. It has no design gate — restoring documented\n" +
			"behavior needs no approval — and needs no spec, plan or tasks, so the\n" +
			"smallest Fix is a single file. It lists under `# Regression cases` the\n" +
			"existing scenarios it proves, each with its verification; the lasting\n" +
			"record is each case in the affected feature's slug.test.md. --affects\n" +
			"takes features' full IDs (features/…). --from bugs/<id> takes over a\n" +
			"bug's `# Symptom` and `# Root cause`, turns its `# Violates` scenarios\n" +
			"into regression cases, and names it in `resolves`.",
		flags: []flagDoc{
			{"--affects <ids>", "comma-separated feature IDs this touches (required unless --from supplies them)"},
			{"--from bugs/<id>", "the bug this repairs: copies its analysis, writes `resolves`"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf fix --affects features/payments/instant-refunds refund-rounding",
			"fdf fix --from bugs/refund-split-capture refund-split-capture",
		},
	},
	{
		name:    "history",
		group:   "Features",
		summary: "Show a feature's Changes, Fixes and known bugs",
		usage:   "fdf history [--root <dir>] <feature-id>",
		body: "Show what has happened to a feature since it was delivered: every Change\n" +
			"and Fix that names it in `affects`, the bugs each one resolves, and the\n" +
			"bugs on the register that name it. It is computed from those documents'\n" +
			"frontmatter, so the feature never has to keep back-links by hand. It\n" +
			"takes the feature's full ID, features/[<group>/…]<slug>.",
		flags:    []flagDoc{rootFlagDoc},
		examples: []string{"fdf history features/payments/instant-refunds"},
	},
	{
		name:    "bug",
		group:   "Registers and practices",
		summary: "List the bug register (known defects), or file a bug",
		usage:   "fdf bug [--root <dir>] [--open|--accepted|--resolved] [--cleanup [--dry-run]] [--affects <ids>] [--resource <paths>] [[<group>/…]<slug>]",
		body: "Read or file the bug register under bugs/: known defects — the software\n" +
			"doing something wrong that someone could observe — that have not been\n" +
			"repaired yet. A gap no one could observe is a debt instead.\n" +
			"\n" +
			"Without a slug it prints the register. With a slug it files a new bug,\n" +
			"flat or in groups nested to any depth, status `open`, with `# Symptom` and\n" +
			"`# Expected` to fill; --affects names features by their full IDs\n" +
			"(features/…). When a scenario already promises the expected behavior, cite\n" +
			"it under `# Violates`: the repair is a Fix (`fdf fix --from bugs/<id> …`).\n" +
			"When none does, the repair is a Change that decides it (`fdf change --from\n" +
			"bugs/<id> …`). A bug is never repaired in place: the Fix or Change that\n" +
			"repairs it names it in `resolves`. A bug that needs no repair (not a\n" +
			"defect, or a duplicate) is resolved with a `# Resolution` saying why.\n" +
			"\n" +
			"--cleanup clears resolved bugs as `fdf debt --cleanup` does, but always\n" +
			"logs them: F10 checks a done Fix or Change against bugs/LOG.md.",
		flags: []flagDoc{
			{"--open", "list only open bugs (also --accepted, --resolved)"},
			{"--cleanup", "clear resolved bugs, recording each in bugs/LOG.md"},
			{"--dry-run", "with --cleanup: show what would be cleared, change nothing"},
			{"--affects <ids>", "when filing: comma-separated feature IDs the defect shows up in"},
			{"--resource <paths>", "when filing: comma-separated paths carrying it"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf bug --open",
			"fdf bug --affects features/payments/instant-refunds --resource internal/payments/refund.go refund-split-capture",
			"fdf bug --cleanup --dry-run",
		},
	},
	{
		name:    "debt",
		group:   "Registers and practices",
		summary: "List the debt register (known gaps), or file a debt",
		usage:   "fdf debt [--root <dir>] [--open|--accepted|--resolved] [--cleanup [--dry-run] [--no-log]] [--resource <paths>] [[<group>/…]<slug>]",
		body: "Read or file the debt register under debts/: the known gaps between what\n" +
			"the project says and what the code does — work left undone, and rules the\n" +
			"codebase does not follow everywhere yet.\n" +
			"\n" +
			"Without a slug it prints the register: status, ID, filing date (UTC) and\n" +
			"title. With a slug it files a new debt, flat or in groups nested to any\n" +
			"depth, status `open` (later `accepted` or `resolved`), and lists it in the\n" +
			"index beside it. Set `resource` to the paths that carry the gap: that is\n" +
			"how later work finds the debt, and R1 fails it once those paths are gone.\n" +
			"\n" +
			"--cleanup clears resolved debts: each gets one line in debts/LOG.md, and\n" +
			"its file, its log and its index listing are removed; a group left with no\n" +
			"entry goes too, at any depth, index and listing included. Open and\n" +
			"accepted debts are never touched. --dry-run shows the plan first; --no-log\n" +
			"clears them without the LOG.md line.",
		flags: []flagDoc{
			{"--open", "list only open debts (also --accepted, --resolved)"},
			{"--cleanup", "clear resolved debts, recording each in debts/LOG.md"},
			{"--dry-run", "with --cleanup: show what would be cleared, change nothing"},
			{"--no-log", "with --cleanup: clear them without writing debts/LOG.md"},
			{"--resource <paths>", "when filing: comma-separated paths carrying the gap"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf debt",
			"fdf debt --open",
			"fdf debt --cleanup --dry-run",
			"fdf debt authz-legacy-handlers",
		},
	},
	{
		name:    "practice",
		group:   "Registers and practices",
		summary: "Write down the one way the project does a recurring mechanism",
		usage:   "fdf practice [--root <dir>] [<group>/…]<slug>",
		body: "Scaffold a Practice under practices/: the project's binding answer to\n" +
			"how one recurring mechanism is done (authorization, permission checks,\n" +
			"payment capture, database access). It is listed in practices/INDEX.md,\n" +
			"or in its group's index; groups nest to any depth, and each new one is\n" +
			"listed in its parent's. A practice is a living document with no spec,\n" +
			"plan, test or tasks; its only sibling is an optional <slug>.log.md. Fill\n" +
			"`# Rules` with the binding statements and set `applies-to` to the\n" +
			"project paths it governs: that is how later work is routed to it, since\n" +
			"features never list the practices they follow. A practice binds all\n" +
			"future code: land one only with human approval.",
		flags: []flagDoc{rootFlagDoc},
		examples: []string{
			"fdf practice permission-checks",
			"fdf practice payments/idempotency",
		},
	},
	{
		name:    "validate",
		group:   "Check and maintain",
		summary: "Check the bundle against its pinned spec (F1-F14, R1)",
		usage:   "fdf validate [--root <dir>] [--repo-root <dir>] [--strict-domain]",
		body: "Check the bundle against the spec version pinned in its root INDEX.md.\n" +
			"A bundle that pins none, or one this fdf does not validate, fails F1 and\n" +
			"is checked no further: `fdf migrate` upgrades a bundle from before 1.0.\n" +
			"Every error names its rule: F1-F14 for the format, R1 for paths that must\n" +
			"exist in the project. Warnings never fail the run: soft checks, and F12's\n" +
			"banned words unless strict domain mode is on (`strict: true` in DOMAIN.md,\n" +
			"or --strict-domain). Exit 0 means conformant. Run it after every bundle\n" +
			"edit; the fdf skills rely on it.",
		flags: []flagDoc{
			{"--strict-domain", "strict domain mode for this run: F12 banned words are errors"},
			{"--repo-root <dir>", "project root for R1's path checks (default: found from the bundle)"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf validate",
			"fdf validate --root wiki/fdf",
			"fdf validate --strict-domain",
			"FDF_ROOT_DIR=wiki/fdf fdf validate",
		},
	},
	{
		name:    "log",
		group:   "Check and maintain",
		summary: "Add an entry to the log it belongs in: fdf log [<id>] \"<entry>\"",
		usage:   "fdf log [--root <dir>] [<id>] \"<entry>\"",
		body: "Add one entry to the log it belongs in, under today's `## YYYY-MM-DD`\n" +
			"heading, newest first; the date is UTC, as every date fdf writes is. An\n" +
			"entry goes in the log of the one document it is about: a feature, Change,\n" +
			"Fix, practice, debt or bug ID puts it in that document's <slug>.log.md,\n" +
			"created on first use, and a task or trail document's entry goes in its\n" +
			"owner's log. A group ID puts it in the group's LOG.md, at any depth. With\n" +
			"no ID, or for a Context document or a release, it goes in the bundle-root\n" +
			"LOG.md, which is for the bundle as a whole. A log is the one sibling a\n" +
			"draft feature may have. Start an entry with a bold label naming the event\n" +
			"(**Specified**, **Decision**, **Done**), then say what happened and, for a\n" +
			"decision, why. fdf adds the list bullet itself; an entry that starts with\n" +
			"\"-\" goes after --, so it is not read as a flag: fdf log -- \"- …\".",
		flags: []flagDoc{rootFlagDoc},
		examples: []string{
			"fdf log features/payments/instant-refunds \"**Specified**: design approved.\"",
			"fdf log changes/refund-window \"**Decision**: refunds in flight keep the old window.\"",
			"fdf log \"**Checkpoint**: Context documents re-read against the code.\"",
		},
	},
	{
		name:    "mv",
		group:   "Check and maintain",
		summary: "Move or rename a document and repair every reference to it",
		usage:   "fdf mv [--root <dir>] [--dry-run] <from-id> <to-id>",
		body: "Move or rename a document with everything it owns — a feature with its\n" +
			"spec, plan, test, surface, log and task directory; a Change or Fix; a\n" +
			"practice, debt or bug; a task within its directory; or a whole group —\n" +
			"within its register, at any depth: flat to grouped and back, into a nested\n" +
			"group, or a group under another. It takes full IDs (features/…), and\n" +
			"repairs every reference to what moved across the bundle, frozen documents\n" +
			"included: links, `affects`, `depends-on`, `replaced-by`, `retires`,\n" +
			"`superseded-by`, `resolves`, declaration headings, index listings and ID\n" +
			"mentions. Logs keep their words and get their links repaired. It never\n" +
			"overwrites, and it lists references outside the bundle without editing\n" +
			"them. A debt and a bug can be re-filed as each other; a debt re-filed as a\n" +
			"bug still needs the `# Expected` a bug states. The move is logged in\n" +
			"LOG.md, then the bundle is validated, and the exit code is validation's.",
		flags: []flagDoc{
			{"--dry-run", "print what would move and what would be repaired, change nothing"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf mv features/payments/store-hours features/venues/opening-hours",
			"fdf mv features/onboarding features/accounts/onboarding",
			"fdf mv --dry-run features/payments features/billing",
			"fdf mv debts/refund-webhook-retries bugs/refund-webhook-retries",
			"fdf mv features/payments/instant-refunds/02-ui features/payments/instant-refunds/03-ui",
		},
	},
	{
		name:    "lexicon",
		group:   "Check and maintain",
		summary: "List banned domain words (F12), or replace them with their term",
		usage:   "fdf lexicon [--root <dir>] [--term <Term>] [--all] [--fix [--dry-run]]",
		body: "Report every banned word F12 sees — file:line:col and the line around it,\n" +
			"grouped by word — and every name using one, with a suggested `fdf mv`.\n" +
			"Triage first: a word used in another sense is qualified and listed under\n" +
			"the term's `except:`; a mention of a word goes in a code span. Then --fix\n" +
			"replaces the rest with the term, one term at a time: plurals, capitals and\n" +
			"a/an are kept right, and a scenario name is renamed everywhere it is a\n" +
			"join, slug.test.md included. Italic mentions, labels quoted in Gherkin\n" +
			"steps, table cells and names are left for a person and listed. The sweep\n" +
			"is logged in LOG.md and validated.",
		flags: []flagDoc{
			{"--term <Term>", "only that term's banned words (required with --fix)"},
			{"--all", "list every occurrence, not the first few per word"},
			{"--fix", "replace them with the term (a lexicon fix)"},
			{"--dry-run", "with --fix: print the diff, change nothing"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf lexicon",
			"fdf lexicon --term Venue --all",
			"fdf lexicon --term Venue --fix --dry-run",
			"fdf lexicon --term Venue --fix",
		},
	},
	{
		name:    "release",
		group:   "Check and maintain",
		summary: "Write releases/<version>.md from the documents' version: fields",
		usage:   "fdf release [--root <dir>] [--date <YYYY-MM-DD>] [--ship] <version>",
		body: "Create or refresh releases/<version>.md, listing the features, Changes and\n" +
			"Fixes whose `version:` is <version>. The lists are derived, never\n" +
			"hand-written; `# Notes` is yours and is kept as it is. fdf never decides\n" +
			"what ships: set `version:` on each document that does. The first release\n" +
			"creates releases/INDEX.md and lists it in the root INDEX.md. --ship marks\n" +
			"the release shipped, and refuses while any listed document is not `done`.",
		flags: []flagDoc{
			{"--date <YYYY-MM-DD>", "target date while planned, actual date once shipped"},
			{"--ship", "mark the release shipped once every listed document is done"},
			rootFlagDoc,
		},
		examples: []string{
			"fdf release 1.2.0",
			"fdf release --date 2026-10-01 1.2.0",
			"fdf release --ship 1.2.0",
		},
	},
	{
		name:    "spec",
		group:   "Reference",
		summary: "Print the format spec: fdf spec [-v <version>]",
		usage:   "fdf spec [-v <version>] [--list]",
		body: "Print the format specification, straight from the copy embedded in this\n" +
			"binary — no bundle, no network, and no checkout of the fdf repository\n" +
			"needed. Defaults to the current version; -v prints another. The 0.x\n" +
			"versions stay embedded, to read the copy a bundle from before 1.0\n" +
			"vendored.",
		flags: []flagDoc{
			{"-v <version>", "spec version to print (default: the current version)"},
			{"--list", "list the spec versions embedded in this binary"},
		},
		examples: []string{"fdf spec", "fdf spec -v 0.7", "fdf spec --list", "fdf spec | less"},
	},
	{
		name:    "serve",
		group:   "Reference",
		summary: "Browse the bundle in a web browser (needs bun)",
		usage:   "fdf serve [--root <dir>]",
		body: "Serve the bundle in a browser by wrapping `bun x mdts`, so the markdown\n" +
			"renders with working cross-links. Requires bun (https://bun.sh).",
		flags:    []flagDoc{rootFlagDoc},
		examples: []string{"fdf serve", "fdf serve --root wiki/fdf"},
	},
	{
		name:    "help",
		group:   "Reference",
		summary: "Every command in full, with examples: fdf help [<command>]",
		usage:   "fdf help [<command>]",
		body: "Print this help. With a command's name, print only its entry; with a\n" +
			"skill's name, say what the skill is for. `fdf <command> --help` prints\n" +
			"the same entry.",
		examples: []string{"fdf help", "fdf help migrate", "fdf migrate --help"},
	},
	{
		name:     "version",
		group:    "Reference",
		summary:  "Print the fdf version",
		usage:    "fdf version",
		body:     "Print the fdf version; `fdf --version` does the same.",
		examples: []string{"fdf version"},
	},
}

// skills are the agent skills `fdf install` places, each with what it is
// for. People ask fdf for them by name (`fdf plan`, `fdf help fdf-plan`); the
// answer is that their AI agent runs them.
var skills = map[string]string{
	"fdf-help":       "it routes work to the right skill by the feature's status, before any code is written",
	"fdf-init":       "it fills the five Context documents through an interview",
	"fdf-adopt":      "it maps code that predates the bundle into adopted features, and backfills their scenarios",
	"fdf-brainstorm": "it turns an idea into a feature document and gets its design approved",
	"fdf-plan":       "it turns an approved spec into tasks and slug.test.md",
	"fdf-execute":    "it works a planned feature's tasks, keeping every status true",
	"fdf-change":     "it writes the Change or Fix for a delivered feature",
	"fdf-debug":      "it finds a defect's root cause before any fix, then routes the repair",
	"fdf-checkpoint": "it audits the Context documents, SPEC.md and the agent's instruction file against the code",
	"fdf-validate":   "it turns fdf validate's rule codes into the right fix",
}

func findTopic(name string) (helpTopic, bool) {
	for _, t := range helpTopics {
		if t.name == name {
			return t, true
		}
	}
	return helpTopic{}, false
}

func topicNames() []string {
	names := make([]string, 0, len(helpTopics))
	for _, t := range helpTopics {
		names = append(names, t.name)
	}
	return names
}

// commandIndex is the overview's grouped list of commands, one line each.
func commandIndex() string {
	var b strings.Builder
	for i, t := range helpTopics {
		if i == 0 || t.group != helpTopics[i-1].group {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(t.group + "\n")
		}
		fmt.Fprintf(&b, "  %-10s %s\n", t.name, t.summary)
	}
	return b.String()
}

// overview is what `fdf` prints on its own, and for -h or --help: every
// command on one line, grouped, and where the rest is.
func overview() string {
	return banner + "\n\n" +
		"Usage: fdf <command> [flags] [arguments]\n\n" +
		commandIndex() + "\n" +
		"Bundle root: --root <dir> > FDF_ROOT_DIR > docs/fdf (or docs/features, from\n" +
		"before 1.0). A relative path resolves against the project root, so fdf\n" +
		"works from any subdirectory.\n" +
		"Flags go before arguments: fdf new --root <dir> [<group>/…]<slug>.\n\n" +
		"Run 'fdf help <command>' for its flags and examples, or 'fdf help' for all."
}

// printTopic prints one command's entry, as `fdf help <command>` does.
func printTopic(w io.Writer, name string) {
	t, _ := findTopic(name)
	fmt.Fprintln(w)
	t.render(w)
}

// printUsage prints a command's usage line — its topic's, so a usage error
// and the help never disagree — and where the rest is.
func printUsage(w io.Writer, name string) {
	t, _ := findTopic(name)
	fmt.Fprintln(w, wrapUsage("usage: ", t.usage))
	fmt.Fprintf(w, "Run 'fdf help %s' for its flags and examples.\n", name)
}

func (t helpTopic) render(w io.Writer) {
	fmt.Fprintln(w, wrapUsage("  ", t.usage))
	for _, line := range strings.Split(t.body, "\n") {
		fmt.Fprintln(w, strings.TrimRight("      "+line, " "))
	}
	if len(t.flags) > 0 {
		fmt.Fprintln(w)
		col := 0
		for _, f := range t.flags {
			col = max(col, utf8.RuneCountInString(f.flag))
		}
		for _, f := range t.flags {
			for i, line := range wrap(f.text, helpWidth-len("      ")-col-2) {
				name := ""
				if i == 0 {
					name = f.flag
				}
				fmt.Fprintf(w, "      %-*s  %s\n", col, name, line)
			}
		}
	}
	if len(t.examples) > 0 {
		fmt.Fprintln(w)
		for _, e := range t.examples {
			for i, line := range commandLines(e, helpWidth-len("      $ ")) {
				if i == 0 {
					fmt.Fprintf(w, "      $ %s\n", line)
				} else {
					fmt.Fprintf(w, "        %s\n", line)
				}
			}
		}
	}
	fmt.Fprintln(w)
}

// runHelp prints the long-form help: the whole document, or one command's
// entry when named. `fdf` with no arguments prints the overview instead.
func runHelp(args []string, stdout io.Writer) int {
	fs := newFlagSet("help")
	rest, exit, ok := parseArgs(fs, args, stdout)
	if !ok {
		return exit
	}
	switch len(rest) {
	case 0:
		fmt.Fprint(stdout, helpPreamble)
		fmt.Fprint(stdout, "\nUsage: fdf <command> [flags] [arguments]\n\n"+commandIndex())
		fmt.Fprintln(stdout, "\nCOMMANDS")
		for _, t := range helpTopics {
			t.render(stdout)
		}
		return 0
	case 1:
		if _, ok := findTopic(rest[0]); ok {
			printTopic(stdout, rest[0])
			return 0
		}
		unknownCommand(stdout, "fdf help", rest[0])
		return 2
	}
	printUsage(stdout, "help")
	return 2
}

// unknownCommand explains a name that is not a command: a skill's (the agent
// runs those, not fdf), a flag written before the command, or a near miss.
// who starts the message: "fdf" or "fdf help".
func unknownCommand(w io.Writer, who, name string) {
	skill := "fdf-" + strings.TrimPrefix(name, "fdf-")
	var msg string
	if purpose, ok := skills[skill]; ok {
		msg = fmt.Sprintf("%s: %q is not a command. %s is a skill: %s. Skills are run by your AI agent, not by fdf — `fdf install <harness>` installs them; ask the agent to use %s.",
			who, name, skill, purpose, skill)
		if cmd := strings.TrimPrefix(skill, "fdf-"); cmd != name {
			if _, isCmd := findTopic(cmd); isCmd {
				msg += fmt.Sprintf(" For the %s command, see `fdf help %s`.", cmd, cmd)
			}
		}
	} else if strings.HasPrefix(name, "-") {
		msg = fmt.Sprintf("%s: %q is not a command — the command comes first, then its flags: fdf <command> [flags] [arguments]", who, name)
	} else if near := nearestCommand(name); near != "" {
		msg = fmt.Sprintf("%s: unknown command %q — did you mean %q?", who, name, near)
	} else {
		msg = fmt.Sprintf("%s: unknown command %q", who, name)
	}
	for _, line := range wrap(msg, 76) {
		fmt.Fprintln(w, line)
	}
	for _, line := range wrap("commands: "+strings.Join(topicNames(), ", "), helpWidth) {
		fmt.Fprintln(w, line)
	}
}

// nearestCommand is the command a mistyped name most likely meant: at most two
// edits away, and fewer edits than the name has letters, so that a short
// name does not "match" every other short one.
func nearestCommand(name string) string {
	best, bestD := "", 3
	for _, t := range helpTopics {
		if d := editDistance(name, t.name); d < bestD && d < utf8.RuneCountInString(name) {
			best, bestD = t.name, d
		}
	}
	return best
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// wrap breaks text at spaces into lines of at most width runes; a word longer
// than width gets a line of its own.
func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	return append(lines, line)
}

// wrapUsage breaks a usage line between its bracketed parts, indenting what
// follows under the first one, so that no line is wider than the help.
func wrapUsage(lead, usage string) string {
	parts := usageParts(usage)
	if len(parts) < 3 {
		return lead + usage
	}
	head := lead + parts[0] + " " + parts[1]
	indent := strings.Repeat(" ", utf8.RuneCountInString(head)+1)
	var lines []string
	line := head
	for _, p := range parts[2:] {
		if utf8.RuneCountInString(line)+1+utf8.RuneCountInString(p) > helpWidth {
			lines = append(lines, line)
			line = indent + p
			continue
		}
		line += " " + p
	}
	return strings.Join(append(lines, line), "\n")
}

// usageParts splits a usage line at the spaces outside brackets:
// "[--cleanup [--dry-run]]" stays one part.
func usageParts(usage string) []string {
	var parts []string
	depth, start := 0, 0
	for i, r := range usage {
		switch r {
		case '[', '<':
			depth++
		case ']', '>':
			depth--
		case ' ':
			if depth == 0 {
				parts = append(parts, usage[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, usage[start:])
}

// commandLines breaks a long example command between its words, ending each
// broken line with a shell continuation, so that it still pastes as one
// command. A quoted argument is never broken, and a flag stays with its value.
func commandLines(cmd string, width int) []string {
	var words []string
	quote, start := rune(0), 0
	for i, r := range cmd {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ':
			words = append(words, cmd[start:i])
			start = i + 1
		}
	}
	words = append(words, cmd[start:])
	var lines, cur []string
	indent := ""
	for _, word := range words {
		next := append(append([]string(nil), cur...), word)
		if len(cur) == 0 || utf8.RuneCountInString(indent+strings.Join(next, " ")) <= width-len(" \\") {
			cur = next
			continue
		}
		carry := []string{word}
		if last := cur[len(cur)-1]; len(cur) > 2 && strings.HasPrefix(last, "-") {
			cur, carry = cur[:len(cur)-1], []string{last, word}
		}
		lines = append(lines, indent+strings.Join(cur, " ")+" \\")
		indent, cur = "  ", carry
	}
	return append(lines, indent+strings.Join(cur, " "))
}

// countWord spells out a small count for prose: "ten skills".
func countWord(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
		"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen", "twenty"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprint(n)
}
