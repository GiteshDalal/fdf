package migrate

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/scaffold"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// buildV01Bundle writes a miniature FDF v0.1 bundle: lowercase reserved
// names, vendored fdf-spec.md, a planned feature without TEST.md.
func buildV01Bundle(t *testing.T, root string) {
	write(t, root, "index.md", "---\nfdf_version: \"0.1\"\n---\n\n# Bundle\n\n* [FDF format](/fdf-spec.md) - vendored spec.\n* [Wdise](/wdise/index.md) - group.\n* [log](/log.md) - root-absolute trail link.\n* [u](https://example.com/a/log.md) - external URL, must stay untouched.\n* [w](https://example.com/other-fdf-spec.md-notes.md) - external URL, must stay untouched.\n* [OKF map](../okf/index.md) - sibling bundle (not FDF; must not be rewritten).\n")
	write(t, root, "log.md", "# Bundle Update Log\n\n## 2026-07-05\n* **Initialization**: v0.1.\n")
	write(t, root, "fdf-spec.md", "---\ntype: Reference\ntitle: FDF v0.1\ndescription: vendored.\ntimestamp: 2026-07-05T00:00:00Z\n---\n\nOld spec body; see [the index](index.md).\n")
	write(t, root, "wdise/index.md", "# Wdise\n\n* [Example](/wdise/example.md) - example.\n")
	write(t, root, "wdise/example.md", `---
type: Feature
title: Example
description: An example feature.
status: planned
timestamp: 2026-07-05T00:00:00Z
---

# Feature

`+"```gherkin\nFeature: Example\n  As a user\n  I want a thing\n  So that I get value\n```"+`

Trail: [spec](example/spec.md), [plan](example/plan.md).

# Scenarios

`+"```gherkin\nScenario: It works\n  Given a thing\n  When it runs\n  Then it works\n```"+`
`)
	write(t, root, "wdise/example/spec.md", "---\ntype: Spec\ntitle: S\ndescription: d.\ntimestamp: 2026-07-05T00:00:00Z\n---\n\n# Design\n\nWords.\n")
	write(t, root, "wdise/example/plan.md", "---\ntype: Plan\ntitle: P\ndescription: d.\ntimestamp: 2026-07-05T00:00:00Z\n---\n\n# Tasks\n\n1. (tasks pending)\n")
}

// buildV03Bundle writes a v0.3 nested-trail bundle with a done feature,
// task, and filled Context docs except no SURFACES.md (v0.4 addition).
func buildV03Bundle(t *testing.T, root string) {
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.3\"\n---\n\n# Bundle\n\n* [Wdise](/wdise/INDEX.md) - group.\n* [Log](/LOG.md) - root log.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* **Initialization**: v0.3.\n")
	write(t, root, "STACK.md", "---\ntype: Context\ntitle: Technology Stack\ndescription: Current stack snapshot.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Technology Stack\n\nGo.\n")
	write(t, root, "ARCHITECTURE.md", "---\ntype: Context\ntitle: Architecture\ndescription: Arch snapshot.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Architecture\n\nFlat.\n")
	write(t, root, "INFRA.md", "---\ntype: Context\ntitle: Infra\ndescription: Infra snapshot.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Infra\n\nLocal.\n")
	write(t, root, "wdise/INDEX.md", "# Wdise\n\n* [Example](/wdise/example.md) - example. (**done**)\n")
	write(t, root, "wdise/example.md", `---
type: Feature
title: Example
description: An example feature.
status: done
timestamp: 2026-07-06T00:00:00Z
---

# Feature

`+"```gherkin\nFeature: Example\n  As a user\n  I want a thing\n  So that I get value\n```"+`

Trail: [spec](example/SPEC.md), [plan](example/PLAN.md), [test](example/TEST.md).

# Scenarios

`+"```gherkin\nScenario: It works\n  Given a thing\n  When it runs\n  Then it works\n```"+`
`)
	write(t, root, "wdise/example/SPEC.md", "---\ntype: Spec\ntitle: Example spec\ndescription: Design.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Design\n\nWords.\n")
	write(t, root, "wdise/example/PLAN.md", "---\ntype: Plan\ntitle: Example plan\ndescription: Plan.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Tasks\n\n1. [Do the thing](01-do-thing.md)\n")
	write(t, root, "wdise/example/TEST.md", "---\ntype: Test\ntitle: Example acceptance\ndescription: How proven.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Test Cases\n\n## It works\n\nverified.\n")
	write(t, root, "wdise/example/01-do-thing.md", "---\ntype: Task\ntitle: Do the thing\ndescription: One unit of work.\nstatus: done\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Objective\n\nDo it.\n")
	write(t, root, "wdise/example/LOG.md", "# Example feature log\n\n## 2026-07-06\n* Completed.\n")
}

func TestMigrateChainsToItsTarget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV01Bundle(t, root)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for _, p := range []string{
		"INDEX.md", "LOG.md", "features/INDEX.md", "features/wdise/INDEX.md",
		"features/wdise/example.spec.md", "features/wdise/example.plan.md", "features/wdise/example.test.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing %s after migrate", p)
		}
	}
	// The feature group moved into features/, and its nested trail is gone.
	for _, p := range []string{"wdise", "features/wdise/example/SPEC.md", "features/wdise/example/PLAN.md", "features/wdise/example/TEST.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Fatalf("%s must not remain after migrate", p)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "fdf-spec.md")); err == nil {
		t.Fatal("vendored fdf-spec.md must be deleted")
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+target+`"`) {
		t.Fatalf("pin not upgraded to the target, %s:\n%s", target, idx)
	}
	// Migration scaffolds the spec copy and the five Context stubs.
	for _, p := range []string{"SPEC.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md", "DOMAIN.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("migrate did not scaffold %s", p)
		}
	}
	// The vendored spec must match the target version, not a stale one.
	spec, _ := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if !strings.Contains(string(spec), "v"+target) {
		t.Fatalf("vendored SPEC.md not the v%s spec:\n%.200s", target, spec)
	}
	// Unfilled stubs are advisory during migrate, so it still exits 0 and
	// points the user at the fdf-init interview — and warns about plain validate.
	if !strings.Contains(out.String(), "fdf-init") {
		t.Fatalf("migrate should direct the user to fdf-init:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "SURFACES.md") {
		t.Fatalf("migrate next-step must list SURFACES.md:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "F9") {
		t.Fatalf("migrate must warn that plain validate will fail F9:\n%s", out.String())
	}
	if !strings.Contains(string(idx), "(/LOG.md)") {
		t.Fatalf("root-absolute link not rewritten to /LOG.md:\n%s", idx)
	}
	// The group's listing moved to features/INDEX.md, and the root lists the
	// Features register where it stood, then the other registers.
	if !strings.Contains(string(idx), "* [FDF format](/SPEC.md) - vendored spec.\n* [Features](/features/INDEX.md) - what the software does.\n* [Changes](/changes/INDEX.md) - ") {
		t.Fatalf("the root lists the vendored SPEC.md and the Features register where the group stood:\n%s", idx)
	}
	if features := string(mustRead(t, filepath.Join(root, "features", "INDEX.md"))); !strings.Contains(features, "* [Wdise](/features/wdise/INDEX.md) - group.\n") {
		t.Fatalf("features/INDEX.md lists the group, as the root did:\n%s", features)
	}
	if !strings.Contains(string(idx), "(https://example.com/a/log.md)") {
		t.Fatalf("external URL was modified:\n%s", idx)
	}
	if !strings.Contains(string(idx), "(https://example.com/other-fdf-spec.md-notes.md)") {
		t.Fatalf("external URL containing fdf-spec.md substring was modified:\n%s", idx)
	}
	if !strings.Contains(string(idx), "(../okf/index.md)") {
		t.Fatalf("out-of-bundle relative link must stay untouched (not rewritten to INDEX.md):\n%s", idx)
	}
	feat, _ := os.ReadFile(filepath.Join(root, "features", "wdise", "example.md"))
	if !strings.Contains(string(feat), "example.spec.md") || !strings.Contains(string(feat), "example.plan.md") {
		t.Fatalf("trail links not rewritten to stem form:\n%s", feat)
	}
	if strings.Contains(string(feat), "example/SPEC.md") || strings.Contains(string(feat), "example/spec.md") {
		t.Fatalf("old nested trail links must not remain:\n%s", feat)
	}
	tst, _ := os.ReadFile(filepath.Join(root, "features", "wdise", "example.test.md"))
	if !strings.Contains(string(tst), "# Test Cases\n\n## It works\n") {
		t.Fatalf("TEST stub missing the scenario's `## It works` case after lift:\n%s", tst)
	}
}

// A bundle whose vendored SPEC.md predates the target version must have it
// refreshed, not left stale, when the pin is bumped.
func TestMigrateRefreshesStaleVendoredSpec(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV01Bundle(t, root)
	// Seed a stale root SPEC.md as if an older version had vendored it.
	stale := "---\ntype: Reference\ntitle: old\ndescription: stale.\ntimestamp: 2026-01-01T00:00:00Z\n---\n\n# Feature Document Format (FDF) — v0.2\n\nOLD VENDORED TEXT.\n"
	if err := os.WriteFile(filepath.Join(root, "SPEC.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	spec, _ := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if strings.Contains(string(spec), "OLD VENDORED TEXT") {
		t.Fatalf("stale vendored spec was not refreshed:\n%.200s", spec)
	}
	if !strings.Contains(string(spec), "v"+target) {
		t.Fatalf("refreshed spec is not v%s:\n%.200s", target, spec)
	}
}

func TestMigrateV03To10(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}

	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+target+`"`) {
		t.Fatalf("pin not upgraded to %s:\n%s", target, idx)
	}

	// Stem trail present, under features/; nested trail gone.
	for _, p := range []string{"features/wdise/example.spec.md", "features/wdise/example.plan.md", "features/wdise/example.test.md", "features/wdise/example.log.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing stem trail %s after migrate", p)
		}
	}
	for _, p := range []string{"features/wdise/example/SPEC.md", "features/wdise/example/PLAN.md", "features/wdise/example/TEST.md", "features/wdise/example/LOG.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Fatalf("nested %s must not remain", p)
		}
	}

	// Tasks stay in the feature directory.
	if _, err := os.Stat(filepath.Join(root, "features", "wdise", "example", "01-do-thing.md")); err != nil {
		t.Fatalf("task must remain under feature dir: %v", err)
	}

	// SURFACES stub scaffolded.
	if _, err := os.Stat(filepath.Join(root, "SURFACES.md")); err != nil {
		t.Fatalf("SURFACES.md must be scaffolded: %v", err)
	}

	// Feature trail links rewritten to stem form.
	feat, _ := os.ReadFile(filepath.Join(root, "features", "wdise", "example.md"))
	for _, want := range []string{"example.spec.md", "example.plan.md", "example.test.md"} {
		if !strings.Contains(string(feat), want) {
			t.Fatalf("feature missing stem link %s:\n%s", want, feat)
		}
	}
	if strings.Contains(string(feat), "example/SPEC.md") {
		t.Fatalf("old nested link remains:\n%s", feat)
	}

	// Plan task links rewritten for new plan location (sibling → slug/task).
	plan, _ := os.ReadFile(filepath.Join(root, "features", "wdise", "example.plan.md"))
	if !strings.Contains(string(plan), "example/01-do-thing.md") {
		t.Fatalf("plan task link not adjusted for stem layout:\n%s", plan)
	}

	// Root LOG.md untouched (not lifted to a stem file).
	if _, err := os.Stat(filepath.Join(root, "LOG.md")); err != nil {
		t.Fatalf("root LOG.md must remain: %v", err)
	}

	// Closing message names the Context docs still to fill — the two the
	// migration scaffolded, not the three the bundle already had — and warns
	// about F9.
	msg := out.String()
	if !strings.Contains(msg, "\nnext: run the fdf-init skill to fill SURFACES.md and DOMAIN.md.\n") {
		t.Fatalf("closing message must name the unfilled context docs:\n%s", msg)
	}
	if !strings.Contains(msg, "F9") {
		t.Fatalf("closing message must warn about F9 on plain validate:\n%s", msg)
	}

	// FreshStubsAdvisory validate already ran inside Run (exit 0).
	// Plain validate should fail F9 on the SURFACES stub while features exist.
	var plain bytes.Buffer
	if code := bundle.Validate(root, bundle.Options{Out: &plain}); code == 0 {
		t.Fatalf("plain validate should fail F9 on unfilled SURFACES stub:\n%s", plain.String())
	}
	if !strings.Contains(plain.String(), "SURFACES.md") || !strings.Contains(plain.String(), "F9") {
		t.Fatalf("plain validate should report SURFACES F9:\n%s", plain.String())
	}
}

