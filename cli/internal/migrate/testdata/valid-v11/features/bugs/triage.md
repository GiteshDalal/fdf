---
type: Feature
status: draft
title: Triage
description: Triage reported defects.
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Triage
  As a support agent
  I want to triage reported defects
  So that the worst are repaired first
```

# Scenarios

```gherkin
Scenario: A reported defect is triaged
  Given a reported defect
  When the agent triages it
  Then it has a severity
```
