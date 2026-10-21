---
type: Feature
status: done
title: Onboarding
description: Sign up and get started.
surface: none
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Onboarding
  As a Venue owner
  I want to sign up
  So that I can list my Venue
```

# Scenarios

```gherkin
Scenario: Venue owner signs up
  Given a new Venue owner
  When they sign up
  Then their account exists
```
