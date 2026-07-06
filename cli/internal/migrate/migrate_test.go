package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	write(t, root, "index.md", "---\nfdf_version: \"0.1\"\n---\n\n# Bundle\n\n* [FDF format](/fdf-spec.md) - vendored spec.\n* [Wdise](/wdise/index.md) - group.\n* [log](/log.md) - root-absolute trail link.\n* [u](https://example.com/a/log.md) - external URL, must stay untouched.\n* [w](https://example.com/other-fdf-spec.md-notes.md) - external URL, must stay untouched.\n")
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

func TestMigrateV01ToV02(t *testing.T) {
	root := filepath.Join(t.TempDir(), "features")
	buildV01Bundle(t, root)
	var out bytes.Buffer
	if code := Run(root, "", &out); code != 0 {
		t.Fatalf("migrate exit %d\n%s", code, out.String())
	}
	for _, p := range []string{"INDEX.md", "LOG.md", "wdise/INDEX.md", "wdise/example/SPEC.md", "wdise/example/PLAN.md", "wdise/example/TEST.md"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Fatalf("missing %s after migrate", p)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "fdf-spec.md")); err == nil {
		t.Fatal("vendored fdf-spec.md must be deleted")
	}
	idx, _ := os.ReadFile(filepath.Join(root, "INDEX.md"))
	if !strings.Contains(string(idx), `fdf_version: "0.2"`) {
		t.Fatalf("pin not upgraded:\n%s", idx)
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
	feat, _ := os.ReadFile(filepath.Join(root, "wdise", "example.md"))
	if !strings.Contains(string(feat), "example/SPEC.md") || !strings.Contains(string(feat), "example/PLAN.md") {
		t.Fatalf("trail links not rewritten:\n%s", feat)
	}
	tst, _ := os.ReadFile(filepath.Join(root, "wdise", "example", "TEST.md"))
	if !strings.Contains(string(tst), "Scenario: It works") {
		t.Fatalf("TEST.md stub missing scenario:\n%s", tst)
	}
}