// The pre-0.4 steps repair links with the one link engine, as fdf mv does,
// in the same pass as the move into features/ and the move of the bundle
// from docs/features to docs/fdf beside it: a link from a file that moves
// deeper gains the ../ its move needs, one whose file keeps its depth stays
// as it was, and a reference definition is repaired like an inline link.
// migrate's own rewriters left links that leave the bundle, and every
// reference definition, behind.
func TestMigrateRepairsLinksWithTheEngine(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	buildV03Bundle(t, root)
	write(t, root, "wdise/example/SPEC.md", "---\ntype: Spec\ntitle: Example spec\ndescription: Design.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Design\n\nThe handler is [refund.go](../../../../src/refund.go), beside [the map][okf].\n\n[okf]: ../../../okf/index.md\n")
	feature := filepath.Join(root, "wdise", "example.md")
	write(t, root, "wdise/example.md", string(mustRead(t, feature))+"\nIts code is [api.go](../../../src/api.go); see [the plan][plan].\n\n[plan]: example/PLAN.md\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	// The spec is lifted out of example/ and moved into features/: one
	// level up, one down.
	root = filepath.Join(filepath.Dir(root), "fdf")
	if !strings.Contains(out.String(), "the bundle is not in a git repository, so nothing can undo it") {
		t.Errorf("outside a git repository, migrate says nothing can undo it:\n%s", out.String())
	}
	spec := string(mustRead(t, filepath.Join(root, "features", "wdise", "example.spec.md")))
	for _, want := range []string{"[refund.go](../../../../src/refund.go)", "[okf]: ../../../okf/index.md"} {
		if !strings.Contains(spec, want) {
			t.Errorf("the lifted spec should hold %q:\n%s", want, spec)
		}
	}
	got := string(mustRead(t, filepath.Join(root, "features", "wdise", "example.md")))
	for _, want := range []string{"[api.go](../../../../src/api.go)", "[plan]: example.plan.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("the feature should hold %q:\n%s", want, got)
		}
	}
	if log := string(mustRead(t, filepath.Join(root, "LOG.md"))); !strings.Contains(log, "moved the bundle from `features/` to `fdf/`") {
		t.Errorf("LOG.md records the move from the directory that holds the bundle, and no machine's path:\n%s", log)
	}
}

// A bundle already at 1.0 moves nothing, however its pin is quoted: migrate
// restores only what is missing — its spec copy, the registers' indexes and
// the Context stubs — and a second run changes no byte.
func TestMigrateRepairsA10BundleInPlace(t *testing.T) {
	for _, pin := range []string{`"1.0"`, `'1.0'`, `1.0`} {
		root := t.TempDir()
		write(t, root, "INDEX.md", "---\nfdf_version: "+pin+"\n---\n\n# Bundle\n\n* [Features](/features/INDEX.md) - features.\n")
		write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-09-25\n* **Initialization**: created.\n")
		write(t, root, "features/INDEX.md", "# Features\n\n* [Onboarding](/features/onboarding.md) - feature.\n")
		write(t, root, "features/onboarding.md", "---\ntype: Feature\nstatus: draft\ntitle: Onboarding\ndescription: d.\ntimestamp: 2026-09-25T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Onboarding\n  As a user\n  I want to sign up\n  So that I can start\n```\n\n```gherkin\nScenario: It works\n  Given a\n  When b\n  Then c\n```\n")
		before := tree(t, root)
		var out bytes.Buffer
		if code := Run(Options{Root: root}, &out); code != 0 || !strings.Contains(out.String(), "nothing to migrate: the bundle already pins fdf_version 1.0") {
			t.Fatalf("pin %s: exit %d\n%s", pin, code, out.String())
		}
		if !strings.Contains(out.String(), "restored: ARCHITECTURE.md, DOMAIN.md, INFRA.md, SPEC.md, STACK.md, SURFACES.md, bugs/INDEX.md, changes/INDEX.md, debts/INDEX.md, practices/INDEX.md\n") {
			t.Errorf("pin %s: migrate restores what is missing, and says so:\n%s", pin, out.String())
		}
		after := tree(t, root)
		for _, file := range strings.SplitAfter(before, "\n== ")[1:] {
			if !strings.Contains(after, "== "+strings.TrimSuffix(file, "== ")) {
				t.Errorf("pin %s: a file that was there changed:\n%s", pin, file)
			}
		}
		out.Reset()
		if code := Run(Options{Root: root}, &out); code != 0 || !strings.Contains(out.String(), "nothing to restore") {
			t.Errorf("pin %s: a second run restores nothing: exit %d\n%s", pin, code, out.String())
		}
		if again := tree(t, root); again != after {
			t.Errorf("pin %s: a second run changes no byte", pin)
		}
	}
	// A spec copy that is not 1.0's own text is restored.
	root := copyFixture(t, "valid-v10")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 || !strings.Contains(out.String(), "restored: SPEC.md\n") {
		t.Errorf("valid-v10: exit %d\n%s", code, out.String())
	}
	// Nothing moves in a bundle at 1.0, so --to is refused there.
	out.Reset()
	before := tree(t, root)
	if code := Run(Options{Root: root, To: filepath.Join(t.TempDir(), "fdf")}, &out); code != 1 ||
		!strings.Contains(out.String(), "cannot migrate: the bundle already pins fdf_version 1.0, and migrate moves nothing in a bundle at 1.0 — move it with git mv, then point --root or FDF_ROOT_DIR at it; the bundle was left as it is.\n") {
		t.Errorf("--to on a 1.0 bundle: exit %d\n%s", code, out.String())
	}
	if tree(t, root) != before {
		t.Errorf("a refused migration changes nothing")
	}
}

// A pin migrate does not know is refused before anything is written: a
// version newer than this fdf knows, which needs a newer fdf; a 0.x version
// FDF never had; and a pin that is not a MAJOR.MINOR version at all.
func TestMigrateRefusesAPinItDoesNotKnow(t *testing.T) {
	notAVersion := func(pin string) string {
		return "cannot migrate: the bundle pins fdf_version " + pin + ", which is not a MAJOR.MINOR version such as " + scaffold.CurrentVersion() + " — correct the pin in INDEX.md; the bundle was left as it is."
	}
	for _, tc := range []struct{ pin, says string }{
		{`"1.1"`, "cannot migrate: the bundle pins fdf_version 1.1, newer than any spec this binary knows (1.0) — upgrade fdf; the bundle was left as it is."},
		{`"2.0"`, "cannot migrate: the bundle pins fdf_version 2.0, newer than any spec this binary knows (1.0) — upgrade fdf; the bundle was left as it is."},
		{`"0.8"`, "cannot migrate: the bundle pins fdf_version 0.8, which is no 0.x version this binary knows (0.1 to 0.7) — correct the pin in INDEX.md; the bundle was left as it is."},
		{`"1.0.0"`, notAVersion("1.0.0")}, {`"v1.0"`, notAVersion("v1.0")},
	} {
		root := t.TempDir()
		write(t, root, "INDEX.md", "---\nfdf_version: "+tc.pin+"\n---\n\n# Bundle\n\n* [Venues](/venues/INDEX.md) - group.\n")
		write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-09-25\n* **Initialization**: created.\n")
		write(t, root, "venues/INDEX.md", "# Venues\n")
		before := tree(t, root)
		var out bytes.Buffer
		if code := Run(Options{Root: root}, &out); code != 1 || !strings.Contains(out.String(), tc.says) {
			t.Errorf("pin %s is refused: exit %d, want 1 saying %q:\n%s", tc.pin, code, tc.says, out.String())
		}
		if after := tree(t, root); after != before {
			t.Errorf("pin %s: a refused migration changes nothing", tc.pin)
		}
	}
}

