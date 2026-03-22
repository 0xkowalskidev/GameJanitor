# Gamejanitor Specification

Game server management platform. Single Go binary that combines an API server with a container orchestrator. Manages Docker-based game server lifecycle, multi-node placement, backups, scheduling, and event-driven webhooks.

## Architecture

**Single binary, two roles:**
- **Controller** — API server, dispatcher, event bus, scheduler, webhook delivery. One per cluster.
- **Worker** — Docker container management, volume operations, SFTP. Runs on each node.

In standalone mode (single node), both roles run in the same process with a `LocalWorker`. In multi-node mode, remote workers connect via gRPC and are managed by the controller's `Dispatcher`.

**Key components:**
- `EventBus` — in-memory pub/sub. All state changes flow through it. Consumers: StatusSubscriber (derives status), WebhookWorker (external delivery), EventStoreSubscriber (DB persistence), SSE handler (browser streaming).
- `Dispatcher` — selects worker nodes for gameserver placement. Scores by available resources, filters by tags. Serialized via mutex to prevent port/capacity races.
- `StatusSubscriber` — single point of status derivation. Maps lifecycle events to 7 gameserver statuses. Publishes derived `status_changed` events.
- `BackupStore` — abstraction over local filesystem and S3-compatible storage for backups and migration data.

## Database

SQLite. Single file at `{dataDir}/gamejanitor.db`. Schema in `internal/db/migrations/001_initial.sql`.

### Tables

**gameservers** — Core resource.
- `id`, `name`, `game_id`, `ports` (JSON), `env` (JSON)
- Resources: `memory_limit_mb`, `cpu_limit`, `cpu_enforced`, `storage_limit_mb`, `backup_limit`
- Container: `container_id`, `volume_name`, `status`, `error_reason`
- Placement: `port_mode` (auto/manual), `node_id`, `node_tags` (JSON)
- Auth: `sftp_username`, `hashed_sftp_password`
- State: `installed`, `auto_restart`
- Timestamps: `created_at`, `updated_at`

**schedules** — Cron-based recurring tasks per gameserver.
- `id`, `gameserver_id`, `name`, `type` (restart/backup/command/update), `cron_expr`, `payload` (JSON), `enabled`, `last_run`, `next_run`, `created_at`

**backups** — Gameserver volume snapshots.
- `id`, `gameserver_id`, `name`, `size_bytes`, `status` (in_progress/completed/failed), `error_reason`, `created_at`

**tokens** — API authentication.
- `id`, `name`, `hashed_token`, `scope` (admin/custom/worker), `gameserver_ids` (JSON), `permissions` (JSON), `created_at`, `last_used_at`, `expires_at`

**worker_nodes** — Multi-node configuration.
- `id`, `lan_ip`, `external_ip`, `port_range_start`, `port_range_end`
- Limits: `max_memory_mb`, `max_cpu`, `max_storage_mb`
- State: `cordoned`, `tags` (JSON), `sftp_port`, `last_seen`
- Timestamps: `created_at`, `updated_at`

**settings** — Key-value global configuration.
- `key`, `value`, `updated_at`

**webhook_endpoints** — Multi-endpoint webhook configuration.
- `id`, `description`, `url`, `secret`, `events` (JSON filter), `enabled`, `created_at`, `updated_at`

**webhook_deliveries** — Persistent delivery queue with retry.
- `id`, `webhook_endpoint_id`, `event_type`, `payload` (JSON), `state` (pending/delivered/failed), `attempts`, `last_attempt_at`, `next_attempt_at`, `last_error`, `created_at`

**events** — Event history for debugging.
- `id`, `event_type`, `gameserver_id`, `actor` (JSON), `data` (JSON), `created_at`

## Gameserver Statuses

7 statuses reflecting container state:

| Status | Meaning |
|---|---|
| `stopped` | Container not running |
| `installing` | Pulling game image |
| `starting` | Container being created / process launching |
| `started` | Process running, waiting for ready check |
| `running` | Ready check passed, accepting players |
| `stopping` | Container being stopped |
| `error` | Something failed (reason in `error_reason`) |

No operation-specific statuses. The action event (start, update, reinstall, migrate) tells WHY the lifecycle is happening.

## Resources

### Per-Gameserver

