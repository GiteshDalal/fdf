package scaffold

// A bundle's pin decides whether the commands can work on it: they read and
// write spec 1.x, and send a 0.x bundle to `fdf migrate`.

import (
	"fmt"
	"io"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/fdfroot"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// Pin returns the fdf_version the bundle at root pins in its root INDEX.md's
// frontmatter, read as the validator reads it (fdfroot.Pin), or "" when there
// is none.
func Pin(root string) string { return fdfroot.PinOf(root) }

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
// root, and when they cannot, says why and what to do: a bundle that pins a
// 0.x version FDF had is upgraded with `fdf migrate` first, the user's
// decision (fdfroot.Upgrading), one that pins none is pinned, or upgraded,
// and one that pins a version newer than this fdf knows needs a newer fdf. A
// pin that is not a version, or is a 0.x version FDF never had, is corrected
// in INDEX.md, and so is frontmatter that never closes (fdfroot.Unclosed). A
// root whose INDEX.md pins nothing inside a pinned bundle is a register or a
// group of that bundle, which is the root to pass.
func RequireSupported(root string, out io.Writer) bool {
	pin := Pin(root)
	supported := Supported()
	for _, v := range supported {
		if pin == v {
			return true
		}
	}
	if pin == "" {
		if bundle := fdfroot.BundleAbove(root); bundle != "" {
			fmt.Fprintln(out, "error:", fdfroot.InsideBundle(root, bundle))
			return false
		}
	}
	list := strings.Join(supported, ", ")
	newest, _ := specver.Parse(supported[len(supported)-1])
	switch v, ok := specver.Parse(pin); {
	case fdfroot.Unclosed(root):
		fmt.Fprintln(out, "error: this bundle's INDEX.md frontmatter has no closing `---` line, so it pins no fdf_version — end the block with one")
	case pin == "":
		fmt.Fprintf(out, "error: this bundle's INDEX.md pins no fdf_version; fdf's commands work on spec %s bundles — pin the version it was written for, as fdf_version: \"%s\"; %s\n", list, currentVersion, fdfroot.Upgrading("a bundle from before 1.0", ""))
	case !ok:
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s, which is not a MAJOR.MINOR version such as %s — correct the pin in INDEX.md\n", pin, currentVersion)
	case newest.Less(v):
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s, newer than any spec this fdf knows (%s) — upgrade fdf\n", pin, list)
	case v.Major == 0 && !specver.Known0x(pin):
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s, which is no 0.x version this fdf knows (0.1 to 0.7) — correct the pin in INDEX.md\n", pin)
	default:
		fmt.Fprintf(out, "error: this bundle pins fdf_version %s; fdf's commands work on spec %s bundles — %s\n", pin, list, fdfroot.Upgrading("the bundle", ""))
	}
	return false
}