// migrate changes what it must, and keeps the rest of each file's form: a
// root INDEX.md whose frontmatter carries no pin gets it in that
// frontmatter, not a second block above it; a file written with CRLF line
// endings keeps them on every line, the lines migrate adds included, and a
// CRLF log takes the migration's entry above its older days; and a document
// that is a symlink moves as the link it is, without migrate writing through
// it into the file it names.
func TestMigrateKeepsTheFormOfWhatItRewrites(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "handbook")
	buildV03Bundle(t, root)
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	write(t, root, "INDEX.md", crlf("---\ntitle: Handbook\n---\n\n# Handbook\n\n* [Wdise](/wdise/INDEX.md) - group.\n"))
	write(t, root, "LOG.md", crlf("# Bundle Update Log\n\n## 2026-07-06\n* **Initialization**: v0.3.\n"))
	shared := "---\ntype: Feature\nstatus: draft\ntitle: Shared\ndescription: d.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Shared\n  As a user\n  I want it\n  So that it helps\n```\n\n```gherkin\nScenario: It is shared\n  Given a\n  When b\n  Then c\n```\n\nSee [the example](example.md) and `wdise/example`.\n"
	write(t, dir, "shared.md", shared)
	if err := os.Symlink(filepath.Join(dir, "shared.md"), filepath.Join(root, "wdise", "shared.md")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	idx := string(mustRead(t, filepath.Join(root, "INDEX.md")))
	if !strings.HasPrefix(idx, "---\r\nfdf_version: \""+target+"\"\r\ntitle: Handbook\r\n---\r\n") {
		t.Errorf("the pin joins the frontmatter INDEX.md has:\n%q", idx)
	}
	for _, rel := range []string{"INDEX.md", "LOG.md"} {
		if text := string(mustRead(t, filepath.Join(root, rel))); strings.Count(text, "\n") != strings.Count(text, "\r\n") {
			t.Errorf("%s keeps its CRLF line endings, on every line:\n%q", rel, text)
		}
	}
	if log := string(mustRead(t, filepath.Join(root, "LOG.md"))); strings.Index(log, "**Migrated**") > strings.Index(log, "## 2026-07-06") {
		t.Errorf("the migration's entry goes above the older days:\n%q", log)
	}
	if s := string(mustRead(t, filepath.Join(root, "features", "INDEX.md"))); strings.Contains(s, "\r") {
		t.Errorf("features/INDEX.md, which migrate writes new, ends every line alike:\n%q", s)
	}
	if fi, err := os.Lstat(filepath.Join(root, "features", "wdise", "shared.md")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symlink moves with its group, as a symlink: %v", err)
	}
	if got := string(mustRead(t, filepath.Join(dir, "shared.md"))); got != shared {
		t.Errorf("migrate writes nothing through a symlink:\n%s", got)
	}
}

// A bundle that is a symbolic link is not where the link is: migrate refuses
// it, names the directory it links to, and writes nothing in it.
func TestMigrateRefusesABundleThatIsASymlink(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "shared", "handbook")
	buildV03Bundle(t, bundle)
	link := filepath.Join(dir, "docs", "features")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(bundle, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	before := tree(t, bundle)
	var out bytes.Buffer
	if code := Run(Options{Root: link}, &out); code != 1 || !strings.Contains(out.String(), "  "+link+": a symbolic link to "+bundle+" — migrate the directory it names, with --root\n") {
		t.Fatalf("a bundle that is a symlink is refused: exit %d\n%s", code, out.String())
	}
	if tree(t, bundle) != before {
		t.Error("migrate wrote into the directory the link names")
	}
}

// migrate writes the root's INDEX.md, LOG.md and SPEC.md whatever they hold,
// so one that is a symbolic link is refused, and the file it names keeps its
// words.
func TestMigrateRefusesARootFileThatIsASymlink(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "handbook")
	buildV03Bundle(t, root)
	log := string(mustRead(t, filepath.Join(root, "LOG.md")))
	write(t, dir, "shared/LOG.md", log)
	if err := os.Remove(filepath.Join(root, "LOG.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "shared", "LOG.md"), filepath.Join(root, "LOG.md")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 1 || !strings.Contains(out.String(), "  LOG.md: a symbolic link to ../shared/LOG.md, which migrate would write through — replace it with the file it names\n") {
		t.Fatalf("a root LOG.md that is a symlink is refused: exit %d\n%s", code, out.String())
	}
	if got := string(mustRead(t, filepath.Join(dir, "shared", "LOG.md"))); got != log {
		t.Errorf("the file the link names keeps its words:\n%s", got)
	}
}

// migrate never writes over a file its plan did not read: a file it would
// write new that is there after all stops it, and keeps its words.
func TestMigrateNeverWritesOverAFileItDidNotRead(t *testing.T) {
	root := t.TempDir()
	buildV03Bundle(t, root)
	p, problems, err := newPlan(root, "0.3", "", root)
	if err != nil || len(problems) > 0 {
		t.Fatalf("newPlan: %v %v", err, problems)
	}
	p.texts["NOTES.md"] = "migrate's\n"
	write(t, root, "NOTES.md", "mine\n")
	if err := p.apply(); err == nil || !os.IsExist(err) {
		t.Errorf("a file the plan writes new that is there stops it: %v", err)
	}
	if got := string(mustRead(t, filepath.Join(root, "NOTES.md"))); got != "mine\n" {
		t.Errorf("the file keeps its words: %q", got)
	}
}

// copyFixture copies a conformance fixture's bundle into a temp dir.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	dst := t.TempDir()
	copyFixtureTo(t, name, dst)
	return dst
}

// copyFixtureTo copies a conformance fixture's bundle to dst.
func copyFixtureTo(t *testing.T, name, dst string) {
	t.Helper()
	src := filepath.Join("..", "..", "..", "testdata", name, "bundle")
	filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(src, p)
			write(t, dst, rel, string(mustRead(t, p)))
		}
		return nil
	})
}

// A register's or a group's INDEX.md pins nothing, so --root
// docs/fdf/features used to be read as a bundle with no pin, and migrate
// built a 0.7 bundle inside the register. A root inside a pinned bundle is
// refused, in the words every command uses for it, and nothing changes.
func TestMigrateRefusesARootInsideABundle(t *testing.T) {
	v03 := t.TempDir()
	buildV03Bundle(t, v03)
	for _, tc := range []struct{ bundle, root string }{
		{copyFixture(t, "valid-v10"), "features"},
		{v03, "wdise"},
	} {
		root := filepath.Join(tc.bundle, tc.root)
		before := tree(t, tc.bundle)
		var out bytes.Buffer
		want := "error: " + root + " is inside the bundle at " + tc.bundle + ", not a bundle of its own — pass --root " + tc.bundle + ", or leave --root out\n"
		if code := Run(Options{Root: root}, &out); code != 1 || out.String() != want {
			t.Errorf("fdf migrate --root %s: exit %d\n got: %q\nwant: %q", root, code, out.String(), want)
		}
		if after := tree(t, tc.bundle); after != before {
			t.Errorf("a refused migration changes nothing:\nbefore:\n%s\nafter:\n%s", before, after)
		}
	}
}

// tree lists every file under root with its content, and every symbolic
// link with what it names, in path order.
func tree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if to, err := os.Readlink(p); err == nil {
			fmt.Fprintf(&b, "== %s -> %s\n", filepath.ToSlash(rel), to)
		} else {
			fmt.Fprintf(&b, "== %s\n%s", filepath.ToSlash(rel), mustRead(t, p))
		}
		return nil
	})
	return b.String()
}

// A root with no bundle in it is refused before anything is written: a
// mistyped --root used to fill a stray directory with stubs and indexes.
func TestMigrateRefusesARootWithNoBundle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-a-bundle")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{root, filepath.Join(root, "missing")} {
		var out bytes.Buffer
		if code := Run(Options{Root: r}, &out); code != 1 || !strings.Contains(out.String(), "no bundle at "+r+" (no INDEX.md)") {
			t.Fatalf("%s: exit %d\n%s", r, code, out.String())
		}
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("a refused migration writes nothing, found %d entries", len(entries))
	}
}

// Half-migrated or hand-mixed layout: nested trail still present AND the
// stem destination already exists. collectTrailMoves must abort before any
// rename so migrate never partially applies destructive moves.
func TestMigrateAbortsWhenStemTrailExists(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	// Conflict: nested SPEC still present, stem sibling already there.
	write(t, root, "wdise/example.spec.md", "---\ntype: Spec\ntitle: Conflict\ndescription: Pre-existing stem trail.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Design\n\nSTALE STEM.\n")

	nestedSpec := filepath.Join(root, "wdise", "example", "SPEC.md")
	nestedBody, err := os.ReadFile(nestedSpec)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	code := Run(Options{Root: root}, &out)
	if code == 0 {
		t.Fatalf("expected non-zero exit when stem destination exists; output:\n%s", out.String())
	}
	msg := out.String()
	// The pre-flight scan catches the stem-named file before anything runs.
	if !strings.Contains(msg, "cannot migrate") || !strings.Contains(msg, "wdise/example.spec.md") {
		t.Fatalf("error should pre-flight the conflicting stem file (cannot migrate + path):\n%s", msg)
	}
	// No trail moves applied.
	if strings.Contains(msg, "moved ") {
		t.Fatalf("must not apply trail moves after destination conflict:\n%s", msg)
	}
	// Nested trail still present (no partial destructive apply).
	for _, p := range []string{
		"wdise/example/SPEC.md",
		"wdise/example/PLAN.md",
		"wdise/example/TEST.md",
		"wdise/example/LOG.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("nested %s must remain after aborted migrate: %v", p, err)
		}
	}
	after, err := os.ReadFile(nestedSpec)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(nestedBody) {
		t.Fatalf("nested SPEC.md content changed on abort:\n%s", after)
	}
	// Pin must not advance past the conflict abort (steps after lift must not run).
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.3"`) {
		t.Fatalf("pin must remain 0.3 when trail lift aborts:\n%s", idx)
	}
}

// requireUntouched asserts pre-flight aborts modified nothing: pin unchanged
// and the given paths still present.
func requireUntouched(t *testing.T, root, wantPin string, paths ...string) {
	t.Helper()
	idx, err := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), wantPin) {
		t.Fatalf("pin must be unchanged (%s):\n%s", wantPin, idx)
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("%s must remain after pre-flight abort: %v", p, err)
		}
	}
}

