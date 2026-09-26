---
type: Fix
title: Refund rounding
description: Refunds were a cent short.
status: done
affects: features/payments/instant-refunds
timestamp: 2026-09-15T00:00:00Z
---

# Symptom

Refunds were one cent short on odd totals.

# Root cause

Integer division truncated instead of rounding.

# Scenario changes

## features/payments/instant-refunds
- Refund completes within 30 seconds — go test ./payments -run TestRefundRounding
