---
type: Fix
title: Opening hours timezone
description: Hours show in the wrong timezone.
status: draft
affects: venues/opening-hours
timestamp: 2026-09-21T00:00:00Z
---

# Symptom

Hours show in UTC rather than the Venue's local time.

# Root cause

The formatter ignores the Venue's timezone.

# Regression cases

## venues/opening-hours
- Store owner sets opening hours — go test ./venues -run TestOpeningHoursTimezone