| Field | Docker enforced? | Dispatcher uses? | Default |
|---|---|---|---|
| `memory_limit_mb` | Always (OOM kill) | Yes | Game recommended |
| `cpu_limit` | When `cpu_enforced=true` | Yes (when > 0) | 0 (unlimited) |
| `storage_limit_mb` | Never (soft/informational) | Yes (when > 0) | null |
| `backup_limit` | Retention enforcement | No | null (global default) |

### Per-Node

| Field | Purpose |
|---|---|
| `max_memory_mb` | Dispatcher placement cap |
| `max_cpu` | Dispatcher placement cap |
| `max_storage_mb` | Dispatcher placement cap |
| `tags` | Placement filtering (k8s-style) |

### Settings

- `require_memory_limit` — reject create/update with 0 memory
- `require_cpu_limit` — reject create/update with 0 CPU
- `require_storage_limit` — reject create/update with 0 storage

## Events

### Event Types

**Action events** — user/schedule initiated, carry actor:
- `gameserver.create`, `gameserver.update`, `gameserver.delete`
- `gameserver.start`, `gameserver.stop`, `gameserver.restart`
- `gameserver.update_game`, `gameserver.reinstall`, `gameserver.migrate`
- `backup.create`, `backup.delete`, `backup.restore`
- `schedule.create`, `schedule.update`, `schedule.delete`

**Lifecycle outcome events** — system, drive status changes:
- `gameserver.image_pulling`, `gameserver.image_pulled`
- `gameserver.container_creating`, `gameserver.container_started`
- `gameserver.ready`
- `gameserver.container_stopping`, `gameserver.container_stopped`
- `gameserver.container_exited`, `gameserver.error`

**Operation outcome events** — system:
- `status_changed` (derived by StatusSubscriber)
- `backup.completed`, `backup.failed`
- `backup.restore.completed`, `backup.restore.failed`
- `worker.connected`, `worker.disconnected`
- `schedule.task.completed`, `schedule.task.failed`

### Actor Model

Every event carries an `actor` object:

| Type | Meaning |
|---|---|
| `token` | Authenticated request. Includes `token_id`. |
| `schedule` | Scheduled task. Includes `schedule_id`. |
| `system` | Gamejanitor acted autonomously (crash recovery, auto-restart, async outcomes). |
| `anonymous` | Auth disabled, no identity. |

Action events carry the real actor. Outcome events are always `system`.

### Delivery

Events flow through the `EventBus` to four consumers:
1. **StatusSubscriber** — derives gameserver status, publishes `status_changed`
2. **WebhookWorker** — matches events to webhook endpoints, persistent delivery with exponential backoff (24 attempts, ~24h retry window)
3. **EventStoreSubscriber** — persists all events to `events` table
4. **SSE handler** — streams to connected clients, filterable by `?types=`

## Authentication

### Token Types

Single token type with permissions list. Scope is a UI hint, not used for authorization.

| Preset | Scope | Permissions | Gameserver filter |
|---|---|---|---|
| Admin | `admin` | All | `[]` (all) |
| Custom | `custom` | Subset | `[]` or specific IDs |
| Worker | `worker` | `worker.connect` | N/A |

`[]` gameserver_ids = access to all gameservers.

### Permissions

**Gameserver lifecycle:** `gameserver.create`, `gameserver.update`, `gameserver.delete`, `gameserver.start`, `gameserver.stop`, `gameserver.restart`

**Gameserver access:** `gameserver.files.read`, `gameserver.files.write`, `gameserver.logs`, `gameserver.command`

**Gameserver config:** `gameserver.edit.name`, `gameserver.edit.env`

**Backups:** `backup.create`, `backup.delete`, `backup.restore`, `backup.download`

**Schedules:** `schedule.create`, `schedule.update`, `schedule.delete`

**Cluster:** `settings.view`, `settings.edit`, `tokens.manage`, `nodes.manage`, `webhooks.manage`

**Worker:** `worker.connect`

### Middleware

- `AuthMiddleware` — validates Bearer token or `_token` cookie. Sets token in context.
- `RequireClusterPermission(permission)` — checks token has the cluster permission.
- `RequirePermission(permission)` — checks token has permission on the gameserver from URL param.
- `RequireGameserverAccess` — checks token has any access to the gameserver.

