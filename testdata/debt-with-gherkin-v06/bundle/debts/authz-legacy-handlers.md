---
type: Debt
status: open
title: Authz legacy handlers
description: Handlers that predate the practice.
timestamp: 2026-09-16T00:00:00Z
---

# Gap

Twelve handlers read user.Role directly instead of calling authz.Can.

```gherkin
Scenario: nope
  Given a thing
```
