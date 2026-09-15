---
description: Open a post-delivery Change on a delivered FDF feature and start the fdf-change skill
---

A delivered (`done`/`retired`) feature needs to behave differently: $ARGUMENTS

Use the fdf-change skill. Confirm first that this really alters documented
behavior — if the feature document is already right and only the code drifted,
it is a Fix (`/fdf-fix`), not a Change. Scaffold with
`fdf change --affects <group>/<slug>[,…] [<group>/]<slug>` and work through
the skill from there.
