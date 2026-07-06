// Package scaffold implements fdf init and fdf new.
package scaffold

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const specURL = "https://github.com/GiteshDalal/fdf/blob/main/SPEC.md"
const currentVersion = "0.2"

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)
var pinRe = regexp.MustCompile(`fdf_version:\s*"([^"]+)"`)

func Init(root string, out io.Writer) int {
	idx := filepath.Join(root, "INDEX.md")
	if raw, err := os.ReadFile(idx); err == nil {
		if m := pinRe.FindSubmatch(raw); m != nil && string(m[1]) == currentVersion {
			fmt.Fprintf(out, "bundle at %s is already initialized and up to date (fdf_version %s)\n", root, currentVersion)
			return 0
		} else if m != nil {
			fmt.Fprintf(out, "bundle at %s pins fdf_version %q; run `fdf migrate` to upgrade to %s\n", root, m[1], currentVersion)
			return 1
		}
		fmt.Fprintf(out, "bundle at %s has an INDEX.md without an fdf_version pin; run `fdf migrate`\n", root)
		return 1
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	today := time.Now().UTC().Format("2006-01-02")
	index := fmt.Sprintf("---\nfdf_version: %q\n---\n\n# Feature Bundle\n\nThis bundle conforms to [FDF v%s](%s): features in Markdown + Gherkin, each\nwith its SPEC/PLAN/TEST/task trail in a paired directory.\n\n# Overview\n\n* [FDF spec](%s) - the format this bundle pins.\n\n# Conventions\n\n* Feature IDs are file paths minus `.md`.\n* Validate with `fdf validate`; see [`LOG.md`](/LOG.md) for change history.\n",
		currentVersion, currentVersion, specURL, specURL)
	log := fmt.Sprintf("# Bundle Update Log\n\n## %s\n* **Initialization**: scaffolded by `fdf init` (FDF v%s).\n", today, currentVersion)
	if err := os.WriteFile(idx, []byte(index), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(root, "LOG.md"), []byte(log), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "initialized FDF bundle at %s\n", root)
	return 0
}

func New(root, id string, out io.Writer) int {
	if !idRe.MatchString(id) {
		fmt.Fprintf(out, "error: feature id must be <group>/<slug>, lowercase [a-z0-9-]; got %q\n", id)
		return 1
	}
	group, slug, _ := strings.Cut(id, "/")
	featurePath := filepath.Join(root, group, slug+".md")
	if _, err := os.Stat(featurePath); err == nil {
		fmt.Fprintf(out, "error: %s already exists\n", featurePath)
		return 1
	}
	if err := os.MkdirAll(filepath.Join(root, group), 0o755); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	title := strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	feature := fmt.Sprintf(`---
type: Feature
title: %s
description: TODO — one sentence.
status: draft
timestamp: %s
---

# Feature

`+"```gherkin"+`
Feature: %s
  As a <role>
  I want <capability>
  So that <value>
`+"```"+`

# Scenarios

`+"```gherkin"+`
Scenario: Replace me
  Given a precondition
  When something happens
  Then an observable outcome
`+"```"+`
`, title, now, title)
	if err := os.WriteFile(featurePath, []byte(feature), 0o644); err != nil {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	// Create or append the group index.
	gidx := filepath.Join(root, group, "INDEX.md")
	entry := fmt.Sprintf("* [%s](/%s/%s.md) - TODO. (**draft**)\n", title, group, slug)
	if raw, err := os.ReadFile(gidx); err == nil {
		if err := os.WriteFile(gidx, append(raw, []byte(entry)...), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		heading := strings.ToUpper(group[:1]) + group[1:]
		if err := os.WriteFile(gidx, []byte(fmt.Sprintf("# %s features\n\n%s", heading, entry)), 0o644); err != nil {
			fmt.Fprintln(out, "error:", err)
			return 1
		}
	} else {
		fmt.Fprintln(out, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "created %s (status: draft)\n", featurePath)
	return 0
}
