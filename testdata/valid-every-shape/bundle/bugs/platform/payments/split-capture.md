---
type: Bug
status: open
title: A full refund misses the second capture
description: A payment settled in two captures is refunded for the first one only.
affects: features/platform/payments/instant-refunds
timestamp: 2026-09-24
---

# Symptom

A payment settled as two captures is refunded for the first one only.

# Expected

The customer is refunded the whole payment.

# Violates

## features/platform/payments/instant-refunds

- Full refund of a settled payment
