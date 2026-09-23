// Package migrate mechanically upgrades a bundle between adjacent FDF spec
// versions. Chains forward to the current pin:
//
//	v0.1 → case renames, vendored-spec removal, link rewrites, TEST stubs
//	v0.2/v0.3 → lift nested trail to stem-qualified siblings, rewrite links
//	v0.4/v0.5/v0.6 → nothing structural: 0.4→0.5, 0.5→0.6 and 0.6→0.7 add
//	      documents, not moves
//	any → pin current, RefreshSpec, EnsureContextStubs, changes/, practices/,
//	      debts/ and bugs/ INDEX.md, drop index status tags, log, validate
//
// Ends by validating the result with FreshStubsAdvisory so unfilled Context
// stubs do not fail the migration (plain `fdf validate` will still enforce F9).
package migrate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/logs"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

const specURL = "https://github.com/GiteshDalal/fdf/blob/main/SPEC.md"

// currentVersion is the pin migrate writes: the one `fdf init` writes too.
var currentVersion = scaffold.CurrentVersion()

// Version is the CLI version, set by the command wrapper. A migrate that
// finds the pin already current is indistinguishable from a migrate that has
// nothing to do — unless the message names the binary doing the looking. An
// old fdf held in place by a version shim reports "already current" about a
// spec several versions behind, which reads as the command being broken.
var Version string

func binaryName() string {
	if Version == "" {
		return "this binary"
	}
	return "fdf " + Version
}

// renames: v0.1 lowercase reserved basenames → uppercase.
var renames = map[string]string{"index.md": "INDEX.md", "log.md": "LOG.md", "spec.md": "SPEC.md", "plan.md": "PLAN.md"}

// trailBasenames: nested trail files under group/slug/ → stem role suffix.
var trailBasenames = map[string]string{
	"SPEC.md": "spec",
	"PLAN.md": "plan",
	"TEST.md": "test",
	"LOG.md":  "log",
}

var linkRe = regexp.MustCompile(`(\]\()([^)]*)(\))`)
var pinLineRe = regexp.MustCompile(`(?m)^fdf_version:.*$`)

// pinValueRe tolerates unquoted pins (`fdf_version: 0.4`): the validator's
// YAML-based readPin accepts them, and migrate must agree with the validator
// about what version a bundle pins.
var pinValueRe = regexp.MustCompile(`fdf_version:\s*"?([^"\s]+)"?`)
var taskFileRe = regexp.MustCompile(`^\d{2}-[a-z0-9][a-z0-9-]*\.md$`)
var statusRe = regexp.MustCompile(`(?m)^status:\s*(\S+)`)
var scenarioRe = regexp.MustCompile(`(?m)^\s*Scenario(?: Outline)?:\s*(\S[^\n]*)`)
var timestampRe = regexp.MustCompile(`(?m)^timestamp:\s*(\S+)`)

