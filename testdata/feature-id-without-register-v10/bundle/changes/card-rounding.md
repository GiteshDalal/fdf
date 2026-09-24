---
type: Fix
status: draft
title: Card rounding
description: Amounts are rounded once.
affects: payments/card-payments
timestamp: 2026-09-24
---

# Regression cases

## payments/card-payments

- Card payment is settled — `go test ./... -run TestSettle`
