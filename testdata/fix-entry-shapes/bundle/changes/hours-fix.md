---
type: Fix
status: draft
title: Work
description: Post-delivery work.
affects: features/venues/opening-hours
timestamp: 2026-09-23T00:00:00Z
---

# Symptom

The closing time was an hour early.

# Regression cases

- Venue owner sets opening hours — `go test ./... -run TestHours`

## features/venues/opening-hours

1. Venue owner sets opening hours — `go test ./... -run TestHours`
