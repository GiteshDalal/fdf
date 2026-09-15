---
type: Feature
title: Refund limits
description: Second feature in the cycle.
status: draft
depends-on: payments/instant-refunds
timestamp: 2026-09-15T00:00:00Z
---

# Feature

```gherkin
Feature: Refund limits
  As an operator
  I want caps
  So that losses are bounded
```

# Scenarios

```gherkin
Scenario: A refund over the cap is rejected
  Given a cap
  When a larger refund is tried
  Then it is rejected
```
