---
type: Fix
status: done
title: Signup confirmation
description: The signup confirmation names the account it created.
affects: features/onboarding
timestamp: 2026-09-24
---

# Symptom

The confirmation named no account.

# Root cause

A missing field in the template.

# Regression cases

## features/onboarding

- Venue owner signs up — `go test ./... -run TestSignup`
