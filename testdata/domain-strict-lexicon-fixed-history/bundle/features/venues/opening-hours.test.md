---
type: Test
title: Opening hours acceptance
description: How proven.
timestamp: 2026-09-20T00:00:00Z
---

# Test Cases

## Venue owner sets opening hours

go test ./venues -run TestOpeningHours

## Venue owner sets holiday hours

go test ./venues -run TestHolidayHours
