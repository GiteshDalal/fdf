package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

// A feature's ID is its full path under features/, at any depth; a trail, a
// task or another register's document is not a feature, and one written the
// 0.7 way gets its full ID suggested. The name is read exactly: a stray
// features/Refunds.md is no features/refunds, even on a disk that ignores
// case.
func TestIsFeatureAndItsHint(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"features/onboarding.md", "features/onboarding.spec.md", "features/onboarding/01-form.md",
		"features/platform/payments/instant-refunds.md", "changes/refund-window.md", "features/Refunds.md"} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("---\ntype: Feature\n---\n"), 0o644)
	}
	for id, want := range map[string]bool{
		"features/onboarding":                        true,
		"features/platform/payments/instant-refunds": true,
		"features/onboarding.spec":                   false,
		"features/onboarding/01-form":                false,
		"features/nope":                              false,
		"features/refunds":                           false,
		"features/Refunds":                           false,
		"changes/refund-window":                      false,
		"onboarding":                                 false,
		"platform/payments/instant-refunds":          false,
	} {
		if got := IsFeature(root, id); got != want {
			t.Errorf("IsFeature(%q) = %v; want %v", id, got, want)
		}
	}
	for id, want := range map[string]string{
		"platform/payments/instant-refunds": " — did you mean features/platform/payments/instant-refunds?",
		"onboarding":                        " — did you mean features/onboarding?",
		"platform/payments/nope":            "",
		"features/nope":                     "",
	} {
		if got := FeatureHint(root, id); got != want {
			t.Errorf("FeatureHint(%q) = %q; want %q", id, got, want)
		}
	}
	// A command that takes a group's ID hints at a feature group's too.
	for id, want := range map[string]string{
		"platform/payments": " — did you mean features/platform/payments?",
		"onboarding":        " — did you mean features/onboarding?",
		"platform/nope":     "",
		"features/platform": "",
	} {
		if got := IDHint(root, id); got != want {
			t.Errorf("IDHint(%q) = %q; want %q", id, got, want)
		}
	}
}
