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
)

const specURL = "https://github.com/GiteshDalal/fdf/blob/main/SPEC.md"

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

	// 3. Upgrade the root pin.
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		s := string(raw)
		if pinRe.MatchString(s) {
			s = pinRe.ReplaceAllString(s, `fdf_version: "0.2"`)
		} else {
			s = "---\nfdf_version: \"0.2\"\n---\n\n" + s
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

	// 6. Validate the result.
	fmt.Fprintln(out, "\nvalidating migrated bundle:")
	return bundle.Validate(root, bundle.Options{RepoRoot: repoRoot, Out: out})
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}
func exists(p string) bool { _, err := os.Stat(p); return err == nil }
