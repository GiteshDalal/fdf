package bundle

// v0.7 adopted features. A capability the software had before the bundle
// existed was never specified, planned or built through FDF, so it enters the
// bundle as `adopted`: documented from the code as it stands. It has no build
// trail — writing one after the fact would invent a history — and it reaches
// its code through its own `resource`, since it has no tasks to do that. It
// may begin as a map entry (a Feature: block and no scenarios); from its
// first scenario, slug.test.md covers each one.

import (
	"fmt"
)

// checkAdopted is F4 and F8 for one adopted feature. p is its trail, or nil
// when it has no siblings at all.
func checkAdopted(fid string, f *featureInfo, p *pairInfo, errs, warns *[]string) {
	if p != nil {
		if p.spec {
			*errs = append(*errs, fmt.Sprintf("%s: status 'adopted' but %s.spec.md exists — an adopted feature documents code that already existed and has no build trail; a design for new work is a Change's spec (F4)", f.rel, fid))
		}
		if p.plan {
			*errs = append(*errs, fmt.Sprintf("%s: status 'adopted' but %s.plan.md exists — an adopted feature has no build trail; work on it is planned by a Change (F4)", f.rel, fid))
		}
		if len(p.tasks) > 0 {
			*errs = append(*errs, fmt.Sprintf("%s: status 'adopted' but task directory %s/ exists — an adopted feature has no build trail; work on it is tasked by a Change (F4)", f.rel, fid))
		}
	}
	if len(f.resource) == 0 {
		*errs = append(*errs, fmt.Sprintf("%s: status 'adopted' requires `resource` naming the code it documents — with no tasks, it is the feature's only link to the code (F4)", f.rel))
	}
	if f.version != "" {
		*errs = append(*errs, fmt.Sprintf("%s: an adopted feature carries no `version` — it shipped before the bundle recorded releases; the Change or Fix that later ships on it carries the version (F4)", f.rel))
	}

	// F8 from the first scenario: a backfilled scenario is only as good as the
	// verification that shows the code already does it.
	names := scenarioRe.FindAllStringSubmatch(f.body, -1)
	if len(names) == 0 {
		return
	}
	testPath := fid + ".test.md"
	if p == nil || !p.test {
		*errs = append(*errs, fmt.Sprintf("%s: status 'adopted' with scenarios requires %s — each backfilled scenario names the check that proves the code already does it (F8)", f.rel, testPath))
		return
	}
	checkTestCases(testPath, fid, featureScenarioNames(f.body), p.testBody, errs, warns)
}
