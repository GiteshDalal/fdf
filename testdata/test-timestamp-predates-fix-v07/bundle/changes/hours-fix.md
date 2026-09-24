---
type: Fix
status: done
title: Work
description: Post-delivery work.
affects: venues/opening-hours
timestamp: 2026-10-02T00:00:00Z
---

# Symptom

The closing time was an hour early.

# Root cause

An off-by-one in the hour arithmetic.

# Regression cases

## venues/opening-hours

- Venue owner sets opening hours — `go test ./... -run TestHours`
