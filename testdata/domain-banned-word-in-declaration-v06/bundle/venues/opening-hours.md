---
type: Feature
title: Opening hours
description: An example feature.
status: done
timestamp: 2026-09-16T00:00:00Z
---

# Feature

```gherkin
Feature: Opening hours
  As a merchant
  I want this capability
  So that customers are served
```

# Scenarios

```gherkin
Scenario: Store owner sets opening hours
  Given a Venue with no opening hours
  When its owner sets them
  Then customers see the new hours
```

```gherkin
Scenario: Store owner opens late
  Given a Venue with regular opening hours
  When its owner opens late for one evening
  Then customers see the late hours that evening
```
