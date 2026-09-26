---
type: Change
title: Refund window
description: Tighten the settlement window.
status: done
affects: features/payments/instant-refunds
timestamp: 2026-09-16T00:00:00Z
---

# Problem

Thirty seconds is slower than competitors.

# Scenario changes

## features/payments/instant-refunds
- remove: Refund completes within 30 seconds
- add: Refund completes within 10 seconds
