# Architecture Refactor Spec

## Overview

Resource management has two layers:
- **Node-level limits** — total capacity of a worker node, used by the dispatcher for placement
- **Per-gameserver resources** — what an individual gameserver is allocated, enforced or informational

These are independent. Node limits constrain placement. Per-gameserver resources constrain the container.

There is NO per-gameserver cap/ceiling layer. The operator (newbie, power user, or business automation) sets resources directly.

---

## Per-Gameserver Resources

| Field | Docker enforced? | Dispatcher uses? | Default |
|---|---|---|---|
| `memory_limit_mb` | Always | Yes | Game's `recommended_memory_mb` |
| `cpu_limit` | When `cpu_enforced` is true | Yes (when > 0) | 0 (unlimited) |
| `storage_limit_mb` | Never (soft/informational) | Yes (when > 0) | 0 (unlimited) |
| `backup_limit` | N/A (retention) | No | null (global default) |

### Memory (`memory_limit_mb`)

- **Docker enforced:** Always. Container is OOM-killed if exceeded.
- **Default:** Game's `recommended_memory_mb` (applied by `applyGameDefaults` on create)
- **Dispatcher:** Yes — summed per node for placement scoring.
- **Status:** Working correctly today. No changes needed.

### CPU (`cpu_limit`, `cpu_enforced`)

- **Docker enforced:** Controlled by `cpu_enforced` boolean (new field, default: `false`)
  - `false` → dispatcher uses it for placement, Docker does not limit. Server can burst to all cores.
  - `true` → Docker enforces via cgroup. Server cannot exceed the limit.
- **Default:** 0 (unlimited — no Docker enforcement, not counted by dispatcher)
- **Dispatcher:** Yes — summed per node for placement scoring (only when > 0).

**Why soft by default:** Most game servers are single-threaded during gameplay but multi-threaded during startup, world saves, and chunk loading. Hard enforcement slows down these operations without improving steady-state performance.

**Status:** Currently always Docker-enforced when set. Need to add `cpu_enforced` field and only pass `NanoCPUs` to Docker when true.

### Storage (`storage_limit_mb`)

- **Docker enforced:** Never. Hitting a hard storage cap mid-save would corrupt the world.
- **Default:** 0 (unlimited — not counted by dispatcher)
- **Dispatcher:** Yes — summed per node for placement scoring (only when > 0).

**Status:** Currently named `max_storage_mb` — needs renaming to `storage_limit_mb`.

### Backups (`backup_limit`)

- **Enforcement:** Retention — oldest backups deleted when limit reached.
- **Default:** null (falls back to global `MaxBackups` setting, default 10)

**Status:** Currently named `max_backups` — needs renaming to `backup_limit`. Already working.

---

## Remove Per-Gameserver Cap Fields

Delete cap fields that were designed for a non-existent use case (end users adjusting resources within a business-set ceiling):

**Delete from schema:**
- `gameservers.max_memory_mb`
- `gameservers.max_cpu`

**Rename in schema:**
- `gameservers.max_storage_mb` → `gameservers.storage_limit_mb`
- `gameservers.max_backups` → `gameservers.backup_limit`

**Delete from code:**
- Cap enforcement block in `UpdateGameserver` (lines 295-320)
- Cap fields from model, API, UI
- Update `AllocatedStorageByNode` query to use `storage_limit_mb`

---

## Node-Level Limits

Node limits on `worker_nodes`: `max_memory_mb`, `max_cpu`, `max_storage_mb`.

**Purpose:** Dispatcher placement only. Operator decides how much hardware to allocate to gameservers.

**Behavior:**
- Create/migrate exceeding capacity → reject with clear error
- No node record = no limits (local worker for newbies/power users)
- Checked by `checkWorkerLimits()` during create, migrate, and auto-migration
- Node decommission is manual (cordon + migrate)

---

## Node Tags

**Schema:** `tags TEXT NOT NULL DEFAULT '[]'` on `worker_nodes`
**Gameserver field:** `node_tags TEXT NOT NULL DEFAULT '[]'` on `gameservers`

**Dispatcher behavior:**
1. Filter candidates to nodes with ALL required `node_tags`
2. Rank remaining by available resources

**Tag validation:** Free-form strings (k8s approach). Clear error on no match mentioning the unmatched tags.

**Auto-migration must respect tags** — premium servers never migrate to standard nodes.

---

## Require Resource Limits Settings

Per-resource-type settings (all boolean, all default false):
- `require_memory_limit` — `memory_limit_mb` must be > 0
- `require_cpu_limit` — `cpu_limit` must be > 0
- `require_storage_limit` — `storage_limit_mb` must be > 0

