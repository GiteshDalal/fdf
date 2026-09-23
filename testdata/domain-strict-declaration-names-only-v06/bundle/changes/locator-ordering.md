---
type: Fix
title: Locator ordering
description: Equidistant Venues come back in the wrong order.
status: draft
affects: shopfront/store-locator
timestamp: 2026-09-21T00:00:00Z
---

# Symptom

With two equidistant Venues, the nearest is sometimes listed second.

# Root cause

The distance sort is not stable.

# Regression cases

## shopfront/store-locator
- Customer finds the nearest Venue — manual: tap "Find a store", allow location, and confirm the nearest Venue is listed first
