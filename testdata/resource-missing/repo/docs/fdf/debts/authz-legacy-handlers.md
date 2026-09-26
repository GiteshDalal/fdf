---
type: Debt
status: open
title: Authz legacy handlers
description: Handlers that predate the practice.
resource: [internal/http/admin.go, no/such/handler.go]
timestamp: 2026-09-25T00:00:00Z
---

# Gap

Twelve handlers read user.Role directly instead of calling authz.Can.
