// Package migrate mechanically upgrades a bundle between adjacent FDF spec
// versions. Chains forward to the current pin:
//
//	v0.1 → case renames, vendored-spec removal, link rewrites, TEST stubs
//	v0.2/v0.3 → lift nested trail to stem-qualified siblings, rewrite links
//	any → pin current, RefreshSpec, EnsureContextStubs, validate
//
// Ends by validating the result with FreshStubsAdvisory so unfilled Context
// stubs do not fail the migration (plain `fdf validate` will still enforce F9).
package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

const specURL = "https://github.com/GiteshDalal/fdf/blob/main/SPEC.md"
const currentVersion = "0.4"

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
var pinValueRe = regexp.MustCompile(`fdf_version:\s*"([^"]+)"`)
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
		fmt.Fprintf(out, "bundle already pins fdf_version %s; nothing to migrate\n", currentVersion)
		fmt.Fprintln(out, "\nvalidating bundle:")
		return bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out})
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
	moves, err := collectTrailMoves(root)
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

	// 9. Validate. Freshly scaffolded Context stubs are advisory here —
	// migration succeeded; filling them is the human's next step via fdf-init.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	code := bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out, FreshStubsAdvisory: true})
	if code == 0 {
		fmt.Fprintln(out, "\nnext: run the fdf-init skill to fill STACK.md, ARCHITECTURE.md, SURFACES.md, and INFRA.md.")
		fmt.Fprintln(out, "warning: the next plain `fdf validate` will fail F9 until those stubs are filled (migrate passes only because FreshStubsAdvisory treats freshly scaffolded stubs as warnings).")
	}
	return code
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
			cases = append(cases, fmt.Sprintf("- Scenario: %s — TODO: specify the concrete verification.", strings.TrimSpace(string(sc[1]))))
		}
		body := fmt.Sprintf("---\ntype: Test\ntitle: %s acceptance\ndescription: How this feature is proven.\ntimestamp: %s\n---\n\n# Test Cases\n\n%s\n",
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
