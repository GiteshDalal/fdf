package scaffold

// A bundle's pin decides which of its bundle-root directories hold one kind of
// document each, and so what a command may write where. The gates here mirror
// the validator's (bundle.pinAtLeast, and its specV5, specV6 and specV7):
// releases/ under every pin, changes/ from v0.5, practices/ and debts/ from
// v0.6, bugs/ from v0.7. Under an older pin the validator reads the same
// directory as a feature group — so a command must neither refuse a feature
// there nor file a register entry there.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// reservedSince is the spec minor version from which each bundle-root
// directory holds one kind of document. releases/ has since v0.2, the oldest
// version the validator checks, so it is reserved under every pin.
var reservedSince = map[string]int{"releases": 0, "changes": 5, "practices": 6, "debts": 6, "bugs": 7}

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

// ReservedDirs returns the bundle-root directories the bundle at root reserves
// under its pin: each holds one kind of document, and any other directory is
// a feature group.
func ReservedDirs(root string) map[string]bool {
	pin := Pin(root)
	out := map[string]bool{}
	for dir, since := range reservedSince {
		if since == 0 || pinAtLeast(pin, since) {
			out[dir] = true
		}
	}
	return out
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

// RequirePin reports whether the bundle at root pins spec 0.<minor> or later,
// and when it does not, says why the command stops and what to run instead.
// what names what the command writes ("the bug register"); why is what goes
// wrong under the older pin ("bugs/ is a feature group, and a Bug filed there
// fails validation (F3)").
func RequirePin(root string, minor int, what, why string, out io.Writer) bool {
	if PinAtLeast(root, minor) {
		return true
	}
	pinned := "no fdf_version"
	if pin := Pin(root); pin != "" {
		pinned = "fdf_version " + pin
	}
	fmt.Fprintf(out, "error: %s arrived in spec v0.%d, and this bundle pins %s: under that pin %s.\n", what, minor, pinned, why)
	fmt.Fprintf(out, "  run `fdf migrate` to bring the bundle to v%s first.\n", currentVersion)
	return false
}
