---
type: Bug
status: wontfix
title: A defect
description: Something is wrong.
timestamp: 2026-09-23T00:00:00Z
---

# Symptom

Setting 09:00-17:00 shows the Venue closing at 16:00. Reproduce:
`go test ./... -run TestHours` fails with `closes 16:00, want 17:00`.

# Expected

The Venue closes at 17:00.