func TestMigratePreflightsDraftFeatureLog(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	write(t, root, "wdise/pending.md", "---\ntype: Feature\ntitle: Pending\ndescription: A draft.\nstatus: draft\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Pending\n```\n\n```gherkin\nScenario: Later\n  Given a\n  When b\n  Then c\n```\n")
	write(t, root, "wdise/pending/LOG.md", "# Pending log\n\n## 2026-07-06\n* Sketched.\n")

	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code == 0 {
		t.Fatalf("expected pre-flight abort for draft feature log:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "wdise/pending/LOG.md") || !strings.Contains(out.String(), "draft") {
		t.Fatalf("abort should name the draft feature log:\n%s", out.String())
	}
	requireUntouched(t, root, `fdf_version: "0.3"`, "wdise/pending/LOG.md", "wdise/example/SPEC.md")
}

func TestMigratePreflightsDottedFeatureSlug(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	write(t, root, "wdise/api.v2.md", "---\ntype: Feature\ntitle: API v2\ndescription: Dotted slug.\nstatus: draft\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: API v2\n```\n\n```gherkin\nScenario: One\n  Given a\n  When b\n  Then c\n```\n")

	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code == 0 {
		t.Fatalf("expected pre-flight abort for dotted slug:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "wdise/api.v2.md") {
		t.Fatalf("abort should name the dotted feature file:\n%s", out.String())
	}
	requireUntouched(t, root, `fdf_version: "0.3"`, "wdise/api.v2.md", "wdise/example/SPEC.md")
}

func TestMigratePreflightsStrayFileInFeatureDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	write(t, root, "wdise/example/INDEX.md", "# stray index\n\n* [x](/wdise/example.md)\n")

	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code == 0 {
		t.Fatalf("expected pre-flight abort for stray feature-dir file:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "wdise/example/INDEX.md") {
		t.Fatalf("abort should name the stray file:\n%s", out.String())
	}
	requireUntouched(t, root, `fdf_version: "0.3"`, "wdise/example/INDEX.md", "wdise/example/SPEC.md")
}

// A second run finds the bundle at 1.0, which it leaves as it is: the repair
// path restores nothing a migration wrote, so running migrate twice in a row
// cannot flip from success to failure.
func TestMigrateIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	var out1 bytes.Buffer
	if code := Run(Options{Root: root}, &out1); code != 0 {
		t.Fatalf("first migrate exit %d\n%s", code, out1.String())
	}
	migrated := tree(t, root)
	var out2 bytes.Buffer
	if code := Run(Options{Root: root}, &out2); code != 0 {
		t.Fatalf("second migrate must stay exit 0 (idempotent), got %d\n%s", code, out2.String())
	}
	if tree(t, root) != migrated {
		t.Fatalf("a second migrate changes no byte:\n%s", out2.String())
	}
}

func TestMigrateReadsUnquotedPin(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	write(t, root, "INDEX.md", "---\nfdf_version: "+target+"\n---\n\n# Bundle\n\n* [log](/LOG.md) - history.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* Init.\n")

	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "already pins fdf_version "+target) {
		t.Fatalf("unquoted pin must hit the already-current path:\n%s", out.String())
	}
}

// A migrate that finds the pin already current looks identical to a migrate
// that is simply too old to know about newer versions — which is what happens
// when a version shim (mise, asdf) holds an old fdf in a directory. The
// message must name the binary so the user can tell those apart.
func TestNoOpMigrateNamesTheBinaryVersion(t *testing.T) {
	root := t.TempDir()
	write(t, root, "INDEX.md", "---\nfdf_version: \""+target+"\"\n---\n\n# Bundle\n")
	write(t, root, "LOG.md", "# Log\n\n## 2026-09-15\n* init.\n")
	Version = "9.9.9-test"
	defer func() { Version = "" }()

	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out.String())
	}
	msg := out.String()
	if !strings.Contains(msg, "nothing to migrate") {
		t.Errorf("should say plainly that nothing was migrated:\n%s", msg)
	}
	if !strings.Contains(msg, "fdf 9.9.9-test") {
		t.Errorf("should name the binary doing the looking:\n%s", msg)
	}
	if !strings.Contains(msg, "upgrade fdf") {
		t.Errorf("should point at upgrading when a newer spec exists:\n%s", msg)
	}
	if strings.Contains(msg, "validating migrated bundle") {
		t.Errorf("nothing was migrated; must not claim it was:\n%s", msg)
	}
}

// A migration's only evidence used to be a scroll of per-file lines. The
// plan states the version transition and how much each step moves, so "it
// did nothing" is distinguishable from "it did a lot".
func TestMigrateSummaryReportsTransitionAndCount(t *testing.T) {
	root := t.TempDir()
	buildV03Bundle(t, root)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out.String())
	}
	for _, want := range []string{
		"plan: fdf_version 0.3 → 1.0\n",
		"  layout     4 trail files lifted (0.3)\n",
		"  features   wdise/ → features/   (1 feature, 7 files)\n",
		"  move    wdise/ → features/wdise/  (7 files)\n",
		"  move    wdise/example/SPEC.md → features/wdise/example.spec.md\n",
		"\ndone: migrated the bundle at " + root + " to fdf_version 1.0; logged in LOG.md.\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should say %q:\n%s", want, out.String())
		}
	}
}

// buildV06Bundle writes a filled v0.6 bundle: five Context documents, a
// lexicon banning "shop", one delivered feature whose spec still says "shop",
// and a grouped debt.
func buildV06Bundle(t *testing.T, root string) {
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.6\"\n---\n\n# Bundle\n\n* [Venues](/venues/INDEX.md) - group.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-09-16\n* **Initialization**: v0.6.\n")
	for _, name := range []string{"STACK", "ARCHITECTURE", "SURFACES", "INFRA"} {
		write(t, root, name+".md", "---\ntype: Context\ntitle: "+name+"\ndescription: Filled.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# "+name+"\n\nFilled.\n")
	}
	write(t, root, "DOMAIN.md", "---\ntype: Context\ntitle: Domain\ndescription: Filled.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Terms\n\n## Venue\nA physical location where a merchant sells.\n- instead-of: shop\n")
	write(t, root, "venues/INDEX.md", "# Venues\n\n* [Hours](/venues/hours.md) - opening hours.\n")
	write(t, root, "venues/hours.md", "---\ntype: Feature\nstatus: specified\ntitle: Hours\ndescription: Opening hours.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Hours\n  As a Venue owner\n  I want hours\n  So that people know\n```\n\n# Scenarios\n\n```gherkin\nScenario: Owner sets hours\n  Given a Venue\n  When the owner sets hours\n  Then they show\n```\n")
	write(t, root, "venues/hours.spec.md", "---\ntype: Spec\ntitle: Design\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Approach\n\nEach shop keeps its own hours.\n")
	write(t, root, "debts/INDEX.md", "# Debt\n\n* [x](/debts/INDEX.md) - register.\n")
	write(t, root, "debts/backend/slow-hours.md", "---\ntype: Debt\nstatus: open\ntitle: Slow\ndescription: d.\nresource: []\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Gap\n\nHours are read twice.\n")
	write(t, root, "practices/INDEX.md", "# Practices\n\n* [x](/practices/INDEX.md) - practices.\n")
	write(t, root, "changes/INDEX.md", "# Changes\n\n* [x](/changes/INDEX.md) - changes.\n")
}

func TestMigrateV06To10(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "1.0"`) {
		t.Fatalf("pin not upgraded:\n%s", idx)
	}
	if _, err := os.Stat(filepath.Join(root, "bugs", "INDEX.md")); err != nil {
		t.Fatalf("bugs/INDEX.md not scaffolded: %v", err)
	}
	spec, _ := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if !strings.Contains(string(spec), "Feature Document Format (FDF) — v1.0") {
		t.Fatal("SPEC.md not re-vendored to v1.0")
	}
	for _, want := range []string{
		"plan: fdf_version 0.6 → 1.0",
		"the domain language now reaches every document and name — 1 banned word(s) in 1 file(s).",
		"of the 1 debt(s) on the register",
		"`fdf mv debts/<id> bugs/<id>`",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q\n%s", want, out.String())
		}
	}
	// The feature group moved into features/; the debt stayed in its register.
	for _, p := range []string{"features/venues/hours.md", "features/venues/hours.spec.md", "debts/backend/slow-hours.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

// Under a 0.6 pin bugs/ is a feature group, and 0.7's migrate refused a
// bundle that had one. It now moves into features/ with the other groups,
// and bugs/ becomes the bug register; its listing moves too, and every
// mention of its features' IDs follows.
func TestMigrateMovesA06BugsGroupIntoFeatures(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.6\"\n---\n\n# Bundle\n\n* [Venues](/venues/INDEX.md) - group.\n* [Bugs](/bugs/INDEX.md) - the tracker.\n")
	write(t, root, "bugs/INDEX.md", "# Bugs\n\n* [Tracker](/bugs/tracker.md) - a feature group named bugs.\n")
	write(t, root, "bugs/tracker.md", "---\ntype: Feature\nstatus: draft\ntitle: Tracker\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Tracker\n  As a user\n  I want it\n  So that it helps\n```\n\n# Scenarios\n\n```gherkin\nScenario: It works\n  Given it\n  When it runs\n  Then it works\n```\n")
	write(t, root, "venues/hours.spec.md", string(mustRead(t, filepath.Join(root, "venues", "hours.spec.md")))+"\nA closed Venue shows on `bugs/tracker`.\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for rel, want := range map[string]string{
		"features/bugs/tracker.md":      "title: Tracker",
		"features/bugs/INDEX.md":        "* [Tracker](/features/bugs/tracker.md) - a feature group named bugs.",
		"features/INDEX.md":             "* [Venues](/features/venues/INDEX.md) - group.\n* [Bugs](/features/bugs/INDEX.md) - the tracker.\n",
		"bugs/INDEX.md":                 "Known defects",
		"INDEX.md":                      "* [Features](/features/INDEX.md) - what the software does.\n",
		"features/venues/hours.spec.md": "A closed Venue shows on `features/bugs/tracker`.",
	} {
		if got := string(mustRead(t, filepath.Join(root, rel))); !strings.Contains(got, want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, got)
		}
	}
	if idx := string(mustRead(t, filepath.Join(root, "INDEX.md"))); !strings.Contains(idx, "* [Bugs](/bugs/INDEX.md) - known defects not repaired yet.") || strings.Contains(idx, "the tracker") {
		t.Errorf("the root lists the bug register, and no longer the group:\n%s", idx)
	}
}

