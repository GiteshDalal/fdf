---
type: Practice
status: superseded
superseded-by: practices/authz-v2
title: Permission checks
description: Old.
timestamp: 2026-09-16T00:00:00Z
---

# Rules

- Every handler resolves permission through authz.Can.

# How

Middleware resolves the principal once per request.
