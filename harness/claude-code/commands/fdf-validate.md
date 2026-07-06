---
description: Validate the FDF bundle and fix any violations it reports
---

Run `fdf validate`. If it exits non-zero, read each FAIL line, fix the bundle
(never weaken content to silence a rule), and re-run until exit 0. Report
warnings but do not treat them as failures.
