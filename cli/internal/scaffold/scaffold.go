// Package scaffold implements fdf init and fdf new.
package scaffold

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	fdf "github.com/GiteshDalal/fdf"
)

const currentVersion = "0.4"
const specURL = "https://github.com/GiteshDalal/fdf/blob/main/spec/" + currentVersion + ".md"

// contextDocs are the bundle-root Context documents fdf init scaffolds as
// stubs; the fdf-init interview fills them. Each stub carries the
// bundle.stubSentinel so `fdf validate` can tell filled from unfilled.
var contextDocs = []struct{ file, title, purpose, headings string }{
	{"STACK.md", "Technology Stack",
		"the languages, frameworks, runtimes, libraries, data stores, and versions this project uses",
		"## Languages & runtimes\n\n## Frameworks & libraries\n\n## Data stores\n\n## Notable dependencies\n"},
	{"ARCHITECTURE.md", "Architecture & Principles",
		"the architecture style, how code is organized, and the design principles contributors follow",
		"## Style\n\n## Code organization\n\n## Design principles\n\n## Key decisions\n"},
	{"SURFACES.md", "Surfaces & Interaction Design",
		"how this project’s surfaces present themselves and are engaged with — APIs, UIs, CLIs, events, and inputs — plus links to assets and exemplars",
		"## Purpose\n\n## Surfaces\n\n## Principles\n\n## Conventions by surface\n\n### API / machine interfaces\n\n### Human UI\n\n### CLI / operator surfaces\n\n### Inputs and processing\n\n## Assets and exemplars\n\n## Out of scope\n"},
	{"INFRA.md", "Build & Deployment Infrastructure",
		"how the project is built, tested, packaged, and deployed, and the environments and targets it runs on",
		"## Build & test\n\n## Packaging\n\n## Environments & targets\n\n## Deployment\n"},
}

// specDoc renders the embedded spec for the current version as a bundle-root
// Reference document. Agents and readers of the bundle need no external
// context to learn the format: the pinned version's spec travels with the
// bundle at /SPEC.md.
func specDoc() ([]byte, error) {
	raw, err := fs.ReadFile(fdf.Assets, "spec/"+currentVersion+".md")
	if err != nil {
		return nil, err
	}
	fm := fmt.Sprintf("---\ntype: Reference\ntitle: Feature Document Format spec\ndescription: The FDF v%s specification this bundle conforms to.\ntimestamp: %s\n---\n\n", currentVersion, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	return append([]byte(fm), raw...), nil
}

// stubSentinel marks an unfilled Context document; it MUST match
// bundle.stubSentinel so validation can distinguish filled from unfilled.
const stubSentinel = "<!-- fdf:stub -->"

// EnsureSpec, RefreshSpec, and EnsureContextStubs let other commands (e.g.
// migrate) place the bundle-root spec copy and Context stubs without
// re-implementing them. EnsureSpec writes only if absent (init); RefreshSpec
// overwrites to the current version's spec (migrate), so a bumped pin never
// leaves a stale vendored spec behind.
func EnsureSpec(root string, out io.Writer) int         { return writeSpec(root, false, out) }
func RefreshSpec(root string, out io.Writer) int        { return writeSpec(root, true, out) }
func EnsureContextStubs(root string, out io.Writer) int { return writeContextStubs(root, out) }

// writeContextStubs places the four Context stubs at the bundle root, each
// only if absent. The fdf-init interview replaces the stub bodies later.
func writeContextStubs(root string, out io.Writer) int {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	for _, c := range contextDocs {
		path := filepath.Join(root, c.file)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		body := fmt.Sprintf(`---
type: Context
title: %s
description: %s
timestamp: %s
---

%s

# %s

> %s **This is a critical context document.** It is the current snapshot of
> %s. Run the `+"`fdf-init`"+` skill to fill it through a guided
> interview, and change it afterward only with explicit human approval,
> logging each change. Accurate context here is what lets an agent do real
> engineering instead of guessing. Delete this banner once filled.

%s`, c.title, "Project "+strings.ToLower(c.title)+" — current snapshot.", now, stubSentinel, c.title, "⚠️", c.purpose, c.headings)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "wrote %s (context stub — fill via the fdf-init skill)\n", path)
	}
	return 0
}

