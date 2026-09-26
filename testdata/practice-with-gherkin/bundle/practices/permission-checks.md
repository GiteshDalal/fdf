---
type: Practice
status: active
title: Permission checks
description: Where authorization decisions are made.
timestamp: 2026-09-16T00:00:00Z
---

# Rules

- Every handler resolves permission through authz.Can.

# How

Middleware resolves the principal once per request.

```gherkin
Scenario: nope
  Given a thing
```
