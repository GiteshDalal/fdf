---
type: Feature
title: Table booking
description: An example feature.
status: retired
timestamp: 2026-09-16T00:00:00Z
---

# Feature

```gherkin
Feature: Table booking
  As a merchant
  I want this capability
  So that customers are served
```

# Scenarios

```gherkin
Scenario: Customer books a table
  Given a Venue with a free table
  When a customer books it
  Then the table is held for them
```
