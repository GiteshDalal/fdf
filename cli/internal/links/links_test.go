package links

import "testing"

func TestFindReturnsEveryTargetAndMarksCode(t *testing.T) {
	text := "See [a](one.md) and ![img](pics/x.png \"title\").\n" +
		"[ref]: ../two.md\n" +
		"   [spaced]: three.md \"t\"\n" +
		"[^1]: A footnote is not a link definition.\n" +
		"Inline `[x](code.md)` span.\n" +
		"```\n[y](fence.md)\n[def]: fence-def.md\n```\n" +
		"~~~markdown\n[z](tilde.md)\n~~~\n" +
		"````\n```\n[w](inner.md)\n```\n````\n" +
		"after [b](four.md#frag)\n"
	want := []wantLink{
		{target: "one.md"},
		{target: "pics/x.png"},
		{target: "../two.md", def: true},
		{target: "three.md", def: true},
		{target: "code.md", inCode: true},
		{target: "fence.md", inCode: true},
		{target: "fence-def.md", def: true, inCode: true},
		{target: "tilde.md", inCode: true},
		{target: "inner.md", inCode: true},
		{target: "four.md#frag"},
	}
	checkFind(t, text, want)
}

// A fence that never closes — a document cut off mid-edit — still marks
// everything after it as code, to the end of the text.
func TestFindAnUnclosedFenceRunsToTheEndOfTheText(t *testing.T) {
	got := Find("Before [a](a.md).\n```\n[b](b.md)\n")
	if len(got) != 2 || got[0].Target != "a.md" || got[0].InCode || got[1].Target != "b.md" || !got[1].InCode {
		t.Fatalf("Find = %+v; want a.md not in code and b.md in code", got)
	}
}

func TestMoveNewPrefersFilesThenTheLongestDirectory(t *testing.T) {
	m := Move{
		Files: map[string]string{"docs/features/ekpie/x.md": "docs/fdf/features/ekpie/renamed.md"},
		Dirs:  map[string]string{"docs/features": "docs/fdf", "docs/features/ekpie": "docs/fdf/features/ekpie"},
	}
	for _, tc := range []struct {
		in, want string
		moved    bool
	}{
		{"docs/features/ekpie/x.md", "docs/fdf/features/ekpie/renamed.md", true},
		{"docs/features/ekpie/y.md", "docs/fdf/features/ekpie/y.md", true},
		{"docs/features/changes/c.md", "docs/fdf/changes/c.md", true},
		{"docs/features", "docs/fdf", true},
		{"docs/features-old/a.md", "docs/features-old/a.md", false},
		{"docs/okf/index.md", "docs/okf/index.md", false},
	} {
		got, moved := m.New(tc.in)
		if got != tc.want || moved != tc.moved {
			t.Errorf("New(%q) = %q, %v; want %q, %v", tc.in, got, moved, tc.want, tc.moved)
		}
	}
}

// retargetCase is one link as written in one file, and what it must read
// once the move is done ("" when it must not change).
type retargetCase struct {
	name, target, want string
}

func runRetarget(t *testing.T, s Site, m Move, cases []retargetCase) {
	t.Helper()
	for _, tc := range cases {
		got, ok := Retarget(tc.target, s, m)
		switch {
		case tc.want == "" && ok:
			t.Errorf("%s: %q was rewritten to %q; it must stay as written", tc.name, tc.target, got)
		case tc.want != "" && (!ok || got != tc.want):
			t.Errorf("%s: %q became %q (changed=%v); want %q", tc.name, tc.target, got, ok, tc.want)
		}
	}
}

// fdf mv changes/refund-window changes/payments/refund-window: a Change filed
// flat moves one level down with its spec and its task directory. Paths are
// bundle-relative, and the bundle itself stays put.
var refundWindowMove = Move{
	Files: map[string]string{
		"changes/refund-window.md":      "changes/payments/refund-window.md",
		"changes/refund-window.spec.md": "changes/payments/refund-window.spec.md",
	},
	Dirs: map[string]string{"changes/refund-window": "changes/payments/refund-window"},
}