// The fdf-init hint follows the validator's stub messages, not the word
// "stub": a debt named stub-gateway in a filled bundle asks for nothing, and
// an unfilled DOMAIN.md is the one stub named.
func TestMigrateNamesOnlyTheUnfilledStubs(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	// No description: validation warns, naming the file.
	write(t, root, "debts/stub-gateway.md", "---\ntype: Debt\nstatus: open\ntitle: Stub gateway\nresource: []\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Gap\n\nThe gateway is a stand-in.\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "debts/stub-gateway.md: missing recommended `description`") {
		t.Fatalf("the test needs a validation line naming the debt:\n%s", out.String())
	}
	if strings.Contains(out.String(), "fdf-init") {
		t.Errorf("every Context document is filled; nothing asks for fdf-init:\n%s", out.String())
	}

	// From v0.5, DOMAIN.md arrives as a stub, and the bundle has a feature.
	// A 0.5 bundle has no practices/ or debts/ register.
	root = t.TempDir()
	buildV06Bundle(t, root)
	os.Remove(filepath.Join(root, "DOMAIN.md"))
	os.RemoveAll(filepath.Join(root, "debts"))
	os.RemoveAll(filepath.Join(root, "practices"))
	write(t, root, "INDEX.md", strings.Replace(string(mustRead(t, filepath.Join(root, "INDEX.md"))), `"0.6"`, `"0.5"`, 1))
	out.Reset()
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for _, want := range []string{
		"\nnext: run the fdf-init skill to fill DOMAIN.md.\n",
		"warning: the next plain `fdf validate` will fail F9 until those stubs are filled",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should say %q:\n%s", want, out.String())
		}
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Older tools wrote a status tag after each index listing and nothing kept it
// current; migration removes it, leaves other bold text and code alone, and
// logs the migration in the bundle-root log.
func TestMigrateDropsIndexStatusTags(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "venues/INDEX.md", "# Venues\n\n* [Hours](/venues/hours.md) - opening hours. (**draft**)\n* [Menu](/venues/hours.md) - the menu. (**important**)\n\n```\n* [Sample](/venues/hours.md) - a sample. (**done**)\n```\n")
	write(t, root, "changes/INDEX.md", "# Changes\n\n* [x](/changes/INDEX.md) - changes. (**specified**)\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	venues, _ := os.ReadFile(filepath.Join(root, "features", "venues", "INDEX.md"))
	for _, want := range []string{
		"* [Hours](/features/venues/hours.md) - opening hours.\n",
		"* [Menu](/features/venues/hours.md) - the menu. (**important**)\n",
		"* [Sample](/venues/hours.md) - a sample. (**done**)\n",
	} {
		if !strings.Contains(string(venues), want) {
			t.Errorf("features/venues/INDEX.md should contain %q:\n%s", want, venues)
		}
	}
	changes, _ := os.ReadFile(filepath.Join(root, "changes", "INDEX.md"))
	if strings.Contains(string(changes), "(**specified**)") {
		t.Errorf("changes/INDEX.md keeps its tag:\n%s", changes)
	}
	log, _ := os.ReadFile(filepath.Join(root, "LOG.md"))
	for _, want := range []string{"* **Migrated**: fdf_version 0.6 → 1.0 with `fdf migrate`: moved `venues/` into `features/` (1 feature)", "Removed the status tag from 2 index listing(s)"} {
		if !strings.Contains(string(log), want) {
			t.Errorf("LOG.md should record the migration, %q:\n%s", want, log)
		}
	}
	if !strings.Contains(out.String(), "; 2 status tags removed\n") {
		t.Errorf("output should count the tags:\n%s", out.String())
	}
}

// v0.7 matches a test case exactly: a `## <scenario name>` heading under
// `# Test Cases`. A 0.6 bundle that lists its cases as bullets migrates, but
// its validation then fails F8, and migrate says what to rewrite by hand —
// and which features still owe a surface decision.
func TestMigrateV06To10ReportsBulletCasesAndSurfaces(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "venues/hours.md", "---\ntype: Feature\nstatus: planned\ntitle: Hours\ndescription: Opening hours.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Hours\n  As a Venue owner\n  I want hours\n  So that people know\n```\n\n# Scenarios\n\n```gherkin\nScenario: Owner sets hours\n  Given a Venue\n  When the owner sets hours\n  Then they show\n```\n")
	write(t, root, "venues/hours.plan.md", "---\ntype: Plan\ntitle: Plan\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Tasks\n")
	write(t, root, "venues/hours.test.md", "---\ntype: Test\ntitle: Tests\ndescription: d.\ntimestamp: 2026-09-16 09:30\n---\n\n# Test Cases\n\n- Scenario: Owner sets hours — `go test ./... -run TestHours`\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 1 {
		t.Fatalf("a bundle whose cases are bullets fails F8 once migrated; want exit 1, got %d\n%s", code, out.String())
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "1.0"`) {
		t.Fatalf("the migration itself still happens:\n%s", idx)
	}
	for _, want := range []string{
		`features/venues/hours.test.md: scenario "Owner sets hours" has no test case`,
		"1 scenario(s) have none (F8). Rewrite those test documents' cases as headings, by hand",
		"1 feature(s) have no slug.surface.md",
		"1 timestamp(s) are neither a date nor an RFC 3339 time with Z or an offset (F1)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q\n%s", want, out.String())
		}
	}
}

// buildV07Bundle copies valid-bugs-v07, a conformant 0.7 bundle with a
// feature group venues/, a Change, Fixes and bugs that name its feature,
// and adds what the move into features/ has to carry or leave alone.
func buildV07Bundle(t *testing.T) string {
	root := copyFixture(t, "valid-bugs-v07")
	feature := filepath.Join(root, "venues", "opening-hours.md")
	write(t, root, "venues/opening-hours.md", string(mustRead(t, feature))+
		"\nIts bugs are [on the register](../bugs/hours-off-by-one.md), a fix is [in changes](/changes/old-fix.md),\n"+
		"its code is [beside the bundle](../../src/hours.go), and its spec is [here][spec].\n"+
		"Read its history with `fdf history venues/opening-hours`.\n\n[spec]: opening-hours.spec.md\n\n"+
		"Its handler is venues/opening-hours/handler.go, served as `GET /venues/opening-hours`.\n\n"+
		"```sh\nfdf new venues/opening-hours\n```\n")
	task := filepath.Join(root, "venues", "opening-hours", "01-build.md")
	write(t, root, "venues/opening-hours/01-build.md", strings.Replace(string(mustRead(t, task)), "status: done\n", "status: done\nresource: [venues/opening-hours/api.go]\n", 1))
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-09-23\n* **Initialization**: created venues/opening-hours; see [it](venues/opening-hours.md).\n")
	write(t, root, "venues/diagram.png", "PNG")
	write(t, root, "assets/logo.png", "PNG")
	write(t, root, ".obsidian/notes.md", "venues/opening-hours\n")
	write(t, root, "menus/daily.md", "---\ntype: Feature\nstatus: draft\ntitle: Daily menu\ndescription: d.\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Daily menu\n  As a Venue owner\n  I want a daily menu\n  So that people know\n```\n\n```gherkin\nScenario: Owner posts the menu\n  Given a Venue\n  When the owner posts it\n  Then it shows\n```\n")
	return root
}

// A 0.7 bundle's feature groups move into features/ with everything in them,
// and every reference follows. Each mention of a feature's ID gains
// features/, in frozen documents too, while logs keep their words and a
// resource path stays a path; the engine repairs every link; and the root
// lists the Features register where the group's listing stood. A group the
// root never listed gets a listing, which links its directory when it has no
// index. A directory that holds no Markdown, and a hidden one, stay where
// they are.
func TestMigrateMovesFeatureGroupsIntoFeatures(t *testing.T) {
	root := buildV07Bundle(t)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for _, p := range []string{
		"features/venues/INDEX.md", "features/venues/opening-hours.md", "features/venues/opening-hours.spec.md",
		"features/venues/opening-hours.plan.md", "features/venues/opening-hours.test.md",
		"features/venues/opening-hours/01-build.md", "features/venues/diagram.png",
		"features/menus/daily.md", "assets/logo.png", ".obsidian/notes.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	for _, p := range []string{"venues", "menus"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Errorf("%s/ moved into features/, and is not left behind", p)
		}
	}
	for rel, wants := range map[string][]string{
		"features/venues/opening-hours.md": {
			"[on the register](../../bugs/hours-off-by-one.md)", "[in changes](/changes/old-fix.md)",
			"[beside the bundle](../../../src/hours.go)", "[spec]: opening-hours.spec.md",
			"`fdf history features/venues/opening-hours`",
			// A path that goes on past an ID, a bare /<id> and a code
			// block keep their words.
			"Its handler is venues/opening-hours/handler.go, served as `GET /venues/opening-hours`.",
			"```sh\nfdf new venues/opening-hours\n```",
		},
		"changes/closed-hours-fix.md":               {"affects: features/venues/opening-hours\n", "## features/venues/opening-hours\n"},
		"bugs/hours-off-by-one.md":                  {"affects: features/venues/opening-hours\n", "## features/venues/opening-hours\n"},
		"features/venues/opening-hours/01-build.md": {"resource: [venues/opening-hours/api.go]\n"},
		"LOG.md":             {"* **Initialization**: created venues/opening-hours; see [it](features/venues/opening-hours.md).\n"},
		".obsidian/notes.md": {"venues/opening-hours\n"},
		"INDEX.md":           {"# Example — Feature Bundle\n\n* [Features](/features/INDEX.md) - what the software does.\n* [Changes](/changes/INDEX.md) - "},
		"features/INDEX.md":  {"* [Venues](/features/venues/INDEX.md) - venues.\n* [Menus](/features/menus/) - features in menus.\n"},
	} {
		got := string(mustRead(t, filepath.Join(root, rel)))
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Errorf("%s should hold %q:\n%s", rel, want, got)
			}
		}
	}
	for _, want := range []string{
		"  features   menus/, venues/ → features/   (2 features, 8 files)\n",
		"  ids        9 mentions in 5 documents; logs keep their words (1)\n",
		"  indexes    features/INDEX.md: 1 listing moved from INDEX.md, 1 listing written for groups INDEX.md did not list\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the plan should say %q:\n%s", want, out.String())
		}
	}
}