func Run(root, repoRoot string, out io.Writer) int {
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		fmt.Fprintf(out, "error: %s is not a directory\n", root)
		return 1
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}

	pin := readPin(root)
	if pin == currentVersion {
		// Idempotent repair path: a re-run (or a hand-pinned bundle) still
		// gets the spec copy and any missing Context stubs, and validates
		// with the same stub leniency as a fresh migration — so running
		// migrate twice in a row cannot flip from success to failure.
		fmt.Fprintf(out, "nothing to migrate: the bundle already pins fdf_version %s, the newest spec %s knows.\n", currentVersion, binaryName())
		fmt.Fprintln(out, "if a newer spec version exists, upgrade fdf and re-run — a version-pinned shim (mise, asdf) can hold an older fdf in this directory.")
		fmt.Fprintln(out, "ensuring spec copy and context stubs:")
		if code := scaffold.RefreshSpec(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureContextStubs(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureChangesIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsurePracticesIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureDebtsIndex(root, out); code != 0 {
			return code
		}
		if code := scaffold.EnsureBugsIndex(root, out); code != 0 {
			return code
		}
		fmt.Fprintln(out, "\nvalidating bundle:")
		return bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out, FreshStubsAdvisory: true})
	}

	// v0.7 reserves bugs/ for the bug register. A bundle that already uses it
	// as a feature group has to move that group first; nothing else here can
	// decide its new name. Refused before anything is touched.
	if problem := bugsGroupConflict(root); problem != "" {
		fmt.Fprintln(out, "cannot migrate — fix this first (bundle left unchanged):")
		fmt.Fprintln(out, "  "+problem)
		return 1
	}

	// A bundle already in the stem-qualified layout (v0.4 onward) needs no
	// structural work: 0.4 → 0.5 only adds changes/, 0.5 → 0.6 only adds
	// practices/, debts/ and DOMAIN.md, and 0.6 → 0.7 only adds bugs/.
	// Running the pre-0.4 chain over one would be actively wrong — pre-flight
	// reads every `slug.spec.md` as an illegal dotted basename.
	stem := pin == "0.4" || pin == "0.5" || pin == "0.6"
	var moves map[string]string
	if !stem {

		// 0. Pre-flight: refuse to start on content the v0.4 layout cannot hold.
		// Nothing has been modified when this fails, so the bundle stays valid
		// under its current pin and re-running after fixes is safe.
		if problems := preflightV4(root); len(problems) > 0 {
			fmt.Fprintln(out, "cannot migrate — fix these first (bundle left unchanged):")
			for _, p := range problems {
				fmt.Fprintln(out, "  "+p)
			}
			return 1
		}

		// 1. Two-step case renames (v0.1 → uppercase). All files are collected
		// before any rename so every collected path stays valid throughout.
		var files []string
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				files = append(files, p)
			}
			return nil
		})
		renameFailed := false
		for _, p := range files {
			if to, ok := renames[filepath.Base(p)]; ok {
				tmp := p + ".migrating"
				final := filepath.Join(filepath.Dir(p), to)
				if err := os.Rename(p, tmp); err != nil {
					fmt.Fprintf(out, "error: renaming %s: %v\n", rel(root, p), err)
					renameFailed = true
					continue
				}
				if err := os.Rename(tmp, final); err != nil {
					fmt.Fprintf(out, "error: renaming %s: %v\n", rel(root, p), err)
					renameFailed = true
					continue
				}
				fmt.Fprintf(out, "renamed %s -> %s\n", rel(root, p), to)
			}
		}
		if renameFailed {
			return 1
		}

		// 2. Delete the vendored v0.1 spec.
		if vend := filepath.Join(root, "fdf-spec.md"); exists(vend) {
			os.Remove(vend)
			fmt.Fprintln(out, "removed vendored fdf-spec.md (spec is pinned by URL now)")
		}

		// 3. Rewrite casing / fdf-spec.md links in every markdown file.
		rewriteCasingLinks(root, rootAbs)

		// 4. TEST.md stubs for planned+ features (nested path; lifted in step 5).
		stubMissingTests(root, out)

		// 5. Lift nested trail files to stem-qualified siblings (0.3 → 0.4 layout).
		var err error
		moves, err = collectTrailMoves(root)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		if err := applyTrailMoves(root, moves, out); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}

		// 6. Rewrite all in-bundle links so resolved destinations stay correct
		// after the layout lift (feature → stem trail, plan → tasks, etc.).
		if len(moves) > 0 {
			rewriteLinksAfterMoves(root, rootAbs, moves)
		}

	} // end of the pre-stem layout transform

	// 7. Upgrade the root pin to the current version.
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		s := string(raw)
		pinLine := fmt.Sprintf(`fdf_version: "%s"`, currentVersion)
		if pinLineRe.MatchString(s) {
			s = pinLineRe.ReplaceAllString(s, pinLine)
		} else {
			s = "---\n" + pinLine + "\n---\n\n" + s
		}
		os.WriteFile(idx, []byte(s), 0o644)
	}

	// 8. Refresh the bundle-root spec copy and scaffold missing Context stubs
	// (including SURFACES.md on v0.4).
	if code := scaffold.RefreshSpec(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureContextStubs(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureChangesIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsurePracticesIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureDebtsIndex(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureBugsIndex(root, out); code != 0 {
		return code
	}

	// 9. Drop the status tag older tools put after an index listing
	// (` (**draft**)`). Nothing kept it current, so it went stale as soon as
	// the document moved on; a status lives only in its document.
	tags := stripIndexStatusTags(root)

	// 10. Log the migration in the bundle-root log, where every bundle-wide
	// event goes.
	from := pin
	if from == "" {
		from = "unpinned"
	}
	entry := fmt.Sprintf("**Migrated**: fdf_version %s → %s with `fdf migrate`.", from, currentVersion)
	if tags > 0 {
		entry += fmt.Sprintf(" Removed the status tag from %d index listing(s); a document's status lives only in its frontmatter.", tags)
	}
	logged := logMigration(root, entry)

	// 11. Report what actually changed, then validate. Without this the only
	// evidence of a migration is a scroll of per-file lines, and a migration
	// that moved nothing is indistinguishable from one that did.
	fmt.Fprintf(out, "\ndone: migrated bundle at %s\n", rootAbs)
	fmt.Fprintf(out, "  fdf_version %s -> %s\n", from, currentVersion)
	fmt.Fprintf(out, "  %d trail file(s) lifted to stem-qualified siblings\n", len(moves))
	if len(moves) == 0 {
		fmt.Fprintln(out, "  (no nested trail files were present — layout already matched)")
	}
	if tags > 0 {
		fmt.Fprintf(out, "  %d status tag(s) removed from index listings\n", tags)
	}
	if logged {
		fmt.Fprintln(out, "  logged in LOG.md")
	}

	// Freshly scaffolded Context stubs are advisory here — migration
	// succeeded; filling them is the human's next step via fdf-init.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	var report bytes.Buffer
	code := bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: io.MultiWriter(out, &report), FreshStubsAdvisory: true})
	if code == 0 {
		fmt.Fprintln(out, "\nnext: run the fdf-init skill to fill "+scaffold.ContextDocNames()+".")
		fmt.Fprintln(out, "warning: the next plain `fdf validate` will fail F9 until those stubs are filled (migrate passes only because FreshStubsAdvisory treats freshly scaffolded stubs as warnings).")
	}
	reportV07(root, report.String(), out)
	return code
}

