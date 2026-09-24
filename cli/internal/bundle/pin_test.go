package bundle

import "testing"

// pinAtLeast gates each rule by the spec version that introduced it. It
// compares versions by number, and a missing or unsupported pin is below
// every gate: such a bundle is validated under v0.2 rules.
func TestPinAtLeast(t *testing.T) {
	for _, tc := range []struct {
		pin   string
		minor int
		want  bool
	}{
		{"0.7", 7, true},
		{"0.7", 5, true},
		{"0.6", 7, false},
		{"0.2", 2, true},
		{"", 2, false},
		{"0.9", 2, false}, // not a version this validator supports
		{"v0.7", 2, false},
		{"0.07", 2, false},
	} {
		if got := pinAtLeast(tc.pin, tc.minor); got != tc.want {
			t.Errorf("pinAtLeast(%q, %d) = %v; want %v", tc.pin, tc.minor, got, tc.want)
		}
	}
}
