package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
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
	write(t, root, "fdf-spec.md", "---\ntype: Reference\ntitle: FDF v0.1\ndescription: vendored.\ntimestamp: 2026-07-05T00:00:00Z\n---\n\nOld spec body.\n")
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
	write(t, root, "wdise/example/TEST.md", "---\ntype: Test\ntitle: Example acceptance\ndescription: How proven.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Test Cases\n\n- Scenario: It works — verified.\n")
	write(t, root, "wdise/example/01-do-thing.md", "---\ntype: Task\ntitle: Do the thing\ndescription: One unit of work.\nstatus: done\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Objective\n\nDo it.\n")
	write(t, root, "wdise/example/LOG.md", "# Example feature log\n\n## 2026-07-06\n* Completed.\n")
}

func TestMigrateChainsToCurrentVersion(t *testing.T) {
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
	if !strings.Contains(string(idx), `fdf_version: "0.4"`) {
		t.Fatalf("pin not upgraded to current version:\n%s", idx)
	}
	// v0.4: migration scaffolds the spec copy and the four Context stubs.
	for _, p := range []string{"SPEC.md", "STACK.md", "ARCHITECTURE.md", "SURFACES.md", "INFRA.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("migrate did not scaffold %s", p)
		}
	}
	// The vendored spec must match the target version, not a stale one.
	spec, _ := os.ReadFile(filepath.Join(root, "SPEC.md"))
	if !strings.Contains(string(spec), "FDF v0.4") && !strings.Contains(string(spec), "— v0.4") && !strings.Contains(string(spec), "v0.4") {
		t.Fatalf("vendored SPEC.md not the v0.4 spec:\n%.200s", spec)
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
	if !strings.Contains(string(tst), "Scenario: It works") {
		t.Fatalf("TEST stub missing scenario after lift:\n%s", tst)
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
	if !strings.Contains(string(spec), "v0.4") {
		t.Fatalf("refreshed spec is not v0.4:\n%.200s", spec)
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
	if !strings.Contains(string(idx), `fdf_version: "0.4"`) {
		t.Fatalf("pin not upgraded to 0.4:\n%s", idx)
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

	// Closing message lists four context docs and warns about F9.
	msg := out.String()
	if !strings.Contains(msg, "STACK.md") || !strings.Contains(msg, "ARCHITECTURE.md") ||
		!strings.Contains(msg, "SURFACES.md") || !strings.Contains(msg, "INFRA.md") {
		t.Fatalf("closing message must list four context docs:\n%s", msg)
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

func TestMigrateAlready04IsNoop(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	// Minimal already-0.4 draft-only bundle (no features → no F9 hard fail).
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.4\"\n---\n\n# Bundle\n\n* [Log](/LOG.md) - log.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* Already on 0.4.\n")
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
		t.Fatalf("already-0.4 must not run layout transform:\n%s", msg)
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.4"`) {
		t.Fatalf("pin changed unexpectedly:\n%s", idx)
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
	if !strings.Contains(msg, "already exists") && !strings.Contains(msg, "cannot move") {
		t.Fatalf("error should mention destination conflict (already exists / cannot move):\n%s", msg)
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
