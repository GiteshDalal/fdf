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
// version past every supported one, such as the next minor, the upgrade,
// the user's decision, for an older one, a correction for one that is not a
// MAJOR.MINOR version or is a 0.x version FDF never had, and the pin itself
// when there is none. A supported pin is no problem.
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
		{"0.7", "is not a supported version (" + supportedList() + ") — upgrading the bundle is the user's decision, since `fdf migrate` moves its documents and rewrites references to them across the project: `fdf migrate --dry-run` shows the plan (F1)"},
		{"0.1", "— upgrading the bundle is the user's decision"},
		{"0.8", `fdf_version "0.8" is no 0.x version this fdf knows (0.1 to 0.7) — correct the pin (F1)`},
		{"0.0", "— correct the pin (F1)"},
		{"1.0.0", `is not a MAJOR.MINOR version such as "` + newestSupported() + `" — correct the pin (F1)`},
		{"v1.0", "— correct the pin (F1)"},
		{"", "INDEX.md: pins no fdf_version — the version of the spec a bundle follows is pinned in its frontmatter, as fdf_version: \"" + newestSupported() + "\"; upgrading a bundle from before 1.0 is the user's decision"},
	} {
		if msg := pinProblem(tc.pin); !strings.Contains(msg, tc.want) {
			t.Errorf("pinProblem(%q) = %q; want it to say %q", tc.pin, msg, tc.want)
		}
	}
}
