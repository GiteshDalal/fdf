---
type: Feature
status: adopted
title: Card payments
description: A Venue capability.
resource: [src/cards]
timestamp: 2026-09-23T00:00:00Z
---

# Feature

```gherkin
Feature: Card payments
  As a Venue owner
  I want it
  So that it helps
```

# Scenarios

```gherkin
Scenario: A settled payment is marked settled
  Given a Venue
  When the owner acts
  Then it works
```

```gherkin
Scenario: A declined card is refused
  Given a Venue
  When the owner acts
  Then it works
```