**Multi-node warning:** Dispatching a gameserver with 0/unlimited resources in multi-node logs a warning but does not block.

---

## Auto-Migration on Resource Update

**Flow:**
1. Snapshot current resource values before applying changes
2. Apply new values to DB
3. Check if current node can still fit (excluding this gameserver's old allocation)
4. If fits → done, server restarts with new limits
5. If doesn't fit → dispatcher placement (filter by `node_tags`, rank by resources)
6. If node found → `MigrateGameserver` async (server goes through stopping → stopped → installing → starting → running on new node)
7. If no node → **rollback**: restore old values, error state with actionable message

**Rollback:** Old values restored, error state: "Upgrade to {value} failed: no node with sufficient capacity. Resource values unchanged." Webhook fires `status_changed` → error. Migration backup (if attempted) persists under `migrations/` prefix.

**API response:** Includes `"migration_triggered": true` when auto-migration starts.

---

## Migration via Backup Store

**Problem:** Controller buffers entire volume tar in memory. 50GB server = 50GB RAM.

**Fix:** Use backup store (S3/local) as intermediary:
1. Tar + gzip → store
2. Store → restore on target
3. Cleanup on success, persist on failure (`migrations/` prefix)

---

## Token Rework

Replace the three-scope model (admin/scoped/worker) with a single token type. Every token has:
- **Permissions list** — subset of action constants
- **Gameserver ID filter** — `[]` = all gameservers, `["id1", "id2"]` = restricted
- **Optional expiry**

**Authorization is one code path:** does this token have permission X? If gameserver-specific, is this gameserver in the filter?

**Convenience presets** (UI buttons, not separate code paths):
- "Admin token" → all permissions, `[]` gameserver filter
- "Worker token" → `worker.connect` permission only
- "Custom token" → pick permissions, optionally restrict gameserver IDs

The `scope` column can remain as a UI hint (`"admin"`, `"worker"`, `"custom"`) but is NOT used for authorization. Only the permissions list matters.

### Permissions

Full permissions list defined in the Constants section below.

---

## Replace Audit with Events

The audit middleware is removed. Events replace it entirely.

### Why
- Events fire on all state changes (HTTP, async, scheduled). Audit only fires on HTTP.
- Events carry domain data. Audit only has HTTP path.
- Events go to webhooks. Audit sits in a DB table.
- One less system to maintain.

### What gets removed
- `internal/web/audit.go` — middleware
- `audit_log` table from schema
- Audit model, handlers, routes, UI

---

## Event System Redesign

### Event bus

The current `EventBroadcaster` becomes the central event bus. All events flow through it. Consumers:
- **Status subscriber** — derives gameserver status from events, writes to DB (replaces `setGameserverStatus` calls scattered across the codebase)
- **Webhooks** — external delivery via `WebhookWorker` (existing)
- **SSE** — real-time browser/client updates, filterable via `?types=` query param
- **Event store** — persists events to DB for history/review
- **Crash counter** — existing subscriber for crash detection (stays)

### Event-driven status

**Current problem:** `setGameserverStatus` is called from 25+ places across the codebase. Status changes and events are tightly coupled in that function. Adding a new lifecycle step means knowing to call `setGameserverStatus` in the right place.

**New model:** Lifecycle code publishes events describing what happened. A single status subscriber derives the gameserver status from those events.

```go
// Lifecycle code — just says what happened
s.bus.Publish(ImagePullingEvent{GameserverID: id})
// ... later ...
s.bus.Publish(ContainerStartedEvent{GameserverID: id})

// Status subscriber — single place that maps events to status
func (s *StatusSubscriber) handleEvent(event Event) {
    switch e := event.(type) {
    case ImagePullingEvent:       s.setStatus(e.GameserverID, "installing")
    case ContainerCreatingEvent:  s.setStatus(e.GameserverID, "starting")
    case ContainerStartedEvent:   s.setStatus(e.GameserverID, "started")
    case GameserverReadyEvent:    s.setStatus(e.GameserverID, "running")
    case ContainerStoppingEvent:  s.setStatus(e.GameserverID, "stopping")
    case ContainerStoppedEvent:   s.setStatus(e.GameserverID, "stopped")
    case ContainerExitedEvent:    // trigger crash handler
    case GameserverErrorEvent:    s.setStatus(e.GameserverID, "error")
    }
}
```

**Benefits:**
- Status derivation in one place instead of 25+ direct calls
- Lifecycle code is simpler — publish what happened, don't decide what it means
- Adding a new lifecycle step = new event + one case in the subscriber
- All consumers (webhooks, SSE, event store) automatically see new events
- Status field stays on the gameserver row for quick reads/filtering

**`setGameserverStatus` is removed from lifecycle code.** It becomes an internal method of the status subscriber only.

### Action vs Outcome events

**Action events** — someone/something requested an operation. Fires immediately from the service layer. Carries real actor.
- `gameserver.create`, `gameserver.update`, `gameserver.delete`
- `gameserver.start`, `gameserver.stop`, `gameserver.restart`
- `backup.create`, `backup.delete`, `backup.restore`
- `schedule.create`, `schedule.update`, `schedule.delete`

**Outcome events** — something happened in the system. Fires when it occurs. Actor is always `{type: "system"}`.

Lifecycle outcomes (drive status changes):
- `gameserver.image_pulling` — downloading game image
- `gameserver.image_pulled` — image ready
- `gameserver.container_creating` — setting up container
- `gameserver.container_started` — process launched
- `gameserver.ready` — ready check passed, accepting players
- `gameserver.container_stopping` — container being stopped
- `gameserver.container_stopped` — container exited cleanly
- `gameserver.container_exited` — unexpected exit, triggers crash handler
- `gameserver.error` — something failed, carries error reason

Operation outcomes:
- `backup.completed`, `backup.failed`
- `backup.restore.completed`, `backup.restore.failed`
- `worker.connected`, `worker.disconnected`
- `schedule.task.completed`, `schedule.task.failed`

### Status derivation mapping

Status reflects what the container is doing right now. No operation statuses (`updating`, `reinstalling`, `migrating`, `restoring`). If you want to know WHY it's installing, check the recent action event.

| Event | Gameserver status |
|---|---|
| `gameserver.image_pulling` | `installing` |
| `gameserver.container_creating` | `starting` |
| `gameserver.container_started` | `started` |
| `gameserver.ready` | `running` |
| `gameserver.container_stopping` | `stopping` |
| `gameserver.container_stopped` | `stopped` |
| `gameserver.container_exited` | triggers crash handler |
| `gameserver.error` | `error` |

**Statuses: `stopped`, `installing`, `starting`, `started`, `running`, `stopping`, `error`**

7 statuses, down from 11. Each maps directly to what the container is doing. Operations (update, reinstall, migrate, restore) all go through the same lifecycle — they pull an image, create a container, start it. The status reflects that lifecycle, not the operation that triggered it.

The action events tell the full story:
- Server shows `installing` → check events → `gameserver.update` was the action → it's updating
- Server shows `installing` → check events → `gameserver.start` was the action → it's first start
- Server shows `stopped` → check events → `backup.restore` was the action + `container.stopped` → restore stopped the server

The UI that initiated the operation already knows the context. Other observers (monitoring, second user) can correlate action + status from the event stream.

### `status_changed` as a derived event

The status subscriber, after updating the gameserver's status, publishes a `status_changed` event with `old_status` and `new_status`. This is a **derived event** — it doesn't come from lifecycle code, it comes from the status subscriber. Webhook consumers who only care about status transitions subscribe to `status_changed` and get the same experience as today.

### Where events publish

**Action events** publish at the start of service methods, where `ctx` is available:
```go
func (s *GameserverService) Start(ctx context.Context, id string) {
    s.bus.Publish(StartEvent{
        Actor: actorFromCtx(ctx),
        GameserverID: id,
    })
    // ... pull image, create container, etc.
    // Outcome events fire as things happen:
    s.bus.Publish(ImagePullingEvent{GameserverID: id})
}
```

**Outcome events** publish where the thing happens — lifecycle code, Docker event watchers, ready watcher, backup callbacks. No actor needed, no ctx needed.

### Naming convention

Actions use the verb: `create`, `delete`, `start`, `stop`, `restore`
Outcomes describe what happened: `pulling`, `pulled`, `started`, `stopped`, `completed`, `failed`, `connected`, `ready`, `error`

`backup.create` is the action (someone requested it). `backup.completed` is the outcome (it finished). No `initialize` — the verb itself is clear.

### SSE filtering

SSE endpoint accepts `?types=` query param for server-side filtering:
- `?types=status_changed` — current browser UI behavior (default)
- `?types=status_changed,gameserver.ready` — more granular
- No param or `?types=*` — everything

### Event storage

Events stored in DB table with retention policy (configurable, default 30 days):
```sql
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    gameserver_id TEXT,
    actor JSON NOT NULL,
    data JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

REST endpoint: `GET /api/events?type=gameserver.*&gameserver_id=xxx&limit=50`

### Actor model

Every event carries an `actor` object. Always present, never null.

```json
{"type": "token", "token_id": "abc-123"}
{"type": "schedule", "schedule_id": "def-456"}
{"type": "system"}
{"type": "anonymous"}
```

| Type | Meaning |
|---|---|
| `token` | Authenticated request via API token. Includes `token_id`. |
| `schedule` | Scheduled task triggered it. Includes `schedule_id`. |
| `system` | Gamejanitor acted autonomously (crash, auto-restart, worker events, async outcomes). |
| `anonymous` | Someone did it but no identity attached (auth disabled). |

Replaces the current `actor_token_id` field on events.

**Actor on action events only** — determined at service method from `ctx`. Outcome events always `{type: "system"}`. No ctx threading through `setGameserverStatus`.

---

## Constants

Single source of truth, defined together for clarity but logically separate.

### Permissions

Used by the token system to gate API access. Stored in the token's permissions list.

```go
// permissions.go

// Gameserver lifecycle
var PermGameserverCreate  = "gameserver.create"
var PermGameserverUpdate  = "gameserver.update"
var PermGameserverDelete  = "gameserver.delete"
var PermGameserverStart   = "gameserver.start"
var PermGameserverStop    = "gameserver.stop"
var PermGameserverRestart = "gameserver.restart"

// Gameserver access
var PermGameserverFilesRead  = "gameserver.files.read"
var PermGameserverFilesWrite = "gameserver.files.write"
var PermGameserverLogs       = "gameserver.logs"
var PermGameserverCommand    = "gameserver.command"

// Gameserver config (per-field)
var PermGameserverEditName = "gameserver.edit.name"
var PermGameserverEditEnv  = "gameserver.edit.env"

// Backups
var PermBackupCreate   = "backup.create"
var PermBackupDelete   = "backup.delete"
var PermBackupRestore  = "backup.restore"
var PermBackupDownload = "backup.download"

// Schedules
var PermScheduleCreate = "schedule.create"
var PermScheduleUpdate = "schedule.update"
var PermScheduleDelete = "schedule.delete"

// Cluster management
var PermSettingsView   = "settings.view"
var PermSettingsEdit   = "settings.edit"
var PermTokensManage   = "tokens.manage"
var PermNodesManage    = "nodes.manage"
var PermWebhooksManage = "webhooks.manage"

// Worker
var PermWorkerConnect = "worker.connect"
```

Some permissions also fire as action events (e.g. `gameserver.start` is both a permission and an event type). Others are pure access gates with no corresponding event (`gameserver.files.read`, `settings.view`).

### Event types

Used by the event bus, webhooks, SSE, and event storage.

```go
// events.go

// Action events — user/schedule initiated, carry actor
var EventGameserverCreate  = "gameserver.create"
var EventGameserverUpdate  = "gameserver.update"
var EventGameserverDelete  = "gameserver.delete"
var EventGameserverStart   = "gameserver.start"
var EventGameserverStop    = "gameserver.stop"
var EventGameserverRestart = "gameserver.restart"
var EventBackupCreate      = "backup.create"
var EventBackupDelete      = "backup.delete"
var EventBackupRestore     = "backup.restore"
var EventScheduleCreate    = "schedule.create"
var EventScheduleUpdate    = "schedule.update"
var EventScheduleDelete    = "schedule.delete"

// Lifecycle outcome events — system, drive status changes
var EventImagePulling        = "gameserver.image_pulling"
var EventImagePulled         = "gameserver.image_pulled"
var EventContainerCreating   = "gameserver.container_creating"
var EventContainerStarted    = "gameserver.container_started"
var EventGameserverReady     = "gameserver.ready"
var EventContainerStopping   = "gameserver.container_stopping"
var EventContainerStopped    = "gameserver.container_stopped"
var EventContainerExited     = "gameserver.container_exited"
var EventGameserverError     = "gameserver.error"

// Operation outcome events — system
var EventBackupCompleted        = "backup.completed"
var EventBackupFailed           = "backup.failed"
var EventBackupRestoreCompleted = "backup.restore.completed"
var EventBackupRestoreFailed    = "backup.restore.failed"
var EventWorkerConnected        = "worker.connected"
var EventWorkerDisconnected     = "worker.disconnected"
var EventScheduleTaskCompleted  = "schedule.task.completed"
var EventScheduleTaskFailed     = "schedule.task.failed"

// Derived event — published by status subscriber
var EventStatusChanged = "status_changed"
```

---

## Input Validation

Resource values need bounds checking:
- `memory_limit_mb`: must be > 0 (always, or when `require_memory_limit` enabled)
- `cpu_limit`: must be >= 0 (must be > 0 when `require_cpu_limit` enabled)
- `storage_limit_mb`: must be >= 0 (must be > 0 when `require_storage_limit` enabled)
- `backup_limit`: must be >= 0 if set (0 = no backups allowed)

---

## Changes Required

### Remove cap fields
1. **Delete `max_memory_mb` from gameservers** — schema, model, API, UI
2. **Delete `max_cpu` from gameservers** — schema, model, API, UI
3. **Rename `max_storage_mb` → `storage_limit_mb`** — schema, model, API, UI, dispatcher queries
4. **Rename `max_backups` → `backup_limit`** — schema, model, API, UI, retention query
5. **Delete cap enforcement code** — remove cap check block from `UpdateGameserver`

### Per-gameserver resources
6. **Add `cpu_enforced` field** — boolean, default false
7. **Conditional CPU enforcement** — `docker.go` only sets `NanoCPUs` when `cpu_enforced` is true
8. **Input validation** — bounds checking on resource values

### Global settings
9. **Add `require_memory_limit` setting** — boolean, default false
10. **Add `require_cpu_limit` setting** — boolean, default false
11. **Add `require_storage_limit` setting** — boolean, default false

### Node tags
12. **Add `tags` field to `worker_nodes`**
13. **Add `node_tags` field to `gameservers`**
14. **Dispatcher tag filtering** — filter before ranking
15. **API/UI for managing node tags**

### Auto-migration
16. **Snapshot + rollback** — save old values, restore on failure
17. **Capacity check on update** — excluding gameserver's old allocation
18. **Auto-migrate on overcommit** — dispatcher placement + async migration
19. **Respect tags during auto-migration**
20. **Error state with clear message + webhook**
21. **Migration response flag** — `migration_triggered: true` in PATCH response

### Migration resilience
22. **Migration via backup store** — replace in-memory buffer
23. **Store migrations separately** — `migrations/` prefix
24. **No auto-cleanup on failure**

### Token rework
25. **Single token type with permissions list** — replace three-scope model
26. **Permissions = action constants** — token's permission list is subset of constants
27. **Gameserver ID filter** — `[]` = all, specific IDs = restricted
28. **Convenience presets** — admin (all perms), worker (`worker.connect`), custom (pick perms)
29. **Define full permissions list** — OPEN, needs detailed specification

### Event-driven status
30. **Create status subscriber** — single subscriber maps events to gameserver status, replaces 25+ `setGameserverStatus` calls
31. **Remove `setGameserverStatus` from lifecycle code** — becomes internal to status subscriber only
32. **`status_changed` as derived event** — published by status subscriber after updating status
33. **Simplify statuses** — 7 states only: `stopped`, `installing`, `starting`, `started`, `running`, `stopping`, `error`. Remove operation statuses (`updating`, `reinstalling`, `migrating`, `restoring`).

### Replace audit with events
34. **Remove audit middleware** — code, table, model, handlers, routes, UI

### Action events
35. **Add action events to service methods** — `gameserver.start`, `gameserver.stop`, `backup.create`, etc. fire at start with actor
36. **Make backup creation async** — return immediately, fire `backup.completed`/`backup.failed` on completion

### Outcome events
37. **Add lifecycle outcome events** — `gameserver.image_pulling`, `gameserver.image_pulled`, `gameserver.container_creating`, `gameserver.container_started`, `gameserver.ready`, `gameserver.container_stopping`, `gameserver.container_stopped`, `gameserver.container_exited`, `gameserver.error`
38. **Add operation outcome events** — `backup.completed`, `backup.failed`, `backup.restore.completed`, `backup.restore.failed`

### Actor model
39. **Replace `actor_token_id` with actor object** — `{type, token_id?, schedule_id?}`
40. **Actor on action events only** — outcome events always `{type: "system"}`

### Constants
41. **Create `permissions.go`** — all permission constants
42. **Create `events.go`** (update existing) — all event type constants
43. **Shared naming where applicable** — `gameserver.start` is both a permission and an event type

### Event bus
44. **Rename `EventBroadcaster` → `EventBus`**
45. **SSE filtering** — `?types=` query param, default `status_changed`
46. **Event storage** — `events` table with configurable retention (default 30 days)
47. **Event history endpoint** — `GET /api/events` with type/gameserver/time filters