// reportV07 says what v0.7 changes about a bundle that has just reached it:
// F12 now reads every document and name, and the debt register may hold
// defects that belong in the new bug register. Neither is a migration step —
// both are judgments — so the command names the tools and stops there.
func reportV07(root, validation string, out io.Writer) {
	if lex, _ := bundle.LoadLexicon(root); lex != nil {
		docs := map[string]bool{}
		occ := bundle.ScanBundle(root, lex)
		for _, o := range occ {
			docs[o.Rel] = true
		}
		if len(occ) > 0 {
			fmt.Fprintf(out, "\nv0.7: the domain language now reaches every document and name — %d banned word(s) in %d place(s).\n", len(occ), len(docs))
			fmt.Fprintln(out, "      `fdf lexicon` lists them; triage the other senses into `except:`, then sweep one term at a time")
			fmt.Fprintln(out, "      with `fdf lexicon --term <Term> --fix --dry-run` and `--fix`.")
		}
	}
	if n := strings.Count(validation, "has no test case — a case is a `## "); n > 0 {
		fmt.Fprintf(out, "\nv0.7: a test case is a `## <scenario name>` heading under `# Test Cases`, matched exactly —\n")
		fmt.Fprintf(out, "      %d scenario(s) have none (F8). Rewrite those test documents' cases as headings, by hand:\n", n)
		fmt.Fprintln(out, "      bullets and tables naming a scenario no longer count.")
	}
	if n := strings.Count(validation, "is neither a date (2026-02-14) nor an RFC 3339 time"); n > 0 {
		fmt.Fprintf(out, "\nv0.7: %d timestamp(s) are neither a date nor an RFC 3339 time with Z or an offset (F1).\n", n)
		fmt.Fprintln(out, "      Give each the Z or offset it was written in, or keep only its date.")
	}
	if n := strings.Count(validation, ".surface.md — write one"); n > 0 {
		fmt.Fprintf(out, "\nv0.7: %d feature(s) have no slug.surface.md. Write one where the feature adds or changes an interface,\n", n)
		fmt.Fprintln(out, "      or say `surface: none` in its frontmatter.")
	}
	if n := countRegisterEntries(filepath.Join(root, "debts")); n > 0 {
		fmt.Fprintf(out, "\nv0.7: of the %d debt(s) on the register, any that describes the software doing something wrong\n", n)
		fmt.Fprintln(out, "      is a bug — re-file it with `fdf mv debts/<id> bugs/<id>`, then give it the `# Expected` a bug states.")
	}
}