Non-admin tokens cannot modify resource, placement, or port fields on gameservers.

## API

All responses use envelope: `{"status": "ok", "data": ...}` or `{"status": "error", "error": "message"}`.

Internal errors return `"internal server error"` — details in server logs only.

### Gameservers

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/gameservers` | — | List (filtered by token gameserver_ids) |
| POST | `/api/gameservers` | `gameserver.create` | Create, returns SFTP password (201) |
| GET | `/api/gameservers/{id}` | access | Get |
| PATCH | `/api/gameservers/{id}` | `gameserver.edit.*` | Update (auto-migrates if overcommit) |
| DELETE | `/api/gameservers/{id}` | `gameserver.delete` | Delete (204) |
| POST | `/api/gameservers/{id}/start` | `gameserver.start` | Start |
| POST | `/api/gameservers/{id}/stop` | `gameserver.stop` | Stop |
| POST | `/api/gameservers/{id}/restart` | `gameserver.restart` | Restart |
| POST | `/api/gameservers/{id}/update-game` | `gameserver.update` | Update game |
| POST | `/api/gameservers/{id}/reinstall` | `gameserver.update` | Reinstall |
| POST | `/api/gameservers/{id}/migrate` | `gameserver.create` | Migrate to node |
| GET | `/api/gameservers/{id}/status` | access | Status + container info |
| GET | `/api/gameservers/{id}/query` | access | Player/game query data |
| GET | `/api/gameservers/{id}/stats` | access | CPU/memory/storage stats |
| GET | `/api/gameservers/{id}/logs` | `gameserver.logs` | Container logs |
| POST | `/api/gameservers/{id}/command` | `gameserver.command` | Send console command |
| POST | `/api/gameservers/bulk` | `gameserver.create` | Bulk start/stop/restart |

### Backups

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/gameservers/{id}/backups` | `backup.create` | List |
| POST | `/api/gameservers/{id}/backups` | `backup.create` | Create (202, async) |
| GET | `/api/gameservers/{id}/backups/{bid}/download` | `backup.download` | Download tar.gz |
| POST | `/api/gameservers/{id}/backups/{bid}/restore` | `backup.restore` | Restore |
| DELETE | `/api/gameservers/{id}/backups/{bid}` | `backup.delete` | Delete (204) |

### Schedules

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/gameservers/{id}/schedules` | `schedule.create` | List |
| POST | `/api/gameservers/{id}/schedules` | `schedule.create` | Create (201) |
| GET | `/api/gameservers/{id}/schedules/{sid}` | `schedule.create` | Get |
| PATCH | `/api/gameservers/{id}/schedules/{sid}` | `schedule.update` | Update |
| DELETE | `/api/gameservers/{id}/schedules/{sid}` | `schedule.delete` | Delete (204) |

### Files

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/gameservers/{id}/files` | `files.read` | List directory |
| GET | `/api/gameservers/{id}/files/content` | `files.read` | Read file |
| PUT | `/api/gameservers/{id}/files/content` | `files.write` | Write file |
| DELETE | `/api/gameservers/{id}/files` | `files.write` | Delete path |
| GET | `/api/gameservers/{id}/files/download` | `files.read` | Download file |
| POST | `/api/gameservers/{id}/files/upload` | `files.write` | Upload file |
| POST | `/api/gameservers/{id}/files/rename` | `files.write` | Rename |
| POST | `/api/gameservers/{id}/files/mkdir` | `files.write` | Create directory |

### Webhooks

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/webhooks` | `webhooks.manage` | List endpoints |
| POST | `/api/webhooks` | `webhooks.manage` | Create endpoint (201) |
| GET | `/api/webhooks/{id}` | `webhooks.manage` | Get endpoint |
| PATCH | `/api/webhooks/{id}` | `webhooks.manage` | Update endpoint |
| DELETE | `/api/webhooks/{id}` | `webhooks.manage` | Delete endpoint (204) |
| POST | `/api/webhooks/{id}/test` | `webhooks.manage` | Send test payload |
| GET | `/api/webhooks/{id}/deliveries` | `webhooks.manage` | Delivery history |

### Events

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/events` | — | SSE stream (`?types=` filter) |
| GET | `/api/events/history` | — | Query event history |

