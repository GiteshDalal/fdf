---
description: Open a post-delivery Fix for a bug in a delivered FDF feature and start the fdf-change skill
---

A delivered FDF feature has a bug: $ARGUMENTS

Use the fdf-change skill. Confirm first that the feature document already
describes the correct behavior and only the code drifted — if you would need
to add or change a scenario, it is a Change (`/fdf-change`), not a Fix.
Scaffold with `fdf fix --affects <group>/<slug>[,…] [<group>/]<slug>`, and
remember the lasting artifact is the regression case in the affected feature's
`slug.test.md`.