// statusTagRe matches an index listing that ends in the status tag `fdf new`,
// `fdf change`, `fdf fix` and `fdf adopt` wrote before v0.7, or one a person
// kept up by hand: ` (**draft**)`. Only a status word counts, so other bold
// text in parentheses stays.
var statusTagRe = regexp.MustCompile(`^([ \t]*[-*+][ \t].*\]\(.*\).*?)[ \t]*\(\*\*(?:draft|specified|planned|implementing|done|retired|adopted|pending|in-progress|active|superseded|open|accepted|resolved|shipped)\*\*\)[ \t]*(\r?)$`)

// stripIndexStatusTags removes the status tag from every listing in every
// INDEX.md, outside code, and says how many it removed.
func stripIndexStatusTags(root string) int {
	n := 0
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "INDEX.md" {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(raw), "\n")
		fence, removed := "", 0
		for i, line := range lines {
			t := strings.TrimSpace(line)
			if fence != "" {
				if strings.HasPrefix(t, fence) {
					fence = ""
				}
				continue
			}
			if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
				fence = t[:3]
				continue
			}
			if m := statusTagRe.FindStringSubmatch(line); m != nil {
				lines[i] = m[1] + m[2]
				removed++
			}
		}
		if removed > 0 && os.WriteFile(p, []byte(strings.Join(lines, "\n")), 0o644) == nil {
			n += removed
		}
		return nil
	})
	return n
}

// logMigration adds entry to the bundle-root LOG.md, creating it if missing,
// and reports whether it was written.
func logMigration(root, entry string) bool {
	p := filepath.Join(root, "LOG.md")
	body := "# Bundle Update Log\n"
	if raw, err := os.ReadFile(p); err == nil {
		body = string(raw)
	}
	return os.WriteFile(p, []byte(logs.Insert(body, logs.Entry(entry))), 0o644) == nil
}

// countRegisterEntries counts the documents in a register directory, groups
// included, leaving out its index, its log and each entry's log sibling.
func countRegisterEntries(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		if base := filepath.Base(p); base != "INDEX.md" && base != "LOG.md" && !strings.HasSuffix(base, ".log.md") {
			n++
		}
		return nil
	})
	return n
}

