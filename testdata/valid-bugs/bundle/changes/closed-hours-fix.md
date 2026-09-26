---
type: Fix
status: done
title: Work
description: Post-delivery work.
affects: features/venues/opening-hours
resolves: bugs/closed-hours-shown-open
timestamp: 2026-09-23T00:00:00Z
---

# Symptom

The closing time was an hour early.

# Root cause

An off-by-one in the hour arithmetic.

# Regression cases

## features/venues/opening-hours

- Venue owner sets opening hours — `go test ./... -run TestHours`
