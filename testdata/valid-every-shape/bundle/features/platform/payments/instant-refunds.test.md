---
type: Test
title: Instant refunds — test cases
description: How each scenario is proven.
timestamp: 2026-09-24
---

# Test Cases

## Full refund of a settled payment

`go test ./... -run TestFullRefund`

## Refund rejected outside the refund window

`go test ./... -run TestOutsideWindow`
