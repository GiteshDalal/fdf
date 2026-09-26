---
type: Feature
title: Instant refunds
description: An example feature.
status: done
depends-on: features/payments/refund-limits
timestamp: 2026-09-15T00:00:00Z
---

# Feature

```gherkin
Feature: Instant refunds
  As a shopper
  I want my money back quickly
  So that I trust the store
```

# Scenarios

```gherkin
Scenario: Refund completes within 30 seconds
  Given a paid order
  When a refund is issued
  Then the money is back within 30 seconds
```
