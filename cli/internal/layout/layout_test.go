package layout

import (
	"strings"
	"testing"
	"testing/fstest"
)

// sample is a 1.0 bundle with every kind of position, and every mistake the
// rules name.
var sample = New(fstest.MapFS{
	"INDEX.md":        {},
	"LOG.md":          {},
	"SPEC.md":         {},
	"README.md":       {},
	"STACK.md":        {},
	"notes.md":        {},
	"drafts/idea.md":  {},
	"assets/logo.png": {},

	"features/INDEX.md":                                          {},
	"features/onboarding.md":                                     {},
	"features/onboarding.spec.md":                                {},
	"features/onboarding/01-signup-form.md":                      {},
	"features/onboarding/INDEX.md":                               {},
	"features/onboarding/sub/02-x.md":                            {},
	"features/platform/LOG.md":                                   {},
	"features/platform/payments/instant-refunds.md":              {},
	"features/platform/payments/instant-refunds.test.md":         {},
	"features/platform/payments/instant-refunds.notes.md":        {},
	"features/platform/payments/instant-refunds/02-refund-ui.md": {},
	"features/platform/payments/Refunds.md":                      {},
	"features/bugs/triage.md":                                    {},
	"features/index/overview.md":                                 {},
	"features/Payments/x.md":                                     {},

	"changes/payments/refund-window.md":            {},
	"changes/payments/refund-window.plan.md":       {},
	"changes/payments/refund-window.test.md":       {},
	"changes/payments/refund-window/01-shorten.md": {},

	"practices/auth/permission-checks.md":          {},
	"practices/auth/permission-checks.log.md":      {},
	"practices/auth/permission-checks/examples.md": {},
	"debts/a/b/c/deep.md":                          {},
	"bugs/x.spec.md":                               {},

	"releases/INDEX.md":      {},
	"releases/1.2.0.md":      {},
	"releases/2026/1.3.0.md": {},
})

func TestFile(t *testing.T) {
	for _, tc := range []struct {
		rel  string
		want Position
	}{
		{"INDEX.md", Position{Kind: Index}},
		{"LOG.md", Position{Kind: Log}},
		{"README.md", Position{Kind: Readme}},
		{"SPEC.md", Position{Kind: Reference}},
		{"STACK.md", Position{Kind: Context}},
		{"features/INDEX.md", Position{Kind: Index, Register: "features", ID: "features"}},
		{"features/onboarding.md", Position{Kind: Document, Register: "features", ID: "features/onboarding"}},
		{"features/onboarding.spec.md", Position{Kind: Trail, Register: "features", ID: "features/onboarding", Role: "spec"}},
		{"features/onboarding/01-signup-form.md", Position{Kind: Task, Register: "features", ID: "features/onboarding"}},
		{"features/platform/LOG.md", Position{Kind: Log, Register: "features", ID: "features/platform"}},
		{"features/platform/payments/instant-refunds.md", Position{Kind: Document, Register: "features", ID: "features/platform/payments/instant-refunds"}},
		{"features/platform/payments/instant-refunds.test.md", Position{Kind: Trail, Register: "features", ID: "features/platform/payments/instant-refunds", Role: "test"}},
		{"features/platform/payments/instant-refunds/02-refund-ui.md", Position{Kind: Task, Register: "features", ID: "features/platform/payments/instant-refunds"}},
		{"features/bugs/triage.md", Position{Kind: Document, Register: "features", ID: "features/bugs/triage"}},
		{"features/index/overview.md", Position{Kind: Document, Register: "features", ID: "features/index/overview"}},
		{"changes/payments/refund-window.md", Position{Kind: Document, Register: "changes", ID: "changes/payments/refund-window"}},
		{"changes/payments/refund-window.plan.md", Position{Kind: Trail, Register: "changes", ID: "changes/payments/refund-window", Role: "plan"}},
		{"changes/payments/refund-window/01-shorten.md", Position{Kind: Task, Register: "changes", ID: "changes/payments/refund-window"}},
		{"practices/auth/permission-checks.md", Position{Kind: Document, Register: "practices", ID: "practices/auth/permission-checks"}},
		{"practices/auth/permission-checks.log.md", Position{Kind: Trail, Register: "practices", ID: "practices/auth/permission-checks", Role: "log"}},
		{"debts/a/b/c/deep.md", Position{Kind: Document, Register: "debts", ID: "debts/a/b/c/deep"}},
		{"releases/INDEX.md", Position{Kind: Index, Register: "releases", ID: "releases"}},
		{"releases/1.2.0.md", Position{Kind: Document, Register: "releases", ID: "releases/1.2.0"}},
	} {
		if got := sample.File(tc.rel); got != tc.want {
			t.Errorf("File(%q) = %+v; want %+v", tc.rel, got, tc.want)
		}
	}
}

