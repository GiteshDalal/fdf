---
type: Spec
title: Design
timestamp: 2026-09-23T00:00:00Z
---

# Approach

Each store keeps its own hours, and a store owner edits them. The hours live
in the data store beside the Venue. See [the old notes](/venues/store-notes.md)
and `store_hours` in the code.

The screen reads "Store hours" today.

<!-- older draft: every shop had one timetable -->

```sql
SELECT * FROM store;
```