### Settings

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/settings` | `settings.view` | Get all settings |
| PATCH | `/api/settings` | `settings.edit` | Update settings |

### Tokens

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/tokens` | `tokens.manage` | List |
| POST | `/api/tokens` | `tokens.manage` | Create (201) |
| DELETE | `/api/tokens/{id}` | `tokens.manage` | Delete (204) |

### Workers

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/workers` | `nodes.manage` | List |
| GET | `/api/workers/{id}` | `nodes.manage` | Get with stats |
| PATCH | `/api/workers/{id}/port-range` | `nodes.manage` | Set port range |
| DELETE | `/api/workers/{id}/port-range` | `nodes.manage` | Clear port range |
| PATCH | `/api/workers/{id}/limits` | `nodes.manage` | Set resource limits |
| DELETE | `/api/workers/{id}/limits` | `nodes.manage` | Clear limits |
| POST | `/api/workers/{id}/cordon` | `nodes.manage` | Cordon |
| DELETE | `/api/workers/{id}/cordon` | `nodes.manage` | Uncordon |
| PATCH | `/api/workers/{id}/tags` | `nodes.manage` | Set tags |
| DELETE | `/api/workers/{id}/tags` | `nodes.manage` | Clear tags |

### Games

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/api/games` | — | List available games |
| GET | `/api/games/{id}` | — | Get game definition |

## Webhook Payloads

```json
{
  "version": 1,
  "id": "uuid",
  "timestamp": "2026-03-22T12:00:00Z",
  "event_type": "gameserver.start",
  "data": { ... }
}
```

Each event type has specific data fields. Endpoints filter events by glob patterns (`["*"]`, `["gameserver.*"]`, `["status_changed", "backup.*"]`). Delivery uses HMAC-SHA256 signing via `X-Webhook-Signature` header.

## Multi-Node

### Placement

1. Filter nodes by gameserver's `node_tags` (all must match)
2. Skip cordoned nodes
3. Score by minimum headroom across memory/CPU/storage
4. Try candidates in order: check limits, allocate ports
5. Serialized by `placementMu` to prevent port/capacity races

### Auto-Migration

When `UpdateGameserver` changes resources and the current node can't fit:
1. Snapshot old values
2. Find new node via dispatcher (respects tags)
3. Write new values, trigger `MigrateGameserver` async
4. If no node found, publish error event, return error (old values unchanged)

### Migration

Uses backup store as intermediary (not in-memory):
1. Stop gameserver on source
2. Tar + gzip volume → store (under `migrations/` prefix)
3. Store → restore on target
4. Reallocate ports on target
5. Update node_id in DB
6. Cleanup source volume and migration data (on success only)

### Worker Communication

gRPC with mTLS. Workers register with the controller, send heartbeats. Controller initiates all streaming operations. Worker auth via dedicated worker tokens.

## SFTP

Built-in SFTP server for file management. Two modes:
- **Controller-proxied** — SFTP connections hit the controller, operations proxied via gRPC to the worker
- **Direct-to-worker** — SFTP connections go directly to the worker node

Auth via SFTP username (auto-generated per gameserver) + password (bcrypt hashed, show-once on create).

## Game Definitions

YAML files in `internal/games/data/{game-id}/game.yaml`. Define:
- `name`, `base_image`, `recommended_memory_mb`
- `default_ports` with protocol, name, autogenerate rules
- `default_env` with key, default, description, options, system flag, triggers_install
- `ready_pattern` regex for log-based readiness detection
- `disabled_capabilities` to opt out of query, console, save, etc.

## Settings

All settings follow the pattern: environment variable override → database value → default.

| Setting | Default | Description |
|---|---|---|
| `connection_address` | auto-detect | Public IP for gameserver connections |
| `port_range_start/end` | 27000-28999 | Auto-allocated port range |
| `port_mode` | auto | Default port allocation mode |
| `max_backups` | 10 | Global backup retention limit |
| `auth_enabled` | false | Enable token-based auth |
| `localhost_bypass` | true | Skip auth for localhost requests |
| `rate_limit_enabled` | false | Enable rate limiting |
| `rate_limit_per_ip/token/login` | 20/10/10 | Rate limit thresholds |
| `trust_proxy_headers` | false | Trust X-Forwarded-For |
| `event_retention_days` | 30 | Event history retention |
| `require_memory/cpu/storage_limit` | false | Enforce non-zero resources |