func TestRetargetFromAFileThatMovesDeeper(t *testing.T) {
	runRetarget(t, Site{OldPath: "changes/refund-window.md", NewPath: "changes/payments/refund-window.md"}, refundWindowMove, []retargetCase{
		{"a document that stays", "../payments/instant-refunds.md", "../../payments/instant-refunds.md"},
		{"a link from the bundle root", "/payments/instant-refunds.md", ""},
		{"a document outside the bundle", "../../okf/modules/auth.md", "../../../okf/modules/auth.md"},
		{"code outside the bundle", "../../../src/refund.go", "../../../../src/refund.go"},
		{"a fragment", "../../okf/modules/auth.md#login", "../../../okf/modules/auth.md#login"},
		{"a query", "../payments/instant-refunds.md?plain=1", "../../payments/instant-refunds.md?plain=1"},
		{"a directory", "../payments/", "../../payments/"},
		{"its own sibling, which moves with it", "refund-window.spec.md", ""},
		{"its own task, which moves with it", "refund-window/01-shorten.md", ""},
		{"a URL", "https://example.com/a.md", ""},
		{"an anchor", "#problem", ""},
		{"a mail address", "mailto:team@example.com", ""},
		{"a destination in angle brackets", "<../../okf/modules/auth.md>", "<../../../okf/modules/auth.md>"},
		{"a destination in angle brackets, with a fragment", "<../../okf/modules/auth.md#login>", "<../../../okf/modules/auth.md#login>"},
		{"an unclosed angle bracket", "<../../okf/modules/auth.md", ""},
		{"a URL in angle brackets", "<https://example.com/a.md>", ""},
	})
}

func TestRetargetFromATaskThatMovesWithItsOwner(t *testing.T) {
	runRetarget(t, Site{OldPath: "changes/refund-window/01-shorten.md", NewPath: "changes/payments/refund-window/01-shorten.md"}, refundWindowMove, []retargetCase{
		{"code outside the bundle", "../../../../src/refund.go", "../../../../../src/refund.go"},
		{"a document that stays", "../../payments/instant-refunds.md", "../../../payments/instant-refunds.md"},
		{"its owner, which moves with it", "../refund-window.md", ""},
	})
}

func TestRetargetFromAFileThatStaysToOneThatMoves(t *testing.T) {
	runRetarget(t, Site{OldPath: "payments/instant-refunds.md", NewPath: "payments/instant-refunds.md"}, refundWindowMove, []retargetCase{
		{"the moved document", "../changes/refund-window.md", "../changes/payments/refund-window.md"},
		{"the moved document from the bundle root", "/changes/refund-window.md", "/changes/payments/refund-window.md"},
		{"a file in the moved task directory", "../changes/refund-window/01-shorten.md", "../changes/payments/refund-window/01-shorten.md"},
		{"nothing moved at either end", "../../okf/modules/auth.md", ""},
		{"the moved document, in angle brackets", "<../changes/refund-window.md>", "<../changes/payments/refund-window.md>"},
	})
}

func TestRetargetFromAFileThatMovesShallower(t *testing.T) {
	m := Move{Files: map[string]string{"changes/payments/x.md": "changes/x.md"}}
	runRetarget(t, Site{OldPath: "changes/payments/x.md", NewPath: "changes/x.md"}, m, []retargetCase{
		{"a document outside the bundle", "../../../okf/a.md", "../../okf/a.md"},
		{"a sibling left behind", "y.md", "payments/y.md"},
	})
}

// The 1.0 migration, in project-relative paths: the bundle moves from
// docs/features to docs/fdf, and its feature group ekpie/ moves under
// features/. A link written from the bundle root resolves against the bundle
// root, which moves too.
var migrationMove = Move{Dirs: map[string]string{
	"docs/features":       "docs/fdf",
	"docs/features/ekpie": "docs/fdf/features/ekpie",
}}

func TestRetargetAcrossABundleThatMoves(t *testing.T) {
	feature := Site{OldPath: "docs/features/ekpie/x.md", NewPath: "docs/fdf/features/ekpie/x.md", OldBase: "docs/features", NewBase: "docs/fdf"}
	runRetarget(t, feature, migrationMove, []retargetCase{
		{"a document beside the bundle", "../../okf/modules/auth.md", "../../../okf/modules/auth.md"},
		{"a feature, from the bundle root", "/ekpie/y.md", "/features/ekpie/y.md"},
		{"a register document, from the bundle root", "/changes/c.md", ""},
		{"a register document", "../changes/c.md", "../../changes/c.md"},
		{"a feature in the same group", "y.md", ""},
	})
	index := Site{OldPath: "docs/features/INDEX.md", NewPath: "docs/fdf/INDEX.md", OldBase: "docs/features", NewBase: "docs/fdf"}
	runRetarget(t, index, migrationMove, []retargetCase{
		{"a group index", "ekpie/INDEX.md", "features/ekpie/INDEX.md"},
		{"a group index, from the bundle root", "/ekpie/INDEX.md", "/features/ekpie/INDEX.md"},
		{"a document beside the bundle, at the same depth", "../okf/index.md", ""},
	})
	outside := Site{OldPath: "docs/launch/plan.md", NewPath: "docs/launch/plan.md"}
	runRetarget(t, outside, migrationMove, []retargetCase{
		{"a debt in the bundle", "../features/debts/x.md", "../fdf/debts/x.md"},
		{"a feature in the bundle", "../features/ekpie/x.md", "../fdf/features/ekpie/x.md"},
		{"the bundle itself", "../features/", "../fdf/"},
	})
}

// wantLink is one link Find must return: its target, and whether it is a
// reference definition and whether it is code.
type wantLink struct {
	target      string
	def, inCode bool
}

