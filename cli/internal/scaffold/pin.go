package scaffold

// A bundle's pin decides whether the commands can work on it: they read and
// write spec 1.x, and send a 0.x bundle to `fdf migrate`.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

var pinKeyRe = regexp.MustCompile(`^fdf_version:\s?(.*)$`)

// Pin returns the fdf_version the bundle at root pins in its root INDEX.md's
// frontmatter, read as the validator reads it, or "" when there is none.
func Pin(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimPrefix(string(raw), "\uFEFF"), "\r\n", "\n"), "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) != "---" {
			continue
		}
		// A delimited block: the pin is its fdf_version key, unquoted.
		for _, l := range lines[1 : i+1] {
			if m := pinKeyRe.FindStringSubmatch(l); m != nil {
				return strings.Trim(strings.Trim(strings.TrimSpace(m[1]), `"`), `'`)
			}
		}
		return ""
	}
	return ""
}

// PinAtLeast reports whether the bundle at root pins spec 0.<minor> or later.
// A missing pin, or one this fdf does not support, is below every gate: the
// validator then checks the bundle under v0.2 rules.
func PinAtLeast(root string, minor int) bool { return pinAtLeast(Pin(root), minor) }

// pinAtLeast is PinAtLeast for a pin already read. A supported pin is one
// whose spec this binary embeds, which are the versions its validator checks.
func pinAtLeast(pin string, minor int) bool {
	supported := false
	for _, v := range SpecVersions() {
		supported = supported || v == pin
	}
	v, ok := specver.Parse(pin)
	return supported && ok && v.AtLeast(specver.Version{Major: 0, Minor: minor})
}

// Supported lists the spec versions the commands work on: the embedded ones of
// the current major version, oldest first. A minor version only adds, so the
// commands read a bundle pinned to any of them. They must not write into one
// what a later minor adds, which is an error there (design §4): a command
// that writes something 1.x added checks the bundle's pin first.
func Supported() []string {
	cur, _ := specver.Parse(currentVersion)
	var out []string
	for _, v := range SpecVersions() {
		if p, _ := specver.Parse(v); p.Major == cur.Major {
			out = append(out, v)
		}
	}
	return out
}

// RequireSupported reports whether the commands can work on the bundle at
// root, and when they cannot, says why and what to run: a bundle that pins a
// 0.x version, or none, is upgraded with `fdf migrate` first, and one that
// pins a version newer than this fdf knows needs a newer fdf.
func RequireSupported(root string, out io.Writer) bool {
	pin := Pin(root)
	supported := Supported()
	for _, v := range supported {
		if pin == v {
			return true
		}
	}
	list := strings.Join(supported, ", ")
	newest, _ := specver.Parse(supported[len(supported)-1])
	switch v, ok := specver.Parse(pin); {
	case pin == "":
		fmt.Fprintf(out, "error: this bundle's INDEX.md pins no fdf_version; fdf's commands work on spec %s bundles — run `fdf migrate` to upgrade it first\n", list)
	case ok && newest.Less(v):
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s, newer than any spec this fdf knows (%s) — upgrade fdf\n", pin, list)
	default:
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s; fdf's commands work on spec %s bundles — run `fdf migrate` to upgrade it first\n", pin, list)
	}
	return false
}
