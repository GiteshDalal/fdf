package scaffold

import (
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// SpecVersions lists the embedded specs oldest first, by number rather than
// by string, so 1.10 will follow 1.2; the current version is the newest.
func TestSpecVersionsAreOrderedByNumber(t *testing.T) {
	vs := SpecVersions()
	if len(vs) == 0 || vs[len(vs)-1] != CurrentVersion() {
		t.Fatalf("SpecVersions() = %v; want every embedded version, the current one (%s) last", vs, CurrentVersion())
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
