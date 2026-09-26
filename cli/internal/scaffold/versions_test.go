package scaffold

import (
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	fdf "github.com/GiteshDalal/fdf"
	"github.com/GiteshDalal/fdf/cli/internal/layout"
	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// SpecVersions lists the embedded specs oldest first, by number rather than
// by string, so 1.10 will follow 1.2. The current version is among them, but
// need not be the newest embedded spec: fdf init can still pin an older
// version than one already embedded for `fdf spec` and `fdf migrate` to read.
func TestSpecVersionsAreOrderedByNumber(t *testing.T) {
	vs := SpecVersions()
	current := false
	for _, v := range vs {
		current = current || v == CurrentVersion()
	}
	if len(vs) == 0 || !current {
		t.Fatalf("SpecVersions() = %v; want every embedded version, including the current one (%s)", vs, CurrentVersion())
	}
	for i := 1; i < len(vs); i++ {
		a, okA := specver.Parse(vs[i-1])
		b, okB := specver.Parse(vs[i])
		if !okA || !okB || !a.Less(b) {
			t.Fatalf("SpecVersions() = %v; want versions only, oldest first", vs)
		}
	}
	if _, err := SpecText("README"); err == nil {
		t.Fatal("spec/README.md is not a version")
	}
}

// RequireSupported names the newest of Supported(), so it must never be
// empty: it holds the version fdf init pins.
func TestSupportedHoldsTheCurrentVersion(t *testing.T) {
	for _, v := range Supported() {
		if v == CurrentVersion() {
			return
		}
	}
	t.Fatalf("Supported() = %v; want it to hold the current version, %s", Supported(), CurrentVersion())
}

// The two indexes of the specs, the repository's SPEC.md and spec/README.md,
// name the version fdf init pins as the current one, so that a version bump
// cannot leave them pointing at the one before.
func TestTheSpecIndexesNameTheCurrentVersion(t *testing.T) {
	v := CurrentVersion()
	top, err := os.ReadFile("../../../SPEC.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := "| **" + v + "** | [spec/" + v + ".md](spec/" + v + ".md) | **Current** |"; !strings.Contains(string(top), want) {
		t.Errorf("SPEC.md does not list %s as current: want the row %q", v, want)
	}
	list, err := fs.ReadFile(fdf.Assets, "spec/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := "- [`" + v + ".md`](" + v + ".md) — current."; !strings.Contains(string(list), want) {
		t.Errorf("spec/README.md does not list %s as current: want the entry %q", v, want)
	}
}

// The current spec's *Casing* section names the Context documents and the
// registers, and layout, which the validator and every command read, knows
// the same ones: a version that adds one must teach layout too, or nothing
// would check or write it (TestEveryRegisterAndContextDocumentHasItsText, in
// migrate, then asks scaffold for its text).
func TestLayoutKnowsTheRootTheCurrentSpecNames(t *testing.T) {
	raw, err := SpecText(CurrentVersion())
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "\n# Casing\n")
	end := strings.Index(text[start+1:], "\n# ")
	if start < 0 || end < 0 {
		t.Fatalf("spec %s has no # Casing section", CurrentVersion())
	}
	casing := text[start : start+1+end]
	spans := regexp.MustCompile("`([^`]+)`")
	names := func(s string) []string {
		var out []string
		for _, m := range spans.FindAllStringSubmatch(s, -1) {
			out = append(out, m[1])
		}
		slices.Sort(out)
		return out
	}
	var context []string
	for _, line := range strings.Split(casing, "\n") {
		if strings.HasSuffix(line, "| The Context documents |") {
			context = names(strings.Split(line, "|")[1])
		}
	}
	var registers []string
	if i := strings.Index(casing, "The register names "); i >= 0 {
		if j := strings.Index(casing[i:], " are reserved"); j >= 0 {
			registers = names(casing[i : i+j])
		}
	}
	for _, c := range []struct {
		what        string
		spec, known []string
	}{
		{"Context documents", context, layout.ContextDocs},
		{"registers", registers, layout.Registers},
	} {
		known := slices.Sorted(slices.Values(c.known))
		if len(c.spec) == 0 || !slices.Equal(c.spec, known) {
			t.Errorf("spec %s's Casing names the %s %v; layout knows %v", CurrentVersion(), c.what, c.spec, known)
		}
	}
}
