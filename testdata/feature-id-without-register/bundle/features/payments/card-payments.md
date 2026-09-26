---
type: Feature
status: adopted
title: Card payments
description: Take and settle card payments.
resource: internal/payments
surface: none
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Card payments
  As a customer
  I want to pay by card
  So that I need no cash
```

# Scenarios

```gherkin
Scenario: Card payment is settled
  Given an authorized card payment
  When the provider settles it
  Then the payment is marked settled
```