// Every old pin goes to 1.0 in one run, and the bundle that comes out is
// conformant: migrate's own validation, which reads an unfilled Context
// stub as a warning, passes on a valid bundle of each 0.x version.
func TestMigrateEveryOldPinTo10(t *testing.T) {
	v01 := t.TempDir()
	buildV01Bundle(t, v01)
	for pin, root := range map[string]string{
		"0.1": v01,
		"0.2": copyFixture(t, "valid-minimal"),
		"0.3": copyFixture(t, "context-docs-no-features"),
		"0.4": copyFixture(t, "valid-minimal-v04"),
		"0.5": copyFixture(t, "valid-retired-v05"),
		"0.6": copyFixture(t, "valid-domain-v06"),
		"0.7": copyFixture(t, "valid-bugs-v07"),
	} {
		var out bytes.Buffer
		if code := Run(Options{Root: root}, &out); code != 0 || !strings.Contains(out.String(), "plan: fdf_version "+pin+" → 1.0\n") {
			t.Errorf("%s: exit %d\n%s", pin, code, out.String())
		}
		if got := readPin(root); got != target {
			t.Errorf("%s: pins %q after migrate", pin, got)
		}
	}
}

// A dry run prints the whole plan, as the migration itself prints it, and
// changes no byte: on a 0.1 bundle, which every step changes, and on a 0.7
// one.
func TestMigrateDryRunChangesNothing(t *testing.T) {
	v01 := t.TempDir()
	buildV01Bundle(t, v01)
	for _, root := range []string{v01, buildV07Bundle(t)} {
		before := tree(t, root)
		var dry bytes.Buffer
		if code := Run(Options{Root: root, DryRun: true}, &dry); code != 0 || !strings.Contains(dry.String(), " (dry run: nothing changed)\n") {
			t.Fatalf("dry run: exit %d\n%s", code, dry.String())
		}
		if after := tree(t, root); after != before {
			t.Fatalf("a dry run changes no byte:\n%s", dry.String())
		}
		var real bytes.Buffer
		if code := Run(Options{Root: root}, &real); code != 0 {
			t.Fatalf("migrate exit %d\n%s", code, real.String())
		}
		if plan := strings.Replace(dry.String(), " (dry run: nothing changed)", "", 1); !strings.HasPrefix(real.String(), plan) {
			t.Errorf("the migration prints the plan the dry run printed:\ndry run:\n%s\nmigration:\n%s", dry.String(), real.String())
		}
	}
}

// A 0.x feature group may be called features; it moves into the register of
// that name, and its features' IDs gain features/ like any other.
func TestMigrateMovesAGroupNamedFeatures(t *testing.T) {
	root := buildV07Bundle(t)
	if err := os.Rename(filepath.Join(root, "menus"), filepath.Join(root, "features")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "features/INDEX.md", "# Features\n\n* [Daily menu](/features/daily.md) - the daily menu.\n")
	write(t, root, "changes/old-fix.md", string(mustRead(t, filepath.Join(root, "changes", "old-fix.md")))+"\nSee `features/daily`.\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for rel, want := range map[string]string{
		"features/features/daily.md": "title: Daily menu",
		"features/features/INDEX.md": "* [Daily menu](/features/features/daily.md) - the daily menu.",
		"features/INDEX.md":          "* [Features](/features/features/INDEX.md) - features in features.",
		"changes/old-fix.md":         "See `features/features/daily`.",
	} {
		if got := string(mustRead(t, filepath.Join(root, rel))); !strings.Contains(got, want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, got)
		}
	}
}

// A scenario's name is matched, word for word, by its test case, by the
// Fixes that declare it and by the bugs that cite it. When the name holds a
// feature's ID, every copy gains features/, the Gherkin's Scenario line with
// them, and the bundle still validates. So does a test stub migrate writes.
func TestMigrateRewritesAnIDInAScenarioNameEverywhere(t *testing.T) {
	root := copyFixture(t, "valid-bugs-v07")
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(p, ".md") {
			raw := string(mustRead(t, p))
			if s := strings.ReplaceAll(raw, "Venue owner sets opening hours", "Venue owner sets venues/opening-hours"); s != raw {
				write(t, root, relSlash(root, p), s)
			}
		}
		return nil
	})
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for rel, want := range map[string]string{
		"features/venues/opening-hours.md":      "Scenario: Venue owner sets features/venues/opening-hours\n",
		"features/venues/opening-hours.test.md": "## Venue owner sets features/venues/opening-hours\n",
		"changes/closed-hours-fix.md":           "- Venue owner sets features/venues/opening-hours — ",
		"bugs/hours-off-by-one.md":              "- Venue owner sets features/venues/opening-hours — ",
	} {
		if got := string(mustRead(t, filepath.Join(root, rel))); !strings.Contains(got, want) {
			t.Errorf("%s should hold %q:\n%s", rel, want, got)
		}
	}

	root = filepath.Join(t.TempDir(), "handbook")
	buildV03Bundle(t, root)
	if err := os.Remove(filepath.Join(root, "wdise", "example", "TEST.md")); err != nil {
		t.Fatal(err)
	}
	feature := string(mustRead(t, filepath.Join(root, "wdise", "example.md")))
	write(t, root, "wdise/example.md", strings.Replace(feature, "Scenario: It works", "Scenario: It works for wdise/example", 1))
	out.Reset()
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if s := string(mustRead(t, filepath.Join(root, "features", "wdise", "example.test.md"))); !strings.Contains(s, "## It works for features/wdise/example\n") {
		t.Errorf("the test stub names the scenario as its Gherkin does:\n%s", s)
	}
}

// A releases/ with no index gets none, since fdf release writes it with the
// first release it cuts, and the root lists the register only once it has
// one: never a link to an index that is not there.
func TestMigrateListsReleasesOnlyWithTheirIndex(t *testing.T) {
	root := copyFixture(t, "release-shipped-retired-v07")
	if err := os.Remove(filepath.Join(root, "releases", "INDEX.md")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if idx := string(mustRead(t, filepath.Join(root, "INDEX.md"))); strings.Contains(idx, "/releases/") {
		t.Errorf("the root lists no releases/ with no index:\n%s", idx)
	}
	if strings.Contains(out.String(), "broken cross-link") {
		t.Errorf("no link is broken:\n%s", out.String())
	}
}

// What 1.0 has no place for, and migrate cannot move for the bundle, is
// refused before anything is written, each named where it is now: a stray
// Markdown file at the root, a document named index.md or log.md, a debt
// beside a directory of Markdown, a feature group whose place in features/
// is taken, and a directory of Markdown that is a symbolic link, at the root
// or in a group. A directory of images beside a debt is outside FDF, and no
// reason to refuse.
func TestMigrateRefusesWhat10HasNoPlaceFor(t *testing.T) {
	root := buildV07Bundle(t)
	debt := func(title string) string {
		return "---\ntype: Debt\nstatus: open\ntitle: " + title + "\ndescription: d.\ntimestamp: 2026-09-23T00:00:00Z\n---\n\n# Gap\n\n" + title + ".\n"
	}
	write(t, root, "worklog.md", "# Work log\n")
	write(t, root, "venues/log.md", strings.Replace(string(mustRead(t, filepath.Join(root, "menus", "daily.md"))), "title: Daily menu", "title: Log", 1))
	write(t, root, "debts/rounding.md", debt("Rounding"))
	write(t, root, "debts/rounding/notes.md", "# Notes\n")
	write(t, root, "debts/slow.md", debt("Slow"))
	write(t, root, "debts/slow/chart.png", "PNG")
	write(t, root, "features/menus/menu.png", "PNG")
	shared := t.TempDir()
	write(t, shared, "x.md", "# X\n")
	if err := os.Symlink(shared, filepath.Join(root, "specials")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := os.Symlink(shared, filepath.Join(root, "venues", "sub")); err != nil {
		t.Fatal(err)
	}
	before := tree(t, root)
	var out bytes.Buffer
	if code := Run(Options{Root: root}, &out); code != 1 {
		t.Fatalf("migrate exit %d, want 1\n%s", code, out.String())
	}
	want := "cannot migrate — fix these first (bundle left unchanged):\n" +
		"  debts/rounding/: shares its name with the debt debts/rounding.md, and a debt owns no directory — rename one of them\n" +
		"  features/menus: already there, where the feature group menus/ moves — move it out of features/\n" +
		"  specials: a symbolic link to " + shared + ", a directory of Markdown that migrate does not read through — replace it with the directory it names\n" +
		"  venues/log.md: no document is named log.md: a disk that ignores case reads it as the LOG.md beside it — rename it\n" +
		"  venues/sub: a symbolic link to " + shared + ", a directory of Markdown that migrate does not read through — replace it with the directory it names\n" +
		"  worklog.md: the bundle root holds only INDEX.md, LOG.md, SPEC.md, README.md and the five Context documents — file it in a register, or move it out of the bundle\n"
	if out.String() != want {
		t.Errorf("migrate names what it cannot place:\n got: %q\nwant: %q", out.String(), want)
	}
	if after := tree(t, root); after != before {
		t.Errorf("a refused migration changes nothing")
	}
	// A features that is no directory leaves the register no place.
	root = buildV07Bundle(t)
	write(t, root, "features", "notes\n")
	out.Reset()
	if code := Run(Options{Root: root}, &out); code != 1 || out.String() != "cannot migrate — fix these first (bundle left unchanged):\n"+
		"  features: not a directory, where the features/ register goes — move it out of the bundle root\n" {
		t.Errorf("a features that is no directory is refused: exit %d\n%s", code, out.String())
	}
}

// Which root directories are registers depends on the pin: changes/ from
// 0.5, practices/ and debts/ from 0.6, bugs/ from 0.7. Under an older pin
// each is a feature group, and moves into features/.
func TestMigrateReadsTheRegistersOfItsPin(t *testing.T) {
	feature := func(title string) string {
		return "---\ntype: Feature\nstatus: draft\ntitle: " + title + "\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: " + title + "\n  As a user\n  I want it\n  So that it helps\n```\n\n```gherkin\nScenario: It works\n  Given it\n  When it runs\n  Then it works\n```\n"
	}
	for _, tc := range []struct {
		pin, group string
	}{{"0.4", "changes"}, {"0.5", "practices"}, {"0.5", "debts"}, {"0.6", "bugs"}} {
		root := t.TempDir()
		write(t, root, "INDEX.md", "---\nfdf_version: \""+tc.pin+"\"\n---\n\n# Bundle\n")
		write(t, root, tc.group+"/x.md", feature("X"))
		var out bytes.Buffer
		if code := Run(Options{Root: root}, &out); code != 0 {
			t.Errorf("%s under %s: exit %d\n%s", tc.group, tc.pin, code, out.String())
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "features", tc.group, "x.md")); err != nil {
			t.Errorf("%s/ is a feature group under %s, and moves into features/: %v", tc.group, tc.pin, err)
		}
		if _, err := os.Stat(filepath.Join(root, tc.group, "INDEX.md")); err != nil {
			t.Errorf("%s/ is a register in 1.0, with its index: %v", tc.group, err)
		}
	}
}

// gitIn runs git in dir and returns what it printed.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// gitProject makes a git repository whose docs/features holds a copy of the
// conformance fixture name, all committed, and returns the repository's root.
func gitProject(t *testing.T, name string) string {
	t.Helper()
	return gitProjectAt(t, name, t.TempDir())
}

// gitProjectAt is gitProject, with the repository at project.
func gitProjectAt(t *testing.T, name, project string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	copyFixtureTo(t, name, filepath.Join(project, "docs", "features"))
	write(t, project, "README.md", "# Project\n")
	gitIn(t, project, "init", "-q")
	gitIn(t, project, "add", "-A")
	gitIn(t, project, "commit", "-qm", "bundle")
	return project
}

// worktree is tree without the repository's own .git, which git status
// touches.
func worktree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			fmt.Fprintf(&b, "== %s\n%s", filepath.ToSlash(rel), mustRead(t, p))
		}
		return nil
	})
	return b.String()
}

