package bundle

import (
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// pinAtLeast gates each rule by the spec version that introduced it. It
// compares versions by number, and a missing or unsupported pin is below
// every gate: such a bundle is validated under v0.2 rules.
func TestPinAtLeast(t *testing.T) {
	v := func(major, minor int) specver.Version { return specver.Version{Major: major, Minor: minor} }
	for _, tc := range []struct {
		pin  string
		gate specver.Version
		want bool
	}{
		{"0.7", v(0, 7), true},
		{"0.7", v(0, 5), true},
		{"0.6", v(0, 7), false},
		{"0.2", v(0, 2), true},
		{"0.7", v(1, 0), false},
		{"", v(0, 2), false},
		{"0.9", v(0, 2), false}, // not a version this validator supports
		{"v0.7", v(0, 2), false},
		{"0.07", v(0, 2), false},
	} {
		if got := pinAtLeast(tc.pin, tc.gate); got != tc.want {
			t.Errorf("pinAtLeast(%q, %v) = %v; want %v", tc.pin, tc.gate, got, tc.want)
		}
	}
}

// Every supported version parses, or a typo such as "1.00" would leave that
// version's bundles below every gate without any test noticing.
func TestSupportedVersionsParse(t *testing.T) {
	for v := range supportedVersions {
		if _, ok := specver.Parse(v); !ok {
			t.Errorf("supportedVersions holds %q, which is not a MAJOR.MINOR version", v)
		}
	}
}

// A pin this validator does not check names the fix: a newer fdf for a
// version past every supported one, such as the next minor, and `fdf migrate`
// for any other.
func TestUnsupportedPinNamesTheFix(t *testing.T) {
	var newest specver.Version
	for s := range supportedVersions {
		if v, _ := specver.Parse(s); newest.Less(v) {
			newest = v
		}
	}
	next := specver.Version{Major: newest.Major, Minor: newest.Minor + 1}.String()
	if msg := unsupportedPin(next); !strings.Contains(msg, "is newer than any version this fdf validates") || !strings.HasSuffix(msg, "upgrade fdf (F1)") {
		t.Errorf("unsupportedPin(%q) = %q; want it to ask for a newer fdf", next, msg)
	}
	for _, pin := range []string{"0.1", "v1.0", ""} {
		if msg := unsupportedPin(pin); !strings.HasSuffix(msg, "run `fdf migrate` (F1)") {
			t.Errorf("unsupportedPin(%q) = %q; want it to point at fdf migrate", pin, msg)
		}
	}
}