// bugsGroupConflict reports a bundle that uses bugs/ as a feature group, which
// v0.7 reserves for the bug register. Bug documents already there are fine.
func bugsGroupConflict(root string) string {
	var offender string
	filepath.WalkDir(filepath.Join(root, "bugs"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || offender != "" || !strings.HasSuffix(p, ".md") {
			return nil
		}
		base := filepath.Base(p)
		if base == "INDEX.md" || base == "LOG.md" || strings.HasSuffix(base, ".log.md") {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		if m := typeLineRe.FindSubmatch(raw); m == nil || strings.Trim(string(m[1]), `"'`) != "Bug" {
			offender = rel(root, p)
		}
		return nil
	})
	if offender == "" {
		return ""
	}
	return fmt.Sprintf("bugs/ is a feature group (%s), but v0.7 reserves bugs/ for the bug register — rename the group first with `fdf mv bugs <new-group>`, then re-run fdf migrate", filepath.ToSlash(offender))
}

var typeLineRe = regexp.MustCompile(`(?m)^type:\s*(\S+)`)

// preflightV4 scans for content the v0.4 layout cannot represent and that
// this migration cannot mechanically fix: dotted group-level filenames
// (v0.4 reserves the dot for trail roles), non-trail non-task files inside
// feature directories (v0.4 task dirs hold only NN-slug.md tasks), and
// draft features with a feature-dir LOG.md (lifting it would create a trail
// sibling, which v0.4 forbids on drafts). Basenames are normalized through
// the v0.1 rename map so pre-rename bundles are screened too.
func preflightV4(root string) []string {
	var problems []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		relPath := filepath.ToSlash(rel(root, p))
		parts := strings.Split(relPath, "/")
		base := filepath.Base(p)
		if to, ok := renames[base]; ok {
			base = to
		}
		switch len(parts) {
		case 2:
			if parts[0] == "releases" || base == "INDEX.md" || base == "LOG.md" {
				return nil
			}
			if strings.Contains(strings.TrimSuffix(base, ".md"), ".") {
				problems = append(problems, fmt.Sprintf("%s: filename contains a dot, which v0.4 reserves for trail roles (slug.spec.md) — rename it before migrating", relPath))
			}
		case 3:
			if _, liftable := trailBasenames[base]; liftable {
				if base == "LOG.md" && featureIsDraft(root, parts[0], parts[1]) {
					problems = append(problems, fmt.Sprintf("%s: draft features may not have trail files under v0.4 — fold this log into the root LOG.md (or advance the feature) before migrating", relPath))
				}
				return nil
			}
			if taskFileRe.MatchString(base) {
				return nil
			}
			problems = append(problems, fmt.Sprintf("%s: v0.4 task directories may contain only NN-slug.md tasks — move or remove this file before migrating", relPath))
		}
		return nil
	})
	sort.Strings(problems)
	return problems
}

// featureIsDraft reports whether the sibling feature document of a paired
// directory carries status: draft.
func featureIsDraft(root, group, slug string) bool {
	raw, err := os.ReadFile(filepath.Join(root, group, slug+".md"))
	if err != nil {
		return false
	}
	m := statusRe.FindSubmatch(raw)
	return m != nil && string(m[1]) == "draft"
}

func readPin(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		// Also try lowercase pre-rename form.
		raw, err = os.ReadFile(filepath.Join(root, "index.md"))
		if err != nil {
			return ""
		}
	}
	if m := pinValueRe.FindSubmatch(raw); m != nil {
		return string(m[1])
	}
	return ""
}

func rewriteCasingLinks(root, rootAbs string) {
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		s := linkRe.ReplaceAllStringFunc(string(raw), func(m string) string {
			parts := linkRe.FindStringSubmatch(m)
			target := parts[2]
			// Leave external and intra-document targets untouched
			// (mirrors bundle.resolveLink's guard).
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") ||
				strings.HasPrefix(target, "tel:") || strings.HasPrefix(target, "#") {
				return m
			}
			// Root-absolute targets keep their leading "/" and get only
			// the relative remainder rewritten — never "//".
			prefix, relTarget := "", target
			if strings.HasPrefix(target, "/") {
				prefix, relTarget = "/", strings.TrimPrefix(target, "/")
			} else {
				// Relative targets that resolve outside the bundle root
				// point at a sibling (non-FDF) tree we don't own —
				// leave them exactly as written.
				pathPart := target
				if i := strings.IndexAny(pathPart, "#?"); i >= 0 {
					pathPart = pathPart[:i]
				}
				joined := filepath.Clean(filepath.Join(filepath.Dir(p), pathPart))
				if relP, err := filepath.Rel(rootAbs, joined); err != nil || relP == ".." || strings.HasPrefix(relP, "../") {
					return m
				}
			}
			dir, base := filepath.Dir(relTarget), filepath.Base(relTarget)
			frag := ""
			if i := strings.IndexAny(base, "#?"); i >= 0 {
				base, frag = base[:i], base[i:]
			}
			if base == "fdf-spec.md" {
				return parts[1] + specURL + parts[3]
			}
			if to, ok := renames[base]; ok {
				if dir == "." {
					return parts[1] + prefix + to + frag + parts[3]
				}
				return parts[1] + prefix + dir + "/" + to + frag + parts[3]
			}
			return m
		})
		os.WriteFile(p, []byte(s), 0o644)
		return nil
	})
}