// A bundle at docs/features moves to docs/fdf beside it, in the same run, and
// migrate marks the files it wrote with git add -N, so that git diff -M shows
// each move as a rename. The plan says where the bundle goes, and the next
// steps how to review it, and that FDF_ROOT_DIR still names the old path.
func TestMigrateMovesDocsFeaturesToDocsFdf(t *testing.T) {
	project := gitProject(t, "valid-bugs-v07")
	root := filepath.Join(project, "docs", "features")
	dest := filepath.Join(project, "docs", "fdf")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: project, EnvRoot: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(root); err == nil {
		t.Errorf("docs/features moved to docs/fdf, and is gone")
	}
	for _, p := range []string{"INDEX.md", "features/INDEX.md", "features/venues/opening-hours.md", "bugs/INDEX.md"} {
		if _, err := os.Stat(filepath.Join(dest, p)); err != nil {
			t.Errorf("docs/fdf/%s: %v", p, err)
		}
	}
	diff := gitIn(t, project, "diff", "-M", "--name-status")
	for _, want := range []string{
		"R100\tdocs/features/venues/opening-hours.plan.md\tdocs/fdf/features/venues/opening-hours.plan.md\n",
		"R100\tdocs/features/bugs/ui/unclear-error.md\tdocs/fdf/bugs/ui/unclear-error.md\n",
		"A\tdocs/fdf/features/INDEX.md\n",
	} {
		if !strings.Contains(diff, want) {
			t.Errorf("git diff -M shows %q:\n%s", want, diff)
		}
	}
	for _, want := range []string{
		"  bundle     docs/features/ → docs/fdf/\n",
		"and moved it to " + dest + "; logged in LOG.md.\n",
		"then review it with `git diff -M` — migrate marked the files it wrote with `git add -N`,\n",
		"FDF_ROOT_DIR still names " + root + ": point it at " + dest + ".\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output should say %q:\n%s", want, out.String())
		}
	}
	if log := string(mustRead(t, filepath.Join(dest, "LOG.md"))); !strings.Contains(log, "moved the bundle from `docs/features/` to `docs/fdf/`") {
		t.Errorf("LOG.md records the move:\n%s", log)
	}
}

// --to chooses where the bundle goes, and naming where it is keeps it there.
func TestMigrateMovesTheBundleWhereToSays(t *testing.T) {
	project := gitProject(t, "valid-bugs-v07")
	root := filepath.Join(project, "docs", "features")
	dest := filepath.Join(project, "wiki", "fdf")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: project, To: dest}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(dest, "features", "venues", "opening-hours.md")); err != nil {
		t.Errorf("the bundle is at --to: %v", err)
	}
	for _, gone := range []string{root, filepath.Join(project, "docs", "fdf")} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s: the bundle went to --to, and nowhere else", gone)
		}
	}

	project = gitProject(t, "valid-bugs-v07")
	root = filepath.Join(project, "docs", "features")
	out.Reset()
	if code := Run(Options{Root: root, Project: project, To: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "features", "venues", "opening-hours.md")); err != nil || strings.Contains(out.String(), "  bundle ") {
		t.Errorf("--to naming where the bundle is keeps it there: %v\n%s", err, out.String())
	}
}

// migrate refuses a destination it cannot take, before it writes anything:
// one that is there and not empty, one inside the bundle, one outside the
// project, one inside git's own directory, one a file stands in the way of,
// and any move of a bundle that is its own repository. An empty directory
// at the destination is no obstacle.
func TestMigrateRefusesADestinationItCannotTake(t *testing.T) {
	project := gitProject(t, "valid-bugs-v07")
	root := filepath.Join(project, "docs", "features")
	dest := filepath.Join(project, "docs", "fdf")
	write(t, project, "docs/fdf/notes.md", "# Notes\n")
	outside := filepath.Join(t.TempDir(), "fdf")
	own := gitProject(t, "valid-bugs-v07")
	ownRoot := filepath.Join(own, "docs", "features")
	gitIn(t, ownRoot, "init", "-q")
	gitIn(t, ownRoot, "add", "-A")
	gitIn(t, ownRoot, "commit", "-qm", "bundle")
	for _, tc := range []struct {
		o    Options
		says string
	}{
		{Options{Root: root, Project: project}, dest + " is there and not empty — pass --to <dir> to choose another destination, or --to " + root + " to keep the bundle where it is"},
		{Options{Root: root, Project: project, To: filepath.Join(root, "fdf")}, filepath.Join(root, "fdf") + " is inside the bundle, which cannot move into itself"},
		{Options{Root: root, Project: project, To: outside}, outside + " is outside the project at " + project + ", where git could neither show the move nor undo it — pass --to a directory inside the project"},
		{Options{Root: root, Project: project, To: filepath.Join(project, ".git", "fdf")}, filepath.Join(project, ".git", "fdf") + " is inside git's own directory — pass --to <dir> to choose another destination"},
		{Options{Root: root, Project: project, To: filepath.Join(project, "README.md", "fdf")}, filepath.Join(project, "README.md") + " is a file, where " + filepath.Join(project, "README.md", "fdf") + " needs a directory — pass --to <dir> to choose another destination"},
		{Options{Root: ownRoot, Project: own, To: filepath.Join(own, "wiki", "fdf")}, "the bundle at " + ownRoot + " is the root of its own git repository, which migrate does not move — move it yourself, or pass --to " + ownRoot + " to keep it where it is"},
	} {
		before := worktree(t, project)
		var out bytes.Buffer
		if code := Run(tc.o, &out); code != 1 || out.String() != "cannot migrate: "+tc.says+"; the bundle was left as it is.\n" {
			t.Errorf("exit %d\n got: %q\nwant: %q", code, out.String(), "cannot migrate: "+tc.says+"; the bundle was left as it is.\n")
		}
		if worktree(t, project) != before {
			t.Errorf("a refused migration changes nothing")
		}
	}
	if err := os.Remove(filepath.Join(dest, "notes.md")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: project}, &out); code != 0 {
		t.Errorf("an empty directory at the destination gives way: exit %d\n%s", code, out.String())
	}
}

// Git is the migration's undo, so migrate starts only from a clean tree: a
// change not committed in the bundle, staged or not, or a file git does not
// track there, one it ignores included, is refused. A file outside the
// bundle that git does not track is no obstacle, and nor is a hidden file
// git ignores, such as a Finder .DS_Store.
func TestMigrateStartsFromACleanTree(t *testing.T) {
	feature := "docs/features/venues/opening-hours.md"
	for _, tc := range []struct {
		name  string
		dirty func(project string)
		shows string
	}{
		{"a change", func(project string) { write(t, project, feature, "changed\n") }, " M " + feature},
		{"a staged change", func(project string) { write(t, project, feature, "changed\n"); gitIn(t, project, "add", feature) }, "M  " + feature},
		{"a file git does not track", func(project string) { write(t, project, "docs/features/venues/draft.md", "# Draft\n") }, "?? docs/features/venues/draft.md"},
		{"a file git ignores", func(project string) {
			write(t, project, ".gitignore", "*.local.md\n")
			write(t, project, "docs/features/venues/notes.local.md", "# Notes\n")
		}, "!! docs/features/venues/notes.local.md"},
	} {
		project := gitProject(t, "valid-bugs-v07")
		tc.dirty(project)
		before := worktree(t, project)
		var out bytes.Buffer
		want := "cannot migrate: files migrate would change have changes not committed, or are files git does not track — commit them, stash them or move them out of the bundle first, so that git can show the migration and undo it (bundle left unchanged):\n  " + tc.shows + "\n"
		if code := Run(Options{Root: filepath.Join(project, "docs", "features"), Project: project}, &out); code != 1 || out.String() != want {
			t.Errorf("%s: exit %d\n got: %q\nwant: %q", tc.name, code, out.String(), want)
		}
		if worktree(t, project) != before {
			t.Errorf("%s: a refused migration changes nothing", tc.name)
		}
	}
	project := gitProject(t, "valid-bugs-v07")
	write(t, project, "notes.txt", "mine\n")
	write(t, project, ".gitignore", ".DS_Store\n")
	write(t, project, "docs/features/venues/.DS_Store", "Finder\n")
	var out bytes.Buffer
	if code := Run(Options{Root: filepath.Join(project, "docs", "features"), Project: project}, &out); code != 0 {
		t.Errorf("a file outside the bundle, and a hidden one git ignores, are no obstacle: exit %d\n%s", code, out.String())
	}
}

