package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/bundle"
	"github.com/GiteshDalal/fdf/cli/internal/refactor"
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
	write(t, root, "wdise/example/TEST.md", "---\ntype: Test\ntitle: Example acceptance\ndescription: How proven.\ntimestamp: 2026-07-06T00:00:00Z\n---\n\n# Test Cases\n\n## It works\n\nverified.\n")
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
	if !strings.Contains(string(idx), `fdf_version: "`+currentVersion+`"`) {
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
	if !strings.Contains(string(spec), "v"+currentVersion) {
		t.Fatalf("vendored SPEC.md not the v%s spec:\n%.200s", currentVersion, spec)
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
	if !strings.Contains(string(spec), "v"+currentVersion) {
		t.Fatalf("refreshed spec is not v%s:\n%.200s", currentVersion, spec)
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
	if !strings.Contains(string(idx), `fdf_version: "`+currentVersion+`"`) {
		t.Fatalf("pin not upgraded to %s:\n%s", currentVersion, idx)
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

func TestMigrateAlready04IsNoop(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	// Minimal already-current draft-only bundle (no features → no F9 hard fail).
	write(t, root, "INDEX.md", "---\nfdf_version: \""+currentVersion+"\"\n---\n\n# Bundle\n\n* [Log](/LOG.md) - log.\n")
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
	if !strings.Contains(string(idx), `fdf_version: "`+currentVersion+`"`) {
		t.Fatalf("pin changed unexpectedly:\n%s", idx)
	}
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
	write(t, root, "INDEX.md", "---\nfdf_version: "+currentVersion+"\n---\n\n# Bundle\n\n* [log](/LOG.md) - history.\n")
	write(t, root, "LOG.md", "# Bundle Update Log\n\n## 2026-07-06\n* Init.\n")

	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "already pins fdf_version "+currentVersion) {
		t.Fatalf("unquoted pin must hit the already-current path:\n%s", out.String())
	}
}

// A migrate that finds the pin already current looks identical to a migrate
// that is simply too old to know about newer versions — which is what happens
// when a version shim (mise, asdf) holds an old fdf in a directory. The
// message must name the binary so the user can tell those apart.
func TestNoOpMigrateNamesTheBinaryVersion(t *testing.T) {
	root := t.TempDir()
	write(t, root, "INDEX.md", "---\nfdf_version: \""+currentVersion+"\"\n---\n\n# Bundle\n")
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
	if !strings.Contains(msg, "fdf_version 0.3 -> "+currentVersion) {
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
		"the domain language now reaches every document and name — 1 banned word(s) in 1 document(s)",
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
		!strings.Contains(out.String(), "`fdf mv bugs <new-group>`") {
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

// The refusal names its way out, and the way out works: `fdf mv bugs
// <new-group>` renames the v0.6 feature group — bugs/ is not a register
// under that pin — and the migration then goes through.
func TestMigrateAfterMovingTheBugsFeatureGroup(t *testing.T) {
	root := t.TempDir()
	buildV06Bundle(t, root)
	write(t, root, "INDEX.md", "---\nfdf_version: \"0.6\"\n---\n\n# Bundle\n\n* [Venues](/venues/INDEX.md) - group.\n* [Bugs](/bugs/INDEX.md) - the tracker.\n")
	write(t, root, "bugs/INDEX.md", "# Bugs\n\n* [Tracker](/bugs/tracker.md) - a feature group named bugs.\n")
	write(t, root, "bugs/tracker.md", "---\ntype: Feature\nstatus: draft\ntitle: Tracker\ndescription: d.\ntimestamp: 2026-09-16T00:00:00Z\n---\n\n# Feature\n\n```gherkin\nFeature: Tracker\n  As a user\n  I want it\n  So that it helps\n```\n\n# Scenarios\n\n```gherkin\nScenario: It works\n  Given it\n  When it runs\n  Then it works\n```\n")
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 1 || !strings.Contains(out.String(), "`fdf mv bugs <new-group>`") {
		t.Fatalf("migrate should refuse and name the move: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := refactor.Move(root, "", "bugs", "issues", false, &out); code != 0 {
		t.Fatalf("fdf mv bugs issues on the v0.6 bundle: exit %d\n%s", code, out.String())
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