func stubMissingTests(root string, out io.Writer) {
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		relPath := rel(root, p)
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) != 2 || parts[0] == "releases" || filepath.Base(p) == "INDEX.md" || filepath.Base(p) == "LOG.md" {
			return nil
		}
		// Skip stem trail files if somehow already present.
		base := filepath.Base(p)
		if strings.Contains(strings.TrimSuffix(base, ".md"), ".") {
			return nil
		}
		raw, _ := os.ReadFile(p)
		m := statusRe.FindSubmatch(raw)
		if m == nil {
			return nil
		}
		status := string(m[1])
		if status != "planned" && status != "implementing" && status != "done" {
			return nil
		}
		dir := strings.TrimSuffix(p, ".md")
		// Prefer nested TEST.md (pre-lift); also skip if stem test already exists.
		testPath := filepath.Join(dir, "TEST.md")
		stemTest := filepath.Join(filepath.Dir(p), strings.TrimSuffix(base, ".md")+".test.md")
		if exists(testPath) || exists(stemTest) {
			return nil
		}
		ts := "2026-01-01T00:00:00Z"
		if tm := timestampRe.FindSubmatch(raw); tm != nil {
			ts = string(tm[1])
		}
		var cases []string
		for _, sc := range scenarioRe.FindAllSubmatch(raw, -1) {
			// One `## <scenario name>` heading per case: the form F8 matches.
			cases = append(cases, fmt.Sprintf("## %s\n\nTODO: specify the concrete verification.\n", strings.TrimSpace(string(sc[1]))))
		}
		body := fmt.Sprintf("---\ntype: Test\ntitle: %s acceptance\ndescription: How this feature is proven.\ntimestamp: %s\n---\n\n# Test Cases\n\n%s",
			strings.TrimSuffix(parts[1], ".md"), ts, strings.Join(cases, "\n"))
		os.MkdirAll(dir, 0o755)
		os.WriteFile(testPath, []byte(body), 0o644)
		fmt.Fprintf(out, "stubbed %s/TEST.md (%d scenario case(s))\n", rel(root, dir), len(cases))
		return nil
	})
}

// collectTrailMoves finds group/slug/{SPEC,PLAN,TEST,LOG}.md → group/slug.<role>.md.
func collectTrailMoves(root string) (map[string]string, error) {
	moves := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		relPath := filepath.ToSlash(rel(root, p))
		parts := strings.Split(relPath, "/")
		if len(parts) != 3 {
			return nil
		}
		role, ok := trailBasenames[parts[2]]
		if !ok {
			return nil
		}
		// parts[0]=group, parts[1]=slug, parts[2]=TRAIL.md
		destRel := parts[0] + "/" + parts[1] + "." + role + ".md"
		destAbs := filepath.Join(root, filepath.FromSlash(destRel))
		if exists(destAbs) {
			return fmt.Errorf("cannot move %s: %s already exists", relPath, destRel)
		}
		moves[relPath] = destRel
		return nil
	})
	return moves, err
}

func applyTrailMoves(root string, moves map[string]string, out io.Writer) error {
	// Deterministic order not required; each source is unique.
	for fromRel, toRel := range moves {
		from := filepath.Join(root, filepath.FromSlash(fromRel))
		to := filepath.Join(root, filepath.FromSlash(toRel))
		tmp := from + ".migrating"
		if err := os.Rename(from, tmp); err != nil {
			return fmt.Errorf("moving %s: %w", fromRel, err)
		}
		if err := os.Rename(tmp, to); err != nil {
			// Best-effort rollback of the temp name.
			_ = os.Rename(tmp, from)
			return fmt.Errorf("moving %s -> %s: %w", fromRel, toRel, err)
		}
		// Feature-dir LOG.md was reserved (no frontmatter required) under
		// v0.2/v0.3; slug.log.md is type: Log and needs a frontmatter block.
		if strings.HasSuffix(toRel, ".log.md") {
			if err := ensureFeatureLogFrontmatter(to, toRel); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "moved %s -> %s\n", fromRel, toRel)
	}
	return nil
}

