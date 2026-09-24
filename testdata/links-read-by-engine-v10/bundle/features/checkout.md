---
type: Feature
status: done
title: Checkout
description: Pay for an order.
version: "2.0.0"
surface: none
timestamp: 2026-09-24
---

# Feature

```gherkin
Feature: Checkout
  As a customer
  I want to pay for an order
  So that it ships
```

Design in [the spec](checkout.spec.md "The approved design"). The retired
flow is [documented][gone], and so are [the notes](<notes/a b.md>).

    A sample, indented: [sample](nowhere-indented.md)

A span, ``with `[x](nowhere-span.md)` inside``.

A sample, fenced:

```text
[sample](nowhere-fenced.md)
```

[gone]: /features/gone.md

# Scenarios

```gherkin
Scenario: Customer pays for an order
  Given an order
  When the customer pays
  Then the order is paid
```
