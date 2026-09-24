---
type: Feature
status: draft
title: Instant refunds
description: Refund a settled payment on the spot.
depends-on: payments/card-payments
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Instant refunds
  As a support agent
  I want to refund a settled payment
  So that the customer is not left waiting
```

# Scenarios

```gherkin
Scenario: Full refund of a settled payment
  Given a settled payment
  When the agent refunds it in full
  Then the customer is refunded
```