// A bundle that is a git submodule moves with git mv, which updates
// .gitmodules, and the next steps say to commit inside it first.
func TestMigrateMovesASubmoduleWithGitMv(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "bundle-repo")
	copyFixtureTo(t, "valid-bugs-v07", repo)
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-qm", "bundle")
	super := filepath.Join(tmp, "super")
	write(t, super, "README.md", "# Super\n")
	gitIn(t, super, "init", "-q")
	gitIn(t, super, "add", "-A")
	gitIn(t, super, "commit", "-qm", "code")
	gitIn(t, super, "-c", "protocol.file.allow=always", "submodule", "add", "-q", repo, "docs/features")
	gitIn(t, super, "commit", "-qm", "mount the bundle")

	root := filepath.Join(super, "docs", "features")
	dest := filepath.Join(super, "docs", "fdf")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: super}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if gm := string(mustRead(t, filepath.Join(super, ".gitmodules"))); !strings.Contains(gm, "[submodule \"docs/features\"]\n\tpath = docs/fdf\n") {
		t.Errorf(".gitmodules follows the move, and the submodule keeps its name:\n%s", gm)
	}
	if diff := gitIn(t, dest, "diff", "-M", "--name-status"); !strings.Contains(diff, "R100\tvenues/opening-hours.plan.md\tfeatures/venues/opening-hours.plan.md\n") {
		t.Errorf("inside the submodule, git diff -M shows the moves:\n%s", diff)
	}
	if !strings.Contains(out.String(), "then review it inside the submodule, with `git -C "+dest+" diff -M`, and commit it there first;\n") {
		t.Errorf("the next steps start inside the submodule:\n%s", out.String())
	}
}

// A bundle that is its own git repository — a documentation repository
// checked out on its own — migrates where it is: migrate checks that
// repository's tree, and marks its new files there, so that git diff -M
// shows each move.
func TestMigrateABundleThatIsItsOwnRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := filepath.Join(t.TempDir(), "handbook")
	copyFixtureTo(t, "valid-bugs-v07", root)
	gitIn(t, root, "init", "-q")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-qm", "bundle")
	write(t, root, "draft.txt", "not committed\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: root}, &out); code != 1 || !strings.HasSuffix(out.String(), "\n  ?? draft.txt\n") {
		t.Fatalf("a file git does not track in the repository is refused: exit %d\n%s", code, out.String())
	}
	if err := os.Remove(filepath.Join(root, "draft.txt")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := Run(Options{Root: root, Project: root}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if diff := gitIn(t, root, "diff", "-M", "--name-status"); !strings.Contains(diff, "R100\tvenues/opening-hours.plan.md\tfeatures/venues/opening-hours.plan.md\n") {
		t.Errorf("git diff -M shows each move:\n%s", diff)
	}
}

// Should a migration stop partway, migrate prints the git commands that put
// everything back, and they do: the clean tree it started from is in git.
// Each path in them is quoted for the shell, as this project's is.
func TestMigrateSaysHowToUndoAStoppedMigration(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes into any directory")
	}
	project := gitProjectAt(t, "valid-bugs-v07", filepath.Join(t.TempDir(), "my project"))
	before := worktree(t, project)
	// practices/ is a directory migrate cannot write its index into.
	blocked := filepath.Join(project, "docs", "features", "practices")
	if err := os.Mkdir(blocked, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(blocked, 0o755)
	var out bytes.Buffer
	if code := Run(Options{Root: filepath.Join(project, "docs", "features"), Project: project}, &out); code != 1 {
		t.Fatalf("migrate exit %d, want 1\n%s", code, out.String())
	}
	_, undo, ok := strings.Cut(out.String(), "the migration stopped partway. To put everything back as it was:\n")
	if !ok {
		t.Fatalf("migrate says how to undo it:\n%s", out.String())
	}
	os.Chmod(blocked, 0o755)
	for _, line := range strings.Split(strings.TrimSpace(undo), "\n") {
		cmd := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if b, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", cmd, err, b)
		}
	}
	if after := worktree(t, project); after != before {
		t.Errorf("the commands put everything back:\n%s", undo)
	}
	if status := gitIn(t, project, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Errorf("git status is clean again:\n%s", status)
	}
}

// Git is the undo in the repository that tracks the bundle: the nearest one
// above it, not the topmost. A repository checked out inside another tracks
// its own files, so migrate checks, and marks, the inner one's.
func TestMigrateChecksTheRepositoryThatTracksTheBundle(t *testing.T) {
	outer := gitProject(t, "valid-bugs-v07")
	inner := filepath.Join(outer, "vendor", "handbook")
	gitProjectAt(t, "valid-bugs-v07", inner)
	root := filepath.Join(inner, "docs", "features")
	write(t, inner, "docs/features/venues/opening-hours.md", "changed\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: outer}, &out); code != 1 || !strings.HasSuffix(out.String(), "\n   M docs/features/venues/opening-hours.md\n") {
		t.Fatalf("a change in the inner repository is refused: exit %d\n%s", code, out.String())
	}
	gitIn(t, inner, "checkout", "--", ".")
	out.Reset()
	if code := Run(Options{Root: root, Project: outer}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if diff := gitIn(t, inner, "diff", "-M", "--name-status"); !strings.Contains(diff, "R100\tdocs/features/venues/opening-hours.plan.md\tdocs/fdf/features/venues/opening-hours.plan.md\n") {
		t.Errorf("git diff -M in the inner repository shows each move:\n%s", diff)
	}
}

// On a disk that ignores case, a root spelled in another case finds the
// bundle, and migrate reads it as the disk spells it, which is how git
// knows it: a change not committed there is seen, and refused.
func TestMigrateReadsTheRootAsTheDiskSpellsIt(t *testing.T) {
	project := gitProject(t, "valid-bugs-v07")
	root := filepath.Join(project, "Docs", "Features")
	if _, err := os.Stat(root); err != nil {
		t.Skip("the disk reads case")
	}
	write(t, project, "docs/features/venues/opening-hours.md", "changed\n")
	var out bytes.Buffer
	if code := Run(Options{Root: root, Project: project}, &out); code != 1 || !strings.HasSuffix(out.String(), "\n   M docs/features/venues/opening-hours.md\n") {
		t.Errorf("the change is seen through a root spelled in another case: exit %d\n%s", code, out.String())
	}
}

// v0.1's renames change only case, index.md to INDEX.md. On a disk that
// ignores case git would not see one in a bundle that stays where it is, and
// would commit the old name: migrate has git forget it, so what is
// committed is spelled as the disk spells it.
func TestMigrateRecordsARenameOnlyCaseTells(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	project := t.TempDir()
	buildV01Bundle(t, filepath.Join(project, "handbook"))
	gitIn(t, project, "init", "-q")
	gitIn(t, project, "add", "-A")
	gitIn(t, project, "commit", "-qm", "bundle")
	var out bytes.Buffer
	Run(Options{Root: filepath.Join(project, "handbook"), Project: project}, &out)
	if !strings.Contains(out.String(), "\ndone: ") {
		t.Fatalf("migrate did not finish:\n%s", out.String())
	}
	gitIn(t, project, "add", "-A")
	files := gitIn(t, project, "ls-files", "handbook")
	for _, want := range []string{"handbook/INDEX.md\n", "handbook/LOG.md\n"} {
		if !strings.Contains(files, want) {
			t.Errorf("git records %q:\n%s", want, files)
		}
	}
	for _, old := range []string{"handbook/index.md\n", "handbook/log.md\n"} {
		if strings.Contains(files, old) {
			t.Errorf("git no longer records %q:\n%s", old, files)
		}
	}
}

// A symbolic link in the bundle that names its target by a relative path is
// named again once it moves, so that it names the same place: one that
// leaves the bundle gains the ../ its move needs, and one beside what it
// names, moving with it, stays as it is. The plan lists each it names
// again.
func TestMigrateRepointsARelativeSymlink(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "docs", "features")
	copyFixtureTo(t, "valid-bugs-v07", root)
	write(t, dir, "assets/logo.png", "PNG")
	write(t, root, "venues/diagram.png", "PNG")
	if err := os.Symlink("../../../assets", filepath.Join(root, "venues", "assets")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := os.Symlink("diagram.png", filepath.Join(root, "venues", "current.png")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := Run(Options{Root: root, To: filepath.Join(dir, "handbook", "fdf")}, &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	group := filepath.Join(dir, "handbook", "fdf", "features", "venues")
	for name, want := range map[string]string{"assets": "../../../../assets", "current.png": "diagram.png"} {
		if got, err := os.Readlink(filepath.Join(group, name)); err != nil || got != want {
			t.Errorf("%s links to %q, want %q: %v", name, got, want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(group, "assets", "logo.png")); err != nil {
		t.Errorf("the repointed link names the same place: %v", err)
	}
	if !strings.Contains(out.String(), "\n  relink  features/venues/assets → ../../../../assets\n") {
		t.Errorf("the plan lists the link it repoints:\n%s", out.String())
	}
}

// A dry run changes nothing, git's own files included: git status, which
// would refresh the index, takes no lock.
func TestMigrateDryRunLeavesGitAsItIs(t *testing.T) {
	project := gitProject(t, "valid-bugs-v07")
	index := filepath.Join(project, ".git", "index")
	before, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(mustRead(t, index))
	var out bytes.Buffer
	if code := Run(Options{Root: filepath.Join(project, "docs", "features"), Project: project, DryRun: true}, &out); code != 0 {
		t.Fatalf("migrate --dry-run exit %d\n%s", code, out.String())
	}
	after, err := os.Stat(index)
	if err != nil || !after.ModTime().Equal(before.ModTime()) || string(mustRead(t, index)) != raw {
		t.Errorf("a dry run leaves git's index as it was: %v", err)
	}
}
