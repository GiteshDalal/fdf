package bundle

import (
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/specver"
)

// Every supported version parses, or a typo such as "1.00" would leave that
// version's bundles unvalidated without any test noticing.
func TestSupportedVersionsParse(t *testing.T) {
	for v := range supportedVersions {
		if _, ok := specver.Parse(v); !ok {
			t.Errorf("supportedVersions holds %q, which is not a MAJOR.MINOR version", v)
		}
	}
}

// A pin this validator does not check names the fix: a newer fdf for a
// version past every supported one, such as the next minor, `fdf migrate`
// for an older one, a correction for one that is not a MAJOR.MINOR
// version, and the pin itself when there is none. A supported pin is no
// problem.
func TestPinProblemNamesTheFix(t *testing.T) {
	var newest specver.Version
	for s := range supportedVersions {
		if v, _ := specver.Parse(s); newest.Less(v) {
			newest = v
		}
		if msg := pinProblem(s); msg != "" {
			t.Errorf("pinProblem(%q) = %q; want none", s, msg)
		}
	}
	for _, tc := range []struct{ pin, want string }{
		{specver.Version{Major: newest.Major, Minor: newest.Minor + 1}.String(), "is newer than any version this fdf validates (" + supportedList() + ") — upgrade fdf (F1)"},
		{specver.Version{Major: newest.Major + 1}.String(), "— upgrade fdf (F1)"},
		{"0.7", "is not a supported version (" + supportedList() + ") — run `fdf migrate` (F1)"},
		{"0.1", "— run `fdf migrate` (F1)"},
		{"1.0.0", `is not a MAJOR.MINOR version such as "` + newestSupported() + `" — correct the pin (F1)`},
		{"v1.0", "— correct the pin (F1)"},
		{"", "INDEX.md: pins no fdf_version"},
	} {
		if msg := pinProblem(tc.pin); !strings.Contains(msg, tc.want) {
			t.Errorf("pinProblem(%q) = %q; want it to say %q", tc.pin, msg, tc.want)
		}
	}
}
