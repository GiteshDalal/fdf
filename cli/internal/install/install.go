// Package install places the FDF skills into an AI harness's configuration.
// Installs are idempotent: an existing install at the current version is
// reported "up to date"; any other version is auto-upgraded in place.
package install

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	fdf "github.com/GiteshDalal/fdf"
)

// Version is stamped by the CLI (main.version) at dispatch time.
var Version = "0.2.0-dev"

var skillNames = []string{"fdf-brainstorm", "fdf-plan", "fdf-execute"}
var blockRe = regexp.MustCompile(`(?s)<!-- fdf:begin v[^>]*-->.*?<!-- fdf:end -->\n?`)

func Run(harness, home string, out io.Writer) int {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	switch harness {
	case "claude-code":
		return claudeCode(home, out)
	case "codex":
		return managedBlock(filepath.Join(home, ".codex", "AGENTS.md"), out)
	case "opencode":
		return managedBlock(filepath.Join(home, ".config", "opencode", "AGENTS.md"), out)
	default:
		fmt.Fprintf(out, "unknown harness %q\n\nusage: fdf install <claude-code|codex|opencode>\n", harness)
		return 2
	}
}

func claudeCode(home string, out io.Writer) int {
	upToDate := true
	for _, name := range skillNames {
		marker := filepath.Join(home, ".claude", "skills", name, ".fdf-version")
		if v, err := os.ReadFile(marker); err != nil || string(v) != Version {
			upToDate = false
		}
	}
	if upToDate {
		fmt.Fprintf(out, "fdf skills for claude-code are up to date (v%s)\n", Version)
		return 0
	}
	hadAny := false
	for _, name := range skillNames {
		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", name)); err == nil {
			hadAny = true
		}
		src := "skills/" + name + "/SKILL.md"
		raw, err := fs.ReadFile(fdf.Assets, src)
		if err != nil {
			fmt.Fprintf(out, "error: embedded %s: %v\n", src, err)
			return 1
		}
		dir := filepath.Join(home, ".claude", "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), raw, 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
		// The marker is written only after SKILL.md landed, so a failed
		// install can never masquerade as "up to date" on the next run.
		if err := os.WriteFile(filepath.Join(dir, ".fdf-version"), []byte(Version), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	}
	cmds, err := fs.ReadDir(fdf.Assets, "harness/claude-code/commands")
	if err != nil {
		fmt.Fprintf(out, "error: embedded harness/claude-code/commands: %v\n", err)
		return 1
	}
	cmdDir := filepath.Join(home, ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	for _, e := range cmds {
		raw, err := fs.ReadFile(fdf.Assets, "harness/claude-code/commands/"+e.Name())
		if err != nil {
			fmt.Fprintf(out, "error: embedded command %s: %v\n", e.Name(), err)
			return 1
		}
		if err := os.WriteFile(filepath.Join(cmdDir, e.Name()), raw, 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	}
	verb := "installed"
	if hadAny {
		verb = "upgraded"
	}
	fmt.Fprintf(out, "%s fdf skills for claude-code (v%s): %s + %d command(s)\n",
		verb, Version, strings.Join(skillNames, ", "), len(cmds))
	return 0
}

func managedBlock(path string, out io.Writer) int {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- fdf:begin v%s (managed by `fdf install` — do not edit) -->\n", Version)
	b.WriteString("# FDF workflow\n\nThis machine uses FDF (https://github.com/GiteshDalal/fdf) for feature\ndocumentation. Bundle at docs/features (or FDF_ROOT_DIR). Validate with\n`fdf validate`; scaffold with `fdf new <group>/<slug>`.\n\n")
	for _, name := range skillNames {
		raw, err := fs.ReadFile(fdf.Assets, "skills/"+name+"/SKILL.md")
		if err != nil {
			fmt.Fprintf(out, "error: embedded skill %s: %v\n", name, err)
			return 1
		}
		body := string(raw)
		if i := strings.Index(body, "\n---\n"); i >= 0 && strings.HasPrefix(body, "---\n") {
			body = body[i+5:] // strip frontmatter for instruction-file consumers
		}
		b.WriteString(body)
		b.WriteString("\n")
	}
	b.WriteString("<!-- fdf:end -->\n")

	block := b.String()
	existing, _ := os.ReadFile(path)
	var content string
	verb := "installed"
	if blockRe.MatchString(string(existing)) {
		// Replace the existing block in place so user content before and
		// after it keeps its position. ReplaceAllStringFunc avoids `$`
		// expansion in the replacement text; any duplicate blocks collapse
		// into the first.
		verb = "upgraded"
		replaced := false
		content = blockRe.ReplaceAllStringFunc(string(existing), func(string) string {
			if replaced {
				return ""
			}
			replaced = true
			return block
		})
	} else {
		content = string(existing)
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += block
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "%s fdf instructions block in %s (v%s)\n", verb, path, Version)
	return 0
}