// ensureFeatureLogFrontmatter wraps a bare feature log (common pre-v0.4
// form) with type: Log frontmatter so validation accepts the stem file.
func ensureFeatureLogFrontmatter(path, toRel string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", toRel, err)
	}
	text := strings.TrimPrefix(string(raw), "\uFEFF")
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "---") {
		// Already has a frontmatter fence; leave body to the author/validate.
		return nil
	}
	// Derive a short title from the stem: group/slug.log.md → slug.
	base := filepath.Base(toRel) // slug.log.md
	stem := strings.TrimSuffix(base, ".log.md")
	title := stem + " feature log"
	body := fmt.Sprintf("---\ntype: Log\ntitle: %s\ndescription: Per-feature history.\ntimestamp: 2026-01-01T00:00:00Z\n---\n\n%s", title, text)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// rewriteLinksAfterMoves rewrites every in-bundle markdown link so that
// destinations that were lifted keep resolving, and relative links from
// moved files are recomputed from their new location.
//
// Algorithm: for a link in file currently at curRel, resolve the target as if
// the source were still at its pre-move path (inverseMoves), map the resolved
// path through moves, then emit a link from the current path to the final dest.
func rewriteLinksAfterMoves(root, rootAbs string, moves map[string]string) {
	inverse := map[string]string{} // newRel -> oldRel
	for oldRel, newRel := range moves {
		inverse[newRel] = oldRel
	}

	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		curRel := filepath.ToSlash(rel(root, p))
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		// Resolve relative links as if still at the pre-move source path.
		srcForResolve := curRel
		if old, ok := inverse[curRel]; ok {
			srcForResolve = old
		}
		srcDirForResolve := filepath.ToSlash(filepath.Dir(filepath.FromSlash(srcForResolve)))
		if srcDirForResolve == "." {
			srcDirForResolve = ""
		}

		changed := false
		s := linkRe.ReplaceAllStringFunc(string(raw), func(m string) string {
			parts := linkRe.FindStringSubmatch(m)
			target := parts[2]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") ||
				strings.HasPrefix(target, "tel:") || strings.HasPrefix(target, "#") {
				return m
			}

			pathPart, frag := target, ""
			if i := strings.IndexAny(pathPart, "#?"); i >= 0 {
				pathPart, frag = pathPart[:i], pathPart[i:]
			}
			if pathPart == "" {
				return m
			}

			absStyle := strings.HasPrefix(pathPart, "/")
			var resolved string
			if absStyle {
				resolved = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(pathPart, "/")))
			} else {
				// Resolve against pre-move source directory.
				base := srcDirForResolve
				if base == "" {
					resolved = filepath.ToSlash(filepath.Clean(pathPart))
				} else {
					resolved = filepath.ToSlash(filepath.Clean(base + "/" + pathPart))
				}
				// Out-of-bundle relative targets: leave untouched.
				if resolved == ".." || strings.HasPrefix(resolved, "../") {
					return m
				}
				// Also guard via rootAbs for ".." segments that clean oddly.
				joined := filepath.Clean(filepath.Join(rootAbs, filepath.FromSlash(resolved)))
				if relP, err := filepath.Rel(rootAbs, joined); err != nil || relP == ".." || strings.HasPrefix(relP, "../") {
					return m
				}
			}

			final := resolved
			if to, ok := moves[resolved]; ok {
				final = to
			}

			// If nothing changed and the source file itself wasn't moved,
			// keep the original spelling (preserves hand-written style).
			if final == resolved && inverse[curRel] == "" {
				return m
			}

			var newTarget string
			if absStyle {
				newTarget = "/" + final + frag
			} else {
				// Relative from the current file's directory to final.
				curDir := filepath.Dir(p)
				destAbs := filepath.Join(rootAbs, filepath.FromSlash(final))
				relT, err := filepath.Rel(curDir, destAbs)
				if err != nil {
					newTarget = final + frag
				} else {
					newTarget = filepath.ToSlash(relT) + frag
				}
			}
			if newTarget == target {
				return m
			}
			changed = true
			return parts[1] + newTarget + parts[3]
		})
		if changed {
			os.WriteFile(p, []byte(s), 0o644)
		}
		return nil
	})
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }
