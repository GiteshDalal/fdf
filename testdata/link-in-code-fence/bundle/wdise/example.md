---
type: Feature
title: Example
description: A feature whose body shows example links inside code fences.
status: draft
timestamp: 2026-07-06T00:00:00Z
---

# Feature

```gherkin
Feature: Example
  As a user
  I want a thing
  So that I get value
```

Real links are checked: [missing](/really-missing.md) does not exist.
An inline code span is not a link either: `[inline](inline-only.md)`.

A fenced example is sample text, not a cross-link:

````markdown
# Tasks

1. [01-first.md](example/01-first.md)
2. [02-second.md](example/02-second.md)

See also [the group](/nowhere/INDEX.md).

```gherkin
Scenario: a nested fence does not close the outer one
  Given a ```` block
  Then [still-sample.md](still-sample.md) is ignored
```
````

# Scenarios

```gherkin
Scenario: It works
  Given a thing
  When it runs
  Then it works
```
