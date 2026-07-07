// Package migrate mechanically upgrades a bundle between adjacent FDF spec
// versions. v0.1 -> v0.2: case renames, vendored-spec removal, fdf_version
// pin, link rewrites, TEST.md stubs. Ends by validating the result.
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
const currentVersion = "0.3"

var renames = map[string]string{"index.md": "INDEX.md", "log.md": "LOG.md", "spec.md": "SPEC.md", "plan.md": "PLAN.md"}
var linkRe = regexp.MustCompile(`(\]\()([^)]*)(\))`)
var pinRe = regexp.MustCompile(`(?m)^fdf_version:.*$`)
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

	// 1. Two-step case renames. All files are collected before any rename;
	// the walk's default top-down order is fine because only basenames
	// change, so every collected path stays valid throughout.
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

	// 3. Upgrade the root pin to the current version (migration chains
	// forward: a v0.1 or v0.2 bundle both land on the latest pin).
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		s := string(raw)
		pin := fmt.Sprintf(`fdf_version: "%s"`, currentVersion)
		if pinRe.MatchString(s) {
			s = pinRe.ReplaceAllString(s, pin)
		} else {
			s = "---\n" + pin + "\n---\n\n" + s
		}
		os.WriteFile(idx, []byte(s), 0o644)
	}

	// 4. Rewrite links in every markdown file.
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
				if rel, err := filepath.Rel(rootAbs, joined); err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
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

	// 5. TEST.md stubs for planned+ features.
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		relPath := rel(root, p)
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) != 2 || parts[0] == "releases" || filepath.Base(p) == "INDEX.md" || filepath.Base(p) == "LOG.md" {
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
		testPath := filepath.Join(dir, "TEST.md")
		if exists(testPath) {
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

	// 6. v0.3: refresh the bundle-root spec copy to the target version (a
	// migrated bundle must not keep a stale vendored spec) and scaffold the
	// Context stubs if absent. The stubs satisfy structure; the fdf-init
	// interview fills them (F9 flags them as unfilled until it does — the
	// intended nudge).
	if code := scaffold.RefreshSpec(root, out); code != 0 {
		return code
	}
	if code := scaffold.EnsureContextStubs(root, out); code != 0 {
		return code
	}

	// 7. Validate the result. Freshly scaffolded Context stubs are advisory
	// here — migration succeeded; filling them is the human's next step.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	code := bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out, FreshStubsAdvisory: true})
	if code == 0 {
		fmt.Fprintln(out, "\nnext: run the fdf-init skill to fill STACK.md, ARCHITECTURE.md, and INFRA.md.")
	}
	return code
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}
func exists(p string) bool { _, err := os.Stat(p); return err == nil }
