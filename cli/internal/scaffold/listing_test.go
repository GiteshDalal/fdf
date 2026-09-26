package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
)

// A document filed in groups within groups is listed in its innermost
// group's index, and each group that is new gets an index of its own, listed
// in its parent's, so every group can be found from the register's index.
func TestListEntryListsEveryNewGroupOnTheWay(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	if code := EnsureIndex(root, "features", &out); code != 0 {
		t.Fatalf("EnsureIndex: exit %d\n%s", code, out.String())
	}
	if err := os.MkdirAll(filepath.Join(root, "features", "platform", "payments"), 0o755); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := ListEntry(root, "features", "platform/payments/instant-refunds", "Instant refunds", "feature", &out); code != 0 {
		t.Fatalf("ListEntry: exit %d\n%s", code, out.String())
	}
	for rel, want := range map[string]string{
		"features/INDEX.md":                   "* [Platform](/features/platform/INDEX.md) - features in platform.\n",
		"features/platform/INDEX.md":          "# Platform\n\n* [Payments](/features/platform/payments/INDEX.md) - features in payments.\n",
		"features/platform/payments/INDEX.md": "# Payments\n\n* [Instant refunds](/features/platform/payments/instant-refunds.md) - feature.\n",
	} {
		raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, raw)
		}
	}
	want := "wrote features/platform/INDEX.md\n" +
		"updated features/INDEX.md (now lists \"Platform\")\n" +
		"wrote features/platform/payments/INDEX.md\n" +
		"updated features/platform/INDEX.md (now lists \"Payments\")\n" +
		"updated features/platform/payments/INDEX.md (now lists \"Instant refunds\")\n"
	if out.String() != want {
		t.Errorf("each index is reported as it is written:\n got: %q\nwant: %q", out.String(), want)
	}
}

// A listing is a list item's first link outside code, read as CommonMark
// reads it: a title is not part of the path, a <…> destination is, and a
// link that names no path (a URL, an anchor) makes the line no listing.
func TestListingTargetReadsTheFirstLink(t *testing.T) {
	for _, tc := range []struct{ line, dir, want string }{
		{"* [A](/features/a.md) - feature.", "features", "features/a.md"},
		{"* [A](a.md) - relative to the index.", "features/platform", "features/platform/a.md"},
		{`* [A](/features/a.md "the title") - feature.`, "features", "features/a.md"},
		{"* [A](</features/a b.md>) - feature.", "features", "features/a b.md"},
		{"* [A](/features/a.md#scenarios) - feature.", "features", "features/a.md"},
		{"  - [Payments](payments/INDEX.md) - a group.", "features", "features/payments/INDEX.md"},
		{"* `[x](/features/x.md)` then [B](/features/b.md) - feature.", "features", "features/b.md"},
		{"* [upstream](https://example.com/a.md) and [B](/features/b.md)", "features", ""},
		{"* [Top](#top) - an anchor.", "features", ""},
		{"[A](/features/a.md) - not a list item.", "features", ""},
		{"* no link at all", "features", ""},
	} {
		if got := ListingTarget(tc.line, tc.dir); got != tc.want {
			t.Errorf("ListingTarget(%q, %q) = %q; want %q", tc.line, tc.dir, got, tc.want)
		}
	}
}

// A document whose listing carries a title is found and taken away with it,
// as one written with fdf's own plain link is.
func TestUnlistFindsATitledListing(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "debts"), 0o755)
	idx := "# Debt\n\n* [Rounding](/debts/rounding.md \"the half-down one\") - debt.\n* [Hours](/debts/hours.md) - debt.\n"
	os.WriteFile(filepath.Join(root, "debts", "INDEX.md"), []byte(idx), 0o644)
	if got := ListedIn(root, "debts/rounding.md"); got != "debts/INDEX.md" {
		t.Errorf("ListedIn = %q; want debts/INDEX.md", got)
	}
	if changed, err := Unlist(root, "debts/rounding.md"); err != nil || changed != "debts/INDEX.md" {
		t.Fatalf("Unlist = %q, %v", changed, err)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "debts", "INDEX.md"))
	if string(raw) != "# Debt\n\n* [Hours](/debts/hours.md) - debt.\n" {
		t.Errorf("only the titled listing goes:\n%s", raw)
	}
}

// Every register but releases/ gets its index from one place.
func TestEnsureIndexWritesEachRegistersIndexOnce(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	for _, reg := range []string{"features", "changes", "practices", "debts", "bugs"} {
		if code := EnsureIndex(root, reg, &out); code != 0 {
			t.Fatalf("EnsureIndex(%s): exit %d\n%s", reg, code, out.String())
		}
		raw, err := os.ReadFile(filepath.Join(root, reg, "INDEX.md"))
		if err != nil || !strings.Contains(string(raw), "* [Format reference](/SPEC.md) - how ") {
			t.Errorf("%s/INDEX.md: %v\n%s", reg, err, raw)
		}
	}
	if !strings.Contains(out.String(), "wrote features/INDEX.md (what the software does)\n") {
		t.Errorf("the features index is reported:\n%s", out.String())
	}
	out.Reset()
	os.WriteFile(filepath.Join(root, "bugs", "INDEX.md"), []byte("# Mine\n"), 0o644)
	if code := EnsureIndex(root, "bugs", &out); code != 0 || out.Len() != 0 {
		t.Errorf("an index that exists is left alone: exit %d %q", code, out.String())
	}
	if code := EnsureIndex(root, "releases", &out); code != 1 {
		t.Errorf("releases/ keeps its own index (fdf release): exit %d", code)
	}
}

// Each register layout knows has its entry in every table scaffold keeps per
// register: the line the root INDEX.md lists it with and, but for releases/,
// which is flat and whose index `fdf release` writes, the index `fdf init`
// writes and the noun its groups are listed with. A register added to
// layout.Registers fails here until every table has it.
func TestEveryRegisterIsInScaffoldsTables(t *testing.T) {
	for _, reg := range layout.Registers {
		listed := false
		for _, r := range registerListings {
			listed = listed || r.reg == reg && r.line != ""
		}
		if !listed {
			t.Errorf("registerListings has no line for %s/", reg)
		}
		if reg == "releases" {
			continue
		}
		if _, ok := registerIndexes[reg]; !ok {
			t.Errorf("registerIndexes has no index for %s/", reg)
		}
		if groupNouns[reg] == "" {
			t.Errorf("groupNouns has no noun for %s/", reg)
		}
	}
}
