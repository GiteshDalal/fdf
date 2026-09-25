package scaffold

import (
	"testing"

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