func checkFind(t *testing.T, text string, want []wantLink) {
	t.Helper()
	got := Find(text)
	if len(got) != len(want) {
		t.Fatalf("Find found %d links, want %d: %+v", len(got), len(want), got)
	}
	for i, g := range got {
		w := want[i]
		if g.Target != w.target || g.Def != w.def || g.InCode != w.inCode {
			t.Errorf("link %d = %+v; want target %q def=%v inCode=%v", i, g, w.target, w.def, w.inCode)
		}
		if text[g.Start:g.End] != g.Target {
			t.Errorf("link %d: offsets [%d:%d] hold %q, not the target %q", i, g.Start, g.End, text[g.Start:g.End], g.Target)
		}
	}
}

// Find reads every form CommonMark gives a link: a title in any of its three
// quotings, a destination in angle brackets, which may hold spaces, spaces
// inside the parentheses, link text that wraps, and a link inside another's
// text. A "](" that no "[" opens in its paragraph is not a link.
func TestFindReadsEveryLinkForm(t *testing.T) {
	checkFind(t, "[a](one.md \"T\") [b](two.md 'T') [c](three.md (T)) [d]( four.md )\n"+
		"[e](<my notes.md>) ![f](<pics/a b.png> \"T\")\n"+
		"[g]: <five six.md> \"T\"\n\n"+
		"Not links: a] (x) and b](seven.md).\n\n"+
		"[![badge](badge.svg)](eight.md)\n\n"+
		"[a link that\nwraps](nine.md)\n", []wantLink{
		{target: "one.md"},
		{target: "two.md"},
		{target: "three.md"},
		{target: "four.md"},
		{target: "<my notes.md>"},
		{target: "<pics/a b.png>"},
		{target: "<five six.md>", def: true},
		{target: "badge.svg"},
		{target: "eight.md"},
		{target: "nine.md"},
	})
}

// Code is where CommonMark puts it. A code span closes only on a run of as
// many backticks as opened it, and may wrap onto the next line of its
// paragraph. An indented code block starts after a blank line, outside a
// list, and a tab indents to column four; a line indented after a paragraph
// line, or inside a list, is prose.
func TestFindMarksCodeAsCommonMarkDoes(t *testing.T) {
	checkFind(t, "Intro.\n\n"+
		"``a `[x](one.md)` b`` then [y](two.md)\n"+
		"`a span that\nwraps [z](three.md)` then [w](four.md)\n\n"+
		"    [v](five.md) in an indented block\n\n"+
		"\t[u]: six.md\n\n"+
		"A paragraph\n    continued with [t](seven.md)\n\n"+
		"- a list item\n\n"+
		"    continued with [s](eight.md)\n", []wantLink{
		{target: "one.md", inCode: true},
		{target: "two.md"},
		{target: "three.md", inCode: true},
		{target: "four.md"},
		{target: "five.md", inCode: true},
		{target: "six.md", def: true, inCode: true},
		{target: "seven.md"},
		{target: "eight.md"},
	})
}

// A line read on its own, as fdf mv reads one listing line of an index, is
// never code for its indentation alone.
func TestFindReadsAnIndentedLineOnItsOwnAsProse(t *testing.T) {
	checkFind(t, "    * [nested](nested.md) - a nested listing", []wantLink{{target: "nested.md"}})
}

func TestResolveReadsATargetAsAPath(t *testing.T) {
	for _, tc := range []struct {
		target, from, base string
		want               Dest
	}{
		{"../b/c.md#x", "a/f.md", "", Dest{Path: "b/c.md", Suffix: "#x"}},
		{"c.md?plain=1", "a/f.md", "", Dest{Path: "a/c.md", Suffix: "?plain=1"}},
		{"/b/c.md", "a/f.md", "", Dest{Path: "b/c.md", FromBase: true}},
		{"/features/", "docs/fdf/a/f.md", "docs/fdf", Dest{Path: "docs/fdf/features", FromBase: true, Dir: true}},
		{"../../okf/x.md", "a/f.md", "", Dest{Path: "../okf/x.md"}},
		{"<my notes.md#top>", "a/f.md", "", Dest{Path: "a/my notes.md", Suffix: "#top", Angle: true}},
	} {
		got, ok := Resolve(tc.target, tc.from, tc.base)
		if !ok || got != tc.want {
			t.Errorf("Resolve(%q, %q, %q) = %+v, %v; want %+v, true", tc.target, tc.from, tc.base, got, ok, tc.want)
		}
	}
	for _, target := range []string{"", "#anchor", "?q", "https://example.com/a.md", "mailto:team@example.com", "tel:+441234", "<unclosed.md", "<>"} {
		if got, ok := Resolve(target, "a/f.md", ""); ok {
			t.Errorf("Resolve(%q) = %+v, true; want it read as no path", target, got)
		}
	}
}
