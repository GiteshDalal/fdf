---
type: Change
title: Retire instant refunds
description: The capability is gone.
status: done
affects: features/payments/instant-refunds
retires: features/payments/instant-refunds
timestamp: 2026-09-15T00:00:00Z
---

# Problem

The payment provider withdrew instant settlement.

# Rationale

The provider withdrew the instant-settlement API; refunds now follow the
standard batch path documented elsewhere. Nothing replaces this capability.

# Scenario changes

## features/payments/instant-refunds
- modify: Refund completes within 30 seconds