// writeSpec places /SPEC.md. With force=false it is a no-op when a copy
// already exists (init); with force=true it overwrites to the current
// version's spec (migrate), reporting whether it wrote or refreshed.
func writeSpec(root string, force bool, out io.Writer) int {
	specPath := filepath.Join(root, "SPEC.md")
	_, statErr := os.Stat(specPath)
	existed := statErr == nil
	if existed && !force {
		return 0
	}
	doc, err := specDoc()
	if err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(specPath, doc, 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	verb := "wrote"
	if existed {
		verb = "refreshed"
	}
	fmt.Fprintf(out, "%s %s (FDF v%s spec copy)\n", verb, specPath, currentVersion)
	return 0
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)

// pinRe tolerates unquoted pins (`fdf_version: 0.4`), matching how the
// validator and migrate read them.
var pinRe = regexp.MustCompile(`fdf_version:\s*"?([^"\s]+)"?`)

func Init(root string, out io.Writer) int {
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		if m := pinRe.FindSubmatch(raw); m != nil && string(m[1]) == currentVersion {
			// Backfill the spec copy and context stubs for bundles
			// initialized before they existed.
			if code := writeSpec(root, false, out); code != 0 {
				return code
			}
			if code := writeContextStubs(root, out); code != 0 {
				return code
			}
			fmt.Fprintf(out, "bundle at %s is already initialized and up to date (fdf_version %s)\n", root, currentVersion)
			return 0
		} else if m != nil {
			fmt.Fprintf(out, "bundle at %s pins fdf_version %q; run `fdf migrate` to upgrade to %s\n", root, m[1], currentVersion)
			return 1
		}
		fmt.Fprintf(out, "bundle at %s has an INDEX.md without an fdf_version pin; run `fdf migrate`\n", root)
		return 1
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	today := time.Now().UTC().Format("2006-01-02")
	index := fmt.Sprintf("---\nfdf_version: %q\n---\n\n# Feature Bundle\n\nThis bundle conforms to [FDF v%s](/SPEC.md): features in Markdown + Gherkin,\neach with its spec/plan/test trail as stem-named siblings and tasks in a paired directory.\n\n# Overview\n\n* [FDF spec](/SPEC.md) - the format this bundle pins ([upstream](%s)).\n\n# Conventions\n\n* Feature IDs are file paths minus `.md`.\n* Validate with `fdf validate`; see [`LOG.md`](/LOG.md) for change history.\n",
		currentVersion, currentVersion, specURL)
	log := fmt.Sprintf("# Bundle Update Log\n\n## %s\n* **Initialization**: scaffolded by `fdf init` (FDF v%s).\n", today, currentVersion)
	if err := os.WriteFile(idx, []byte(index), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(root, "LOG.md"), []byte(log), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if code := writeSpec(root, false, out); code != 0 {
		return code
	}
	if code := writeContextStubs(root, out); code != 0 {
		return code
	}
	fmt.Fprintf(out, "initialized FDF bundle at %s\n", root)
	fmt.Fprintln(out, "next: run the fdf-init skill to fill STACK.md, ARCHITECTURE.md, SURFACES.md, and INFRA.md before adding features.")
	return 0
}

func New(root, id string, out io.Writer) int {
	if !idRe.MatchString(id) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	group, slug, _ := strings.Cut(id, "/")
	featurePath := filepath.Join(root, group, slug+".md")
	if _, err := os.Stat(featurePath); err == nil {
		fmt.Fprintf(out, "error: %s already exists\n", featurePath)
		return 1
	}
	if err := os.MkdirAll(filepath.Join(root, group), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	feature := fmt.Sprintf(`---
type: Feature
title: %s
description: TODO — one sentence.
status: draft
timestamp: %s
---

# Feature

`+"```gherkin"+`
Feature: %s
  As a <role>
  I want <capability>
  So that <value>
`+"```"+`

# Scenarios

`+"```gherkin"+`
Scenario: Replace me
  Given a precondition
  When something happens
  Then an observable outcome
`+"```"+`
`, title, now, title)
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	// Create or append the group index.
	gidx := filepath.Join(root, group, "INDEX.md")
	entry := fmt.Sprintf("* [%s](/%s/%s.md) - TODO. (**draft**)\n", title, group, slug)
	if raw, err := os.ReadFile(gidx); err == nil {
		if err := os.WriteFile(gidx, append(raw, []byte(entry)...), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		heading := strings.ToUpper(group[:1]) + group[1:]
		if err := os.WriteFile(gidx, fmt.Appendf(nil, "# %s features\n\n%s", heading, entry), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	} else {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created %s (status: draft)\n", featurePath)
	return 0
}