// Every path with no 1.0 position is a Stray that names the path at fault —
// the file, or the directory that should not hold it — and says why.
func TestFileStray(t *testing.T) {
	for _, tc := range []struct {
		rel, where, problem string
	}{
		{"notes.md", "notes.md", "the bundle root holds only INDEX.md, LOG.md, SPEC.md, README.md"},
		{"drafts/idea.md", "drafts/", "a feature group belongs under features/"},
		{"features/onboarding/INDEX.md", "features/onboarding/INDEX.md", "task directories may contain only NN-slug.md tasks"},
		{"features/onboarding/sub/02-x.md", "features/onboarding/sub/", "task directories may contain only NN-slug.md tasks, and no directory"},
		{"features/platform/payments/instant-refunds.notes.md", "features/platform/payments/instant-refunds.notes.md", `unknown trail role "notes" — allowed roles are spec, plan, test, surface, log`},
		{"features/platform/payments/Refunds.md", "features/platform/payments/Refunds.md", "filenames are lowercase"},
		{"features/Payments/x.md", "features/Payments/", "directory names must be lowercase [a-z0-9-]"},
		{"changes/payments/refund-window.test.md", "changes/payments/refund-window.test.md", `unknown trail role "test" under changes/`},
		{"practices/auth/permission-checks/examples.md", "practices/auth/permission-checks/", "shares its name with the practice practices/auth/permission-checks.md, and a practice owns no directory"},
		{"bugs/x.spec.md", "bugs/x.spec.md", `unknown trail role "spec" under bugs/ — a bug is a register entry`},
		{"releases/2026/1.3.0.md", "releases/2026/", "releases/ is flat"},
		// A disk that ignores case reads these as the reserved files beside
		// them, so they are never a document's name, wherever they are.
		{"features/index.md", "features/index.md", "no document is named index.md: a disk that ignores case reads it as the INDEX.md beside it"},
		{"debts/a/log.md", "debts/a/log.md", "no document is named log.md: a disk that ignores case reads it as the LOG.md beside it"},
		{"releases/index.md", "releases/index.md", "no document is named index.md"},
	} {
		got := sample.File(tc.rel)
		if got.Kind != Stray || got.Where != tc.where || !strings.Contains(got.Problem, tc.problem) {
			t.Errorf("File(%q) = %+v; want a Stray at %q saying %q", tc.rel, got, tc.where, tc.problem)
		}
	}
}

func TestDir(t *testing.T) {
	for _, tc := range []struct {
		rel  string
		want Position
	}{
		{"features", Position{Kind: Register, Register: "features", ID: "features"}},
		{"features/platform/payments", Position{Kind: Group, Register: "features", ID: "features/platform/payments"}},
		{"features/bugs", Position{Kind: Group, Register: "features", ID: "features/bugs"}},
		{"features/index", Position{Kind: Group, Register: "features", ID: "features/index"}},
		{"features/onboarding", Position{Kind: TaskDir, Register: "features", ID: "features/onboarding"}},
		{"changes/payments/refund-window", Position{Kind: TaskDir, Register: "changes", ID: "changes/payments/refund-window"}},
		{"debts/a/b/c", Position{Kind: Group, Register: "debts", ID: "debts/a/b/c"}},
		{"releases", Position{Kind: Register, Register: "releases", ID: "releases"}},
	} {
		if got := sample.Dir(tc.rel); got != tc.want {
			t.Errorf("Dir(%q) = %+v; want %+v", tc.rel, got, tc.want)
		}
	}
	for _, rel := range []string{"drafts", "assets", "releases/2026", "practices/auth/permission-checks", "features/Payments"} {
		if got := sample.Dir(rel); got.Kind != Stray {
			t.Errorf("Dir(%q) = %+v; want a Stray", rel, got)
		}
	}
}

// A directory that holds no Markdown at any depth, hidden files aside, is
// outside FDF, however deep its files are.
func TestHoldsMarkdown(t *testing.T) {
	b := New(fstest.MapFS{
		"features/shop-images/logo.png":        {},
		"features/shop-images/raw/logo.psd":    {},
		"features/shop-images/.cache/notes.md": {},
		"features/platform/payments/x.md":      {},
		"debts/rounding/diagram.png":           {},
	})
	for rel, want := range map[string]bool{
		"features/shop-images":     false,
		"features/shop-images/raw": false,
		"features/platform":        true,
		"features":                 true,
		"debts/rounding":           false,
		"debts/nowhere":            false,
	} {
		if got := b.HoldsMarkdown(rel); got != want {
			t.Errorf("HoldsMarkdown(%q) = %v; want %v", rel, got, want)
		}
	}
}

func TestRegistersAndContextDocs(t *testing.T) {
	for _, name := range []string{"features", "changes", "practices", "debts", "bugs", "releases"} {
		if !IsRegister(name) {
			t.Errorf("IsRegister(%q) = false", name)
		}
	}
	if IsRegister("payments") || IsContext("SPEC.md") || !IsContext("DOMAIN.md") {
		t.Error("payments is no register, SPEC.md no Context document, and DOMAIN.md is one")
	}
}
