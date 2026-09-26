package bundle

// Test documents. A case in slug.test.md is a `## <scenario name>` heading
// under `# Test Cases`, and F8 matches it exactly, so a short name inside
// another case's text does not count as covered.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// caseFenceRe opens a fenced block: three or more backticks or tildes.
var caseFenceRe = regexp.MustCompile("^(`{3,}|~{3,})")

// TestCases reads a test document's `# Test Cases` section: the name of each
// `## <scenario name>` heading, with how many times it appears, and whether
// the section exists at all. A `##` line inside a fenced block is not a case.
func TestCases(body string) (map[string]int, bool) {
	cases := map[string]int{}
	inSection, found := false, false
	fence := ""
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if fence != "" {
			// A closing fence repeats the opener's character, at least as long.
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if m := caseFenceRe.FindStringSubmatch(t); m != nil {
			fence = m[1]
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			inSection = strings.EqualFold(strings.TrimSpace(m[1]), "Test Cases")
			found = found || inSection
			continue
		}
		if inSection {
			if m := subHeadingRe.FindStringSubmatch(line); m != nil {
				cases[strings.TrimSpace(m[1])]++
			}
		}
	}
	return cases, found
}

// featureScenarioNames lists a feature's scenario names in document order.
func featureScenarioNames(body string) []string {
	var names []string
	for _, m := range scenarioRe.FindAllStringSubmatch(body, -1) {
		names = append(names, strings.TrimSpace(m[1]))
	}
	return names
}

// checkSurfaces is v0.7's surface-document check. `surface`, when present, is
// `none`, and then no slug.surface.md exists (F4). A feature that has no
// surface document and does not say `surface: none` draws a warning from
// `specified` on, and an adopted one once it has a scenario: map entries,
// drafts and retired features are not asked yet, or any more.
func checkSurfaces(features map[string]*featureInfo, pairs map[string]*pairInfo, errs, warns *[]string) {
	for _, fid := range sortedFeatureIDs(features) {
		f := features[fid]
		p := pairs[fid]
		has := p != nil && p.surface
		switch {
		case f.surface != "" && f.surface != "none":
			*errs = append(*errs, fmt.Sprintf("%s: `surface` is `none` or absent, got %q (F4)", f.rel, f.surface))
			continue
		case f.surface == "none" && has:
			*errs = append(*errs, fmt.Sprintf("%s: says `surface: none` but %s.surface.md exists — drop one of them (F4)", f.rel, fid))
			continue
		case has || f.surface == "none":
			continue
		}
		asked := in([]string{"specified", "planned", "implementing", "done"}, f.status) ||
			(f.status == "adopted" && len(scenarioRe.FindAllString(f.body, -1)) > 0)
		if asked {
			*warns = append(*warns, fmt.Sprintf("%s: has no %s.surface.md — write one if the feature adds or changes something a person or another system uses directly (a screen, an endpoint, a command, an event), or say `surface: none` in its frontmatter", f.rel, fid))
		}
	}
}

// checkTestCases is F8 on a test document: every scenario has its
// `## <scenario name>` case (an error when missing). A case that names no
// scenario, or one repeated, is out of date and draws a warning.
func checkTestCases(testPath, featureID string, names []string, body string, errs, warns *[]string) {
	cases, found := TestCases(body)
	if !found {
		*errs = append(*errs, fmt.Sprintf("%s: has no `# Test Cases` section — each scenario's case is a `## <scenario name>` heading under it (F8)", testPath))
		return
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
		if cases[n] == 0 {
			*errs = append(*errs, fmt.Sprintf("%s: scenario %q has no test case — a case is a `## %s` heading under `# Test Cases`, matched exactly (F8)", testPath, n, n))
		}
	}
	var heads []string
	for h := range cases {
		heads = append(heads, h)
	}
	sort.Strings(heads)
	for _, h := range heads {
		switch {
		case !want[h]:
			*warns = append(*warns, fmt.Sprintf("%s: case `## %s` names no scenario of %s — remove it, or give it the scenario's exact name", testPath, h, featureID))
		case cases[h] > 1:
			*warns = append(*warns, fmt.Sprintf("%s: case `## %s` appears %d times — one case per scenario", testPath, h, cases[h]))
		}
	}
}