---

# Issues & Recommendations

## Problems

### P1: ValidateToken scans all tokens with bcrypt (HIGH)
`AuthService.ValidateToken` lists ALL tokens from DB and bcrypt-compares each one sequentially. With 100 tokens, every API request does 100 bcrypt comparisons. This is O(n) per request with expensive crypto. Should use a token prefix/hash lookup to find the candidate, then bcrypt-verify only that one.

### P2: Backup restore is still synchronous (MEDIUM)
`CreateBackup` was made async but `RestoreBackup` still blocks the HTTP request. Restore stops the server, downloads from store, decompresses, restores volume, optionally restarts. For large servers this can take minutes. Should follow the same async pattern as CreateBackup with status tracking.

### P3: QueryService polls at fixed 5s interval regardless of game (LOW)
All games poll at 5s. Some games (like Minecraft) respond slowly to queries. Others (like CS2) can handle faster polling. Should be configurable per-game in the game definition.

### P4: No graceful handling of EventBus subscriber backup (MEDIUM)
EventBus channels have 64-element buffer. If a subscriber (webhook worker, event store) falls behind, events are silently dropped. No logging, no metric. The `select/default` pattern in `Publish()` means slow consumers lose data without any indication.

### P5: settings.go boilerplate (LOW)
Every setting requires ~15 lines of boilerplate (constant, getter, fromEnv, setter). 30+ settings = 450+ lines of repetitive code. A table-driven approach could halve this. Already tracked as F9.

### P6: Webhook delivery processes sequentially (LOW)
`processPendingDeliveries` fetches 10 pending deliveries and delivers them one at a time. If one endpoint is slow (5s timeout), the other 9 wait. Should deliver in parallel with a worker pool or at least per-endpoint.

### P7: No API pagination on list endpoints (MEDIUM)
`GET /api/gameservers`, `GET /api/tokens`, `GET /api/webhooks` return all records. Fine for small deployments but businesses with hundreds of gameservers will get large responses. Should add `limit`/`offset` params. Event history already has pagination.

### P8: Scheduler has no way to view upcoming tasks (LOW)
The scheduler loads cron entries but there's no API to list "what's scheduled to run next" beyond the `next_run` field on each schedule. A `GET /api/schedules/upcoming` would be useful for monitoring.

## Simplification Opportunities

### S1: Worker gRPC wrappers are mechanical duplication
`remote.go` (413 lines) and `agent.go` (404 lines) mirror each other — every Worker interface method has a gRPC call on one side and a handler on the other. A code generator from the proto definition could produce these automatically.

### S2: Sidecar fileops fallback adds 230 lines
The sidecar container pattern for file operations (`local_fileops_sidecar.go`) is a fallback for when Docker volumes aren't directly accessible (rootless Docker, Docker Desktop). Could be documented as a known limitation instead of maintained as a parallel code path.

### S3: Page handlers (~2.5k lines) are throwaway
The page handlers (`page_*.go`) duplicate API handler logic with HTML response formatting. They exist only for the test UI. When the UI is rebuilt as a separate frontend, these can be deleted entirely.

## Performance Notes

### N1: SQLite single-writer
SQLite serializes all writes. Under load, concurrent gameserver creates queue on the placement mutex AND on SQLite writes. This is fine for the target scale (hundreds of gameservers, not thousands) but would need a different DB for massive deployments.

### N2: Event store writes on every event
Every event (including lifecycle events during start, which fire 5-6 times rapidly) writes to the events table. With many gameservers starting simultaneously, this could create write contention. The event store subscriber could batch writes with a short flush interval.

### P9: CSP allows unsafe-inline and unsafe-eval (LOW)
Content Security Policy includes `'unsafe-inline' 'unsafe-eval'` for script-src (required by Alpine.js). This weakens XSS protection. When the UI is rebuilt, consider a framework that doesn't require eval. Not a concern for API-only deployments.

### N3: Webhook endpoint lookup per event
`enqueueEvent` calls `models.ListEnabledWebhookEndpoints` for every event. With frequent events, this is many DB reads. Should cache endpoint list and invalidate on create/update/delete.
