package migrate

import (
	"bytes"
	"fmt"
	"io"
	"os"
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
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for _, p := range []string{
		"INDEX.md", "LOG.md", "wdise/INDEX.md",
		"wdise/example.spec.md", "wdise/example.plan.md", "wdise/example.test.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing %s after migrate", p)
		}
	}
	// Nested trail must be gone.
	for _, p := range []string{"wdise/example/SPEC.md", "wdise/example/PLAN.md", "wdise/example/TEST.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Fatalf("nested trail %s must not remain after migrate", p)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "fdf-spec.md")); err == nil {
		t.Fatal("vendored fdf-spec.md must be deleted")
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+target+`"`) {
		t.Fatalf("pin not upgraded to the target, %s:\n%s", target, idx)
	}
	// v0.4: migration scaffolds the spec copy and the four Context stubs.
	for _, p := range []string{"SPEC.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md"} {
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
	if !strings.Contains(string(idx), "* [FDF format](/SPEC.md) - vendored spec.") {
		t.Fatalf("a link to the vendored fdf-spec.md names the vendored SPEC.md:\n%s", idx)
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
	feat, _ := os.ReadFile(filepath.Join(root, "wdise", "example.md"))
	if !strings.Contains(string(feat), "example.spec.md") || !strings.Contains(string(feat), "example.plan.md") {
		t.Fatalf("trail links not rewritten to stem form:\n%s", feat)
	}
	if strings.Contains(string(feat), "example/SPEC.md") || strings.Contains(string(feat), "example/spec.md") {
		t.Fatalf("old nested trail links must not remain:\n%s", feat)
	}
	tst, _ := os.ReadFile(filepath.Join(root, "wdise", "example.test.md"))
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
	if code := Run(root, "", &out); code != 0 {
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

func TestMigrateV03ToV04StemLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}

	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+target+`"`) {
		t.Fatalf("pin not upgraded to %s:\n%s", target, idx)
	}

	// Stem trail present; nested trail gone.
	for _, p := range []string{"wdise/example.spec.md", "wdise/example.plan.md", "wdise/example.test.md", "wdise/example.log.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing stem trail %s after migrate", p)
		}
	}
	for _, p := range []string{"wdise/example/SPEC.md", "wdise/example/PLAN.md", "wdise/example/TEST.md", "wdise/example/LOG.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			t.Fatalf("nested %s must not remain", p)
		}
	}

	// Tasks stay in the feature directory.
	if _, err := os.Stat(filepath.Join(root, "wdise", "example", "01-do-thing.md")); err != nil {
		t.Fatalf("task must remain under feature dir: %v", err)
	}

	// SURFACES stub scaffolded.
	if _, err := os.Stat(filepath.Join(root, "SURFACES.md")); err != nil {
		t.Fatalf("SURFACES.md must be scaffolded: %v", err)
	}

	// Feature trail links rewritten to stem form.
	feat, _ := os.ReadFile(filepath.Join(root, "wdise", "example.md"))
	for _, want := range []string{"example.spec.md", "example.plan.md", "example.test.md"} {
		if !strings.Contains(string(feat), want) {
			t.Fatalf("feature missing stem link %s:\n%s", want, feat)
		}
	}
	if strings.Contains(string(feat), "example/SPEC.md") {
		t.Fatalf("old nested link remains:\n%s", feat)
	}

	// Plan task links rewritten for new plan location (sibling → slug/task).
	plan, _ := os.ReadFile(filepath.Join(root, "wdise", "example.plan.md"))
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

