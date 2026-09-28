---
type: Change
status: done
title: Shorten the refund window
description: Refunds past 90 days are rejected.
affects: features/platform/payments/instant-refunds
version: "1.2.0"
timestamp: 2026-09-24
---

# Problem

The window outlived the chargeback window.

# Scenario changes

## features/platform/payments/instant-refunds

- modify: Refund rejected outside the refund window
