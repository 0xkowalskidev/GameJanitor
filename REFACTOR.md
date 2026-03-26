# Refactor Tracking

## Architecture

```
api/handler/    → controller/*     → store interfaces → model/
(HTTP transport)  (business logic)   (data access)      (pure types)
```

All controller sub-packages use Store interfaces, not `*sql.DB`.
Cross-domain data access goes through sibling interfaces.
Event types live in `controller/` parent.

---

## Remaining Cleanup

### 1. Remove model/ DB functions (medium effort, high value)

model/ still has ~70 DB query functions duplicated in store/. Three production callers remain:

- `sftp/auth_local.go:22` — `model.GetGameserverBySFTPUsername(a.db, ...)`
- `cli/serve.go:338` — `model.PruneEvents(database, ...)`
- `cli/serve.go:349` — `model.PruneEvents(database, ...)`

Plus ~14 calls in service/ test files for direct DB setup.

**Fix:** Update 3 production callers to use stores, update test helpers, delete all
DB functions from model/. model/ becomes truly pure types + validation + constants.

- [ ] Add `GetGameserverBySFTPUsername` to store/gameserver.go (if not already there)
- [ ] Add `PruneEvents` to store/event.go (if not already there)
- [ ] Update `sftp/auth_local.go` to receive a store interface instead of `*sql.DB`
- [ ] Update `cli/serve.go` PruneEvents calls to use event store
- [ ] Update service/ test files to use store methods instead of model functions
- [ ] Delete all `func X(db *sql.DB, ...)` functions from model/*.go
- [ ] Delete `model/setting.go` entirely (100% DB queries, no types)
- [ ] Remove `database/sql` import from model files that no longer need it
- [ ] Run tests

### 2. Add EventActor() to WebhookEvent interface (low effort, eliminates type switch)

`controller/event/subscriber.go:extractActor()` type-switches on 6 event types to get the
Actor field. Adding `EventActor() Actor` to the `WebhookEvent` interface eliminates this.

- [ ] Add `EventActor() Actor` to `controller.WebhookEvent` interface
- [ ] Implement on all event types (action events return their Actor, lifecycle events return SystemActor)
- [ ] Replace `extractActor()` in subscriber.go with `event.EventActor()`
- [ ] Inline `extractData()` (it just returns its argument)
- [ ] Run tests

### 3. Unified store.DB struct (medium effort, cleaner wiring)

`cli/serve.go` has 7 anonymous composite structs to satisfy multi-store interfaces:

```go
scheduleStore := struct {
    *store.ScheduleStore
    *store.GameserverStore
}{store.NewScheduleStore(database), gsStore}
```

Replace with a single `store.DB` struct that embeds all domain stores:

```go
type DB struct {
    *GameserverStore
    *BackupStore
    *EventStore
    *ScheduleStore
    *TokenStore
    *WebhookStore
    *WorkerNodeStore
    *ModStore
    *SettingStore
}

func New(db *sql.DB) *DB { ... }
```

Services receive `*store.DB` which satisfies any Store interface via embedding.
Interface enforcement still limits what each service can call.

- [ ] Create `store/store.go` with `DB` struct and `New()` constructor
- [ ] Replace all composite structs in `cli/serve.go` with `store.New(database)`
- [ ] Replace all composite structs in `testutil/services.go` with same
- [ ] Remove `webhookGameserverLookup` struct in serve.go (store.DB satisfies it directly)
- [ ] Run tests

### 4. sftp/ store interface (low effort)

`sftp/auth_local.go` holds `*sql.DB` and calls `model.GetGameserverBySFTPUsername` directly.
Should receive a store interface.

- [ ] Define `Auth` interface in sftp/auth_local.go: `GetGameserverBySFTPUsername(username string) (*model.Gameserver, error)`
- [ ] Replace `db *sql.DB` with interface
- [ ] Update `cli/serve.go` to pass store
- [ ] Run tests

### 5. Store method naming (large effort, optional — discuss first)

Store methods repeat the domain name: `GameserverStore.ListGameservers`, `WebhookStore.ListWebhookEndpoints`.
Through interfaces this reads as `s.store.ListGameservers()` inside a gameserver service.

Cleaner: `List`, `Get`, `Create`, `Update`, `Delete`. The store type/interface already scopes the domain.

**Trade-off:** Touches every store method, every interface definition, and every call site.
Significant churn for a naming preference. The current names are unambiguous and grep-able.

- [ ] Discuss: is this worth the churn?

### 6. Docker UID/GID constant duplication (low effort)

`docker/docker.go` duplicates `gameserverUID/gameserverGID/gameserverPerm` from `worker/types.go`
to avoid a circular import. Comment says "Phase 2 of the refactor" — that's done.

- [ ] Move constants to `model/` or `pkg/container/` (both importable by docker/ and worker/)
- [ ] Remove duplicates
- [ ] Run tests

---

## Completed

### Phase 1–3: Foundation [DONE]
- model/ renamed from models/, validate/→pkg/validate/, naming/netinfo/tlsutil→pkg/
- constants/ deleted and inlined, api/handlers/→api/handler/
- worker/logparse/ extracted
- controller/ parent: EventBus, errors, status constants, event types

### Phase 4: Controller domain extractions [DONE]
- controller/auth/ — AuthService, permissions, token context (store interface)
- controller/settings/ — SettingsService (store interface)
- controller/event/ — EventHistoryService + EventStoreSubscriber (store interface)
- controller/orchestrator/ — Dispatcher, Registry, ControllerGRPC, WorkerNodeService (store interfaces)
- controller/gameserver/ — GameserverService + lifecycle + ports + migration + inspect + console + file (store interface)
- controller/backup/ — BackupService + LocalStorage/S3Storage (store interface)
- controller/status/ — StatusManager, QueryService, ReadyWatcher, StatsPoller, StatusSubscriber (store interface)
- controller/schedule/ — ScheduleService + Scheduler (store interface)
- controller/webhook/ — WebhookWorker + WebhookEndpointService (store interface)
- controller/mod/ — ModService + ModSource impls (store interface)

### Phase 5: Store layer [DONE]
- store/ package with 9 domain stores (gameserver, backup, event, schedule, token, webhook, worker_node, mod, setting)
- All controller sub-packages use store interfaces, zero *sql.DB