// The pre-0.4 steps repair links with the one link engine, as fdf mv does: a
// link that leaves the bundle from a lifted trail file gains the ../ its
// move needs, and a reference definition is repaired like an inline link.
// migrate's own rewriters left both behind.
func TestMigrateRepairsLinksWithTheEngine(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs", "features")
	buildV03Bundle(t, root)
	write(t, root, "wdise/example/SPEC.md", "---\ntype: Spec\ntitle: Example spec\ndescription: Design.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Design\n\nThe handler is [refund.go](../../../../src/refund.go), beside [the map][okf].\n\n[okf]: ../../../okf/index.md\n")
	feature := filepath.Join(root, "wdise", "example.md")
	write(t, root, "wdise/example.md", string(mustRead(t, feature))+"\nSee [the plan][plan].\n\n[plan]: example/PLAN.md\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	spec := string(mustRead(t, filepath.Join(root, "wdise", "example.spec.md")))
	for _, want := range []string{"[refund.go](../../../src/refund.go)", "[okf]: ../../okf/index.md"} {
		if !strings.Contains(spec, want) {
			t.Errorf("the lifted spec should hold %q:\n%s", want, spec)
		}
	}
	if got := string(mustRead(t, feature)); !strings.Contains(got, "[plan]: example.plan.md") {
		t.Errorf("a reference definition to a lifted file is repaired:\n%s", got)
	}
}

func TestMigrateAlready04IsNoop(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	// Minimal already-current draft-only bundle (no features → no F9 hard fail).
	write(t, root, "INDEX.md", "---\nfdf_version: \""+target+"\"\n---\n\n# Bundle\n\n* [Log](/LOG.md) - log.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* Already current.\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	msg := out.String()
	if !strings.Contains(msg, "already") {
		t.Fatalf("expected already-at-current no-op message:\n%s", msg)
	}
	// Must not re-scaffold as if migrating (no "moved" trail messages).
	if strings.Contains(msg, "moved ") {
		t.Fatalf("already-current must not run layout transform:\n%s", msg)
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "`+target+`"`) {
		t.Fatalf("pin changed unexpectedly:\n%s", idx)
	}
}

// migrate upgrades a 0.x bundle to 0.7, and a bundle pinned to 1.0 is not
// one: its steps would pin a conformant 1.0 bundle back to 0.7, and refuse
// another with messages about v0.4. It is refused before anything but its
// pin is read, or anything is written, whatever it holds, and however its
// pin is quoted; so is a pin that is almost 1.0 but no version.
func TestMigrateLeavesA10BundleAsItIs(t *testing.T) {
	const at10 = "cannot migrate: the bundle pins fdf_version 1.0, and this binary upgrades a 0.x bundle to 0.7 — the bundle was left as it is."
	notAVersion := func(pin string) string {
		return "cannot migrate: the bundle pins fdf_version " + pin + ", which is not a MAJOR.MINOR version such as " + scaffold.CurrentVersion() + " — correct the pin in INDEX.md; the bundle was left as it is."
	}
	type refusal struct{ root, says string }
	var refusals []refusal
	// A pin that is almost 1.0 is no 0.x version either: migrate used to
	// take it for one, and pin the bundle back to 0.7.
	for _, tc := range []struct{ pin, says string }{
		{`"1.0"`, at10}, {`'1.0'`, at10}, {`1.0`, at10},
		{`"1.0.0"`, notAVersion("1.0.0")}, {`"v1.0"`, notAVersion("v1.0")},
	} {
		flat := t.TempDir()
		write(t, flat, "INDEX.md", "---\nfdf_version: "+tc.pin+"\n---\n\n# Bundle\n\n* [Features](/features/INDEX.md) - features.\n")
		write(t, flat, "LOG.md", "# Bundle Update Log\n\n## 2026-09-25\n* **Initialization**: created.\n")
		write(t, flat, "features/INDEX.md", "# Features\n\n* [Onboarding](/features/onboarding.md) - feature.\n")
		write(t, flat, "features/onboarding.md", "---\ntype: Feature\nstatus: draft\ntitle: Onboarding\ndescription: d.\ntimestamp: 2026-09-25T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Onboarding\n  As a user\n  I want to sign up\n  So that I can start\n```\n\n```gherkin\nScenario: It works\n  Given a\n  When b\n  Then c\n```\n")
		refusals = append(refusals, refusal{flat, tc.says})
	}
	refusals = append(refusals, refusal{copyFixture(t, "valid-v10"), at10})
	for _, r := range refusals {
		before := tree(t, r.root)
		var out bytes.Buffer
		if code := Run(r.root, "", &out); code != 1 || !strings.Contains(out.String(), r.says) {
			t.Errorf("a 1.0 bundle is refused: exit %d, want 1 saying %q:\n%s", code, r.says, out.String())
		}
		if after := tree(t, r.root); after != before {
			t.Errorf("a refused migration changes nothing:\nbefore:\n%s\nafter:\n%s", before, after)
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
	if code := Run(root, "", &out); code != 0 {
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
	if fi, err := os.Lstat(filepath.Join(root, "wdise", "shared.md")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symlink stays in its group, as a symlink: %v", err)
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
	if code := Run(link, "", &out); code != 1 || !strings.Contains(out.String(), "  "+link+": a symbolic link to "+bundle+" — migrate the directory it names, with --root\n") {
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
	if code := Run(root, "", &out); code != 1 || !strings.Contains(out.String(), "  LOG.md: a symbolic link to ../shared/LOG.md, which migrate would write through — replace it with the file it names\n") {
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
	p, problems, err := newPlan(root, "0.3")
	if err != nil || len(problems) > 0 {
		t.Fatalf("newPlan: %v %v", err, problems)
	}
	p.texts["NOTES.md"] = "migrate's\n"
	write(t, root, "NOTES.md", "mine\n")
	if err := p.apply(io.Discard); err == nil || !os.IsExist(err) {
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
	src := filepath.Join("..", "..", "..", "testdata", name, "bundle")
	filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(src, p)
			write(t, dst, rel, string(mustRead(t, p)))
		}
		return nil
	})
	return dst
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
		if code := Run(root, "", &out); code != 1 || out.String() != want {
			t.Errorf("fdf migrate --root %s: exit %d\n got: %q\nwant: %q", root, code, out.String(), want)
		}
		if after := tree(t, tc.bundle); after != before {
			t.Errorf("a refused migration changes nothing:\nbefore:\n%s\nafter:\n%s", before, after)
		}
	}
}

// tree lists every file under root with its content, in path order.
func tree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
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
		if code := Run(r, "", &out); code != 1 || !strings.Contains(out.String(), "no bundle at "+r+" (no INDEX.md)") {
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
	code := Run(root, "", &out)
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
	if code := Run(root, "", &out); code == 0 {
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
	if code := Run(root, "", &out); code == 0 {
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
	if code := Run(root, "", &out); code == 0 {
		t.Fatalf("expected pre-flight abort for stray feature-dir file:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "wdise/example/INDEX.md") {
		t.Fatalf("abort should name the stray file:\n%s", out.String())
	}
	requireUntouched(t, root, `fdf_version: "0.3"`, "wdise/example/INDEX.md", "wdise/example/SPEC.md")
}

func TestMigrateIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV03Bundle(t, root)
	var out1 bytes.Buffer
	if code := Run(root, "", &out1); code != 0 {
		t.Fatalf("first migrate exit %d\n%s", code, out1.String())
	}
	var out2 bytes.Buffer
	if code := Run(root, "", &out2); code != 0 {
		t.Fatalf("second migrate must stay exit 0 (idempotent), got %d\n%s", code, out2.String())
	}
}

func TestMigrateReadsUnquotedPin(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	write(t, root, "INDEX.md", "---\nfdf_version: "+target+"\n---\n\n# Bundle\n\n* [log](/LOG.md) - history.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* Init.\n")

	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
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
	if code := Run(root, "", &out); code != 0 {
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
// summary states the version transition and how much moved, so "it did
// nothing" is distinguishable from "it did a lot".
func TestMigrateSummaryReportsTransitionAndCount(t *testing.T) {
	root := t.TempDir()
	buildV03Bundle(t, root)
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("expected exit 0, got %d\n%s", code, out.String())
	}
	msg := out.String()
	if !strings.Contains(msg, "fdf_version 0.3 -> "+target) {
		t.Errorf("summary should state the version transition:\n%s", msg)
	}
	if !strings.Contains(msg, "trail file(s) lifted to stem-qualified siblings") {
		t.Errorf("summary should count lifted trail files:\n%s", msg)
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

func TestMigrateV06ToV07(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.7"`) {
		t.Fatalf("pin not upgraded:\n%s", idx)
	}
	if _, err := os.Stat(filepath.Join(root, "bugs", "INDEX.md")); err != nil {
		t.Fatalf("bugs/INDEX.md not scaffolded: %v", err)
	}
	spec, _ := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if !strings.Contains(string(spec), "Feature Document Format (FDF) — v0.7") {
		t.Fatal("SPEC.md not re-vendored to v0.7")
	}
	for _, want := range []string{
		"fdf_version 0.6 -> 0.7",
		"0 trail file(s) lifted",
		"the domain language now reaches every document and name — 1 banned word(s) in 1 file(s).",
		"of the 1 debt(s) on the register",
		"`fdf mv debts/<id> bugs/<id>`",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q\n%s", want, out.String())
		}
	}
	// Nothing moved: the feature and its spec are where they were.
	if _, err := os.Stat(filepath.Join(root, "venues", "hours.spec.md")); err != nil {
		t.Fatal("a v0.6 → v0.7 migration must not move documents")
	}
}

func TestMigrateRefusesBugsFeatureGroup(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "bugs/INDEX.md", "# Bugs\n\n* [Tracker](/bugs/tracker.md) - a feature group named bugs.\n")
	write(t, root, "bugs/tracker.md", "---\ntype: Feature\nstatus: draft\ntitle: Tracker\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Tracker\n  As a user\n  I want it\n  So that it helps\n```\n\n# Scenarios\n\n```gherkin\nScenario: It works\n  Given it\n  When it runs\n  Then it works\n```\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 1 {
		t.Fatalf("migrate exit %d, want 1\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "bugs/ is a feature group (bugs/tracker.md)") ||
		!strings.Contains(out.String(), "rename the group first (its directory, its listing in INDEX.md and the links to it)") ||
		strings.Contains(out.String(), "fdf mv") {
		t.Fatalf("refusal does not name the conflict and its fix:\n%s", out.String())
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.6"`) {
		t.Fatal("a refused migration must leave the bundle untouched")
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
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "debts/stub-gateway.md: missing recommended `description`") {
		t.Fatalf("the test needs a validation line naming the debt:\n%s", out.String())
	}
	if strings.Contains(out.String(), "fdf-init") {
		t.Errorf("every Context document is filled; nothing asks for fdf-init:\n%s", out.String())
	}

	// From v0.5, DOMAIN.md arrives as a stub, and the bundle has a feature.
	root = t.TempDir()
	buildV06Bundle(t, root)
	os.Remove(filepath.Join(root, "DOMAIN.md"))
	write(t, root, "INDEX.md", strings.Replace(string(mustRead(t, filepath.Join(root, "INDEX.md"))), `"0.6"`, `"0.5"`, 1))
	out.Reset()
	if code := Run(root, "", &out); code != 0 {
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

// The refusal names its way out, and the way out works: once the v0.6
// feature group is renamed — bugs/ is not a register under that pin — the
// migration goes through.
func TestMigrateAfterMovingTheBugsFeatureGroup(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.6\"\n---\n\n# Bundle\n\n* [Venues](/venues/INDEX.md) - group.\n* [Bugs](/bugs/INDEX.md) - the tracker.\n")
	write(t, root, "bugs/INDEX.md", "# Bugs\n\n* [Tracker](/bugs/tracker.md) - a feature group named bugs.\n")
	write(t, root, "bugs/tracker.md", "---\ntype: Feature\nstatus: draft\ntitle: Tracker\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Tracker\n  As a user\n  I want it\n  So that it helps\n```\n\n# Scenarios\n\n```gherkin\nScenario: It works\n  Given it\n  When it runs\n  Then it works\n```\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 1 || !strings.Contains(out.String(), "rename the group first") {
		t.Fatalf("migrate should refuse and name the rename: exit %d\n%s", code, out.String())
	}
	if err := os.Rename(filepath.Join(root, "bugs"), filepath.Join(root, "issues")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"INDEX.md", "issues/INDEX.md"} {
		write(t, root, rel, strings.ReplaceAll(string(mustRead(t, filepath.Join(root, rel))), "/bugs/", "/issues/"))
	}
	out.Reset()
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate after the move: exit %d\n%s", code, out.String())
	}
	if idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md")); !strings.Contains(string(idx), `fdf_version: "0.7"`) || !strings.Contains(string(idx), "(/issues/INDEX.md)") {
		t.Fatalf("migrated, with the root index following the moved group:\n%s", idx)
	}
	if raw, err := os.ReadFile(filepath.Join(root, "bugs", "INDEX.md")); err != nil || !strings.Contains(string(raw), "Known defects") {
		t.Fatalf("bugs/ is now the bug register: %v\n%s", err, raw)
	}
	if _, err := os.Stat(filepath.Join(root, "issues", "tracker.md")); err != nil {
		t.Fatalf("the feature moved with its group: %v", err)
	}
}

// v0.7 matches a test case exactly: a `## <scenario name>` heading under
// `# Test Cases`. A 0.6 bundle that lists its cases as bullets migrates, but
// its validation then fails F8, and migrate says what to rewrite by hand —
// and which features still owe a surface decision.
// Older tools wrote a status tag after each index listing and nothing kept it
// current; migration removes it, leaves other bold text and code alone, and
// logs the migration in the bundle-root log.
func TestMigrateV06ToV07DropsIndexStatusTags(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "venues/INDEX.md", "# Venues\n\n* [Hours](/venues/hours.md) - opening hours. (**draft**)\n* [Menu](/venues/hours.md) - the menu. (**important**)\n\n```\n* [Sample](/venues/hours.md) - a sample. (**done**)\n```\n")
	write(t, root, "changes/INDEX.md", "# Changes\n\n* [x](/changes/INDEX.md) - changes. (**specified**)\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	venues, _ := os.ReadFile(filepath.Join(root, "venues", "INDEX.md"))
	for _, want := range []string{
		"* [Hours](/venues/hours.md) - opening hours.\n",
		"* [Menu](/venues/hours.md) - the menu. (**important**)\n",
		"* [Sample](/venues/hours.md) - a sample. (**done**)\n",
	} {
		if !strings.Contains(string(venues), want) {
			t.Errorf("venues/INDEX.md should contain %q:\n%s", want, venues)
		}
	}
	changes, _ := os.ReadFile(filepath.Join(root, "changes", "INDEX.md"))
	if strings.Contains(string(changes), "(**specified**)") {
		t.Errorf("changes/INDEX.md keeps its tag:\n%s", changes)
	}
	log, _ := os.ReadFile(filepath.Join(root, "LOG.md"))
	if !strings.Contains(string(log), "* **Migrated**: fdf_version 0.6 → 0.7 with `fdf migrate`. Removed the status tag from 2 index listing(s)") {
		t.Errorf("LOG.md should record the migration:\n%s", log)
	}
	if !strings.Contains(out.String(), "2 status tag(s) removed from index listings") {
		t.Errorf("output should count the tags:\n%s", out.String())
	}
}

func TestMigrateV06ToV07ReportsBulletCasesAndSurfaces(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "venues/hours.md", "---\ntype: Feature\nstatus: planned\ntitle: Hours\ndescription: Opening hours.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Hours\n  As a Venue owner\n  I want hours\n  So that people know\n```\n\n# Scenarios\n\n```gherkin\nScenario: Owner sets hours\n  Given a Venue\n  When the owner sets hours\n  Then they show\n```\n")
	write(t, root, "venues/hours.plan.md", "---\ntype: Plan\ntitle: Plan\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Tasks\n")
	write(t, root, "venues/hours.test.md", "---\ntype: Test\ntitle: Tests\ndescription: d.\ntimestamp: 2026-09-16 09:30\n---\n\n# Test Cases\n\n- Scenario: Owner sets hours — `go test ./... -run TestHours`\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 1 {
		t.Fatalf("a bundle whose cases are bullets fails F8 once migrated; want exit 1, got %d\n%s", code, out.String())
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.7"`) {
		t.Fatalf("the migration itself still happens:\n%s", idx)
	}
	for _, want := range []string{
		`venues/hours.test.md: scenario "Owner sets hours" has no test case`,
		"1 scenario(s) have none (F8). Rewrite those test documents' cases as headings, by hand",
		"1 feature(s) have no slug.surface.md",
		"1 timestamp(s) are neither a date nor an RFC 3339 time with Z or an offset (F1)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q\n%s", want, out.String())
		}
	}
}
