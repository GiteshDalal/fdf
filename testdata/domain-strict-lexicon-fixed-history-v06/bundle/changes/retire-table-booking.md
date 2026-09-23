---
type: Change
title: Retire table booking
description: The capability is gone.
status: done
affects: venues/table-booking
retires: venues/table-booking
timestamp: 2026-09-17T00:00:00Z
---

# Problem

Venues stopped taking bookings through us.

# Rationale

The booking partner closed its API and no Venue asked for a replacement.
Nothing replaces this capability.

# Scenario changes

## venues/table-booking
- modify: Customer books a table
