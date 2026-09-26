package links

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFindReturnsEveryTargetAndMarksCode(t *testing.T) {
	text := "See [a](one.md) and ![img](pics/x.png \"title\").\n\n" +
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
		{"its own sibling, written ./", "./refund-window.spec.md", ""},
		{"a sibling left behind, written ./", "./other.md", "../other.md"},
		// The bundle root does not move, so a link written from it still
		// names the same file, wherever it points.
		{"a document beside the bundle, from the bundle root", "/../okf/modules/auth.md", ""},
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
		{"the moved document, written ./", "./../changes/refund-window.md", "../changes/payments/refund-window.md"},
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
		{"a feature in the same group, written ./", "./y.md", ""},
		// A link to a file outside the bundle is written relative (spec 1.0,
		// Cross-linking), so one written from the bundle root that the move
		// must rewrite is written relative.
		{"a document beside the bundle, from the bundle root", "/../okf/modules/auth.md", "../../../okf/modules/auth.md"},
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
		"[e](<my notes.md>) ![f](<pics/a b.png> \"T\")\n\n"+
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

// Code returns the ranges Find reads as code, in order: a fenced block from
// its opening fence through its closing one, a heading it quotes included;
// each line of an indented block; each code span with its backticks, one of
// unequal runs included; and a fence that nothing closes, to the end of the
// text. A link is in code exactly when its target starts in one of them.
func TestCodeReturnsTheRangesFindReadsAsCode(t *testing.T) {
	text := "Use ``a`b`` then [x](x.md) and `c`.\n\n" +
		"```markdown\n# Example\n[y](y.md)\n```\n\n" +
		"    [z](z.md) indented\n    and more\n\n" +
		"~~~\n[w](w.md) unclosed"
	want := []string{
		"``a`b``",
		"`c`",
		"```markdown\n# Example\n[y](y.md)\n```\n",
		"    [z](z.md) indented\n",
		"    and more\n",
		"~~~\n[w](w.md) unclosed",
	}
	got := Code(text)
	if len(got) != len(want) {
		t.Fatalf("Code found %d ranges, want %d: %+v", len(got), len(want), got)
	}
	for i, s := range got {
		if text[s.Start:s.End] != want[i] {
			t.Errorf("range %d = %q, want %q", i, text[s.Start:s.End], want[i])
		}
	}
	for _, l := range Find(text) {
		in := false
		for _, s := range got {
			in = in || s.Start <= l.Start && l.Start < s.End
		}
		if in != l.InCode || in != (l.Target != "x.md") {
			t.Errorf("link %q: InCode=%v, in a range of Code=%v; want both %v", l.Target, l.InCode, in, l.Target != "x.md")
		}
	}
}

// A destination may hold parentheses that balance, and a backslash escapes
// the next character; one whose parentheses do not balance is no link.
func TestFindReadsBalancedParenthesesInADestination(t *testing.T) {
	checkFind(t, "[a](five(1).md) [b](x\\)y.md) [c](open(.md) [d](<six(.md> \"T\")\n", []wantLink{
		{target: "five(1).md"},
		{target: "x\\)y.md"},
		{target: "<six(.md>"},
	})
}

// A fence marker indented four columns or more opens no fence: after a blank
// line it is indented code, and inside a paragraph it is prose. So nothing
// after it is swallowed as code when no fence closes.
func TestFindAFenceIndentedFourColumnsOpensNoFence(t *testing.T) {
	checkFind(t, "Intro.\n\n    ```\n    [a](indented.md)\n\n[b](after.md)\n\n"+
		"A paragraph\n    ```\n[c](still-prose.md)\n", []wantLink{
		{target: "indented.md", inCode: true},
		{target: "after.md"},
		{target: "still-prose.md"},
	})
}

// A backtick fence's info string holds no backtick, as CommonMark reads it:
// a line of prose that starts with a code span of three backticks, such as
// "```go fmt``` formats …", opens no fence, so the links on it and after it
// are prose. A tilde fence's info string may hold one.
func TestFindABacktickInTheInfoStringOpensNoFence(t *testing.T) {
	checkFind(t, "```go fmt``` formats [a](a.md).\n\n[b](b.md)\n\n"+
		"~~~ `info`\n[c](c.md)\n~~~\n[d](d.md)\n", []wantLink{
		{target: "a.md"},
		{target: "b.md"},
		{target: "c.md", inCode: true},
		{target: "d.md"},
	})
}

// A heading is a block of its own: a backtick in it pairs with nothing on the
// next line, so it hides no link there.
func TestFindAHeadingEndsItsBlock(t *testing.T) {
	checkFind(t, "# A ` in a heading\n[a](a.md) and `code`\n", []wantLink{{target: "a.md"}})
}

// Text that only looks like a link is none: a reference definition that
// would interrupt a paragraph, a bracket a backslash escapes, and a [ in a
// code span, which opens no link text when the ] is outside it.
func TestFindReadsNoLinkWhereCommonMarkReadsText(t *testing.T) {
	checkFind(t, "A paragraph\n[x]: interrupts.md\n\n"+
		"[y]: starts-a-block.md\n[z]: follows-a-definition.md\n\n"+
		"\\[a](escaped.md) and `[` then ](after-span.md)\n\n"+
		"[b `[` c](text-with-a-span.md)\n", []wantLink{
		{target: "starts-a-block.md", def: true},
		{target: "follows-a-definition.md", def: true},
		{target: "text-with-a-span.md"},
	})
}

// A line indented four columns right after a heading or a closing fence is
// indented code, as it is after a blank line.
func TestFindReadsIndentedCodeAfterAHeadingOrAFence(t *testing.T) {
	checkFind(t, "# Heading\n    [a](after-heading.md)\n\n```\nx\n```\n    [b](after-fence.md)\n", []wantLink{
		{target: "after-heading.md", inCode: true},
		{target: "after-fence.md", inCode: true},
	})
}

// A line read on its own, as fdf mv reads one listing line of an index, is
// never code for its indentation alone.
func TestFindReadsAnIndentedLineOnItsOwnAsProse(t *testing.T) {
	checkFind(t, "    * [nested](nested.md) - a nested listing", []wantLink{{target: "nested.md"}})
}

// Blocks returns the code blocks alone: a fenced block and each line of an
// indented one, but no code span.
func TestBlocksReturnsTheCodeBlocksAlone(t *testing.T) {
	text := "Prose with `a span`.\n\n```sh\nfdf new x\n```\n\n    indented\n"
	var got []string
	for _, b := range Blocks(text) {
		got = append(got, text[b.Start:b.End])
	}
	if want := []string{"```sh\nfdf new x\n```\n", "    indented\n"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Blocks = %q, want %q", got, want)
	}
}

// Rendered reads where a text links: an inline link at its target, and a
// reference — full, collapsed or shortcut — where it is, with its
// definition's target, the first of a label's, matched whatever its case and
// spacing. A definition renders nothing, nor does a reference no definition
// names, a footnote, a checkbox, or a link in code.
func TestRenderedReadsAReferenceWhereItIs(t *testing.T) {
	text := "# Tasks\n\n1. [Signup form][signup]\n2. [Welcome   Mail][]\n3. [Setup] and [x] done, [^1] noted.\n4. [Inline](inline.md) and `[signup]`.\n\n" +
		"# Notes\n\n[signup]: tasks/01-signup.md\n[welcome mail]: <tasks/02-welcome mail.md>\n[setup]: tasks/03-setup.md \"T\"\n[SIGNUP]: tasks/other.md\n"
	var got []string
	for _, l := range Rendered(text) {
		got = append(got, fmt.Sprintf("%s@%d", l.Target, l.Start))
	}
	want := []string{
		"tasks/01-signup.md@" + fmt.Sprint(strings.Index(text, "[Signup form]")),
		"<tasks/02-welcome mail.md>@" + fmt.Sprint(strings.Index(text, "[Welcome")),
		"tasks/03-setup.md@" + fmt.Sprint(strings.Index(text, "[Setup]")),
		"inline.md@" + fmt.Sprint(strings.Index(text, "inline.md")),
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Rendered:\n got %q\nwant %q", got, want)
	}
}

// Find reads a text in one pass. A paragraph of 2,000 lines, each with a ]
// that closes no link's text and a code span, reads in well under a second:
// read by looking back from each ] over the paragraph, it took seconds.
func TestFindReadsALongParagraphInOnePass(t *testing.T) {
	text := strings.Repeat("a see](b.md) `c` and ](d) text\n", 2000)
	start := time.Now()
	if got := Find(text); len(got) != 0 {
		t.Fatalf("no [ opens a link's text, so there is no link: %+v", got[0])
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("Find took %v over a paragraph of 2,000 lines", d)
	}
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
