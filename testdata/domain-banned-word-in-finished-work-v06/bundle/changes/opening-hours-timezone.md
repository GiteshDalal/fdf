---
type: Fix
title: Opening hours timezone
description: Hours showed in the wrong timezone.
status: done
affects: venues/opening-hours
timestamp: 2026-09-18T00:00:00Z
---

# Symptom

Hours showed in UTC rather than the Venue's local time.

# Root cause

The formatter ignored the Venue's timezone.

# Regression cases

## venues/opening-hours
- Store owner sets opening hours — go test ./venues -run TestOpeningHoursTimezone
