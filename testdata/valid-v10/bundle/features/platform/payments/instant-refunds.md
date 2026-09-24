---
type: Feature
status: done
title: Instant refunds
description: Refund a settled payment on the spot.
depends-on: features/onboarding
version: "1.2.0"
surface: none
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Instant refunds
  As a support agent
  I want to refund a settled payment immediately
  So that the customer is not left waiting
```

# Scenarios

```gherkin
Scenario: Full refund of a settled payment
  Given a settled payment
  When the agent refunds it in full
  Then the customer is refunded
```

```gherkin
Scenario: Refund rejected outside the refund window
  Given a settled payment made 100 days ago
  When the agent attempts to refund it
  Then the refund is rejected
```
