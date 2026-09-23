---
type: Feature
title: Venue locator
description: An example feature.
status: done
timestamp: 2026-09-16T00:00:00Z
---

# Feature

```gherkin
Feature: Venue locator
  As a merchant
  I want this capability
  So that customers are served
```

# Scenarios

```gherkin
Scenario: Customer finds the nearest Venue
  Given three Venues at different distances
  When a customer searches from where they are
  Then the nearest Venue is listed first
```
