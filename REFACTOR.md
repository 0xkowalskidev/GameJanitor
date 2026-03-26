# Refactor Tracking

## Architecture

```
api/handler/    → controller/*     → store interfaces → model/
(HTTP transport)  (business logic)   (data access)      (pure types)
```

All controller sub-packages use Store interfaces, not `*sql.DB`.
Cross-domain data access goes through sibling interfaces.
Event types live in `controller/` parent.
`store.DB` embeds all domain stores — single entry point for DB access.
`model/` is pure types + validation + constants. Zero SQL.

---

## All Complete

### Foundation
- [x] model/ renamed, validate/naming/netinfo/tlsutil → pkg/, constants inlined, handlers → handler
- [x] worker/logparse/ extracted
- [x] controller/ parent: EventBus, errors, status constants, event types

### Controller domain extractions (all with store interfaces)
- [x] controller/auth/ — AuthService, permissions, token context
- [x] controller/settings/ — SettingsService
- [x] controller/event/ — EventHistoryService + EventStoreSubscriber
- [x] controller/orchestrator/ — Dispatcher, Registry, ControllerGRPC, WorkerNodeService
- [x] controller/gameserver/ — GameserverService + lifecycle + ports + migration + inspect + console + file
- [x] controller/backup/ — BackupService + LocalStorage/S3Storage
- [x] controller/status/ — StatusManager, QueryService, ReadyWatcher, StatsPoller, StatusSubscriber
- [x] controller/schedule/ — ScheduleService + Scheduler
- [x] controller/webhook/ — WebhookWorker + WebhookEndpointService
- [x] controller/mod/ — ModService + ModSource impls

### Store layer
- [x] store/ package with 9 domain stores
- [x] store.DB unified struct — single `store.New(db)` replaces all composite wiring

### Cleanup
- [x] model/ DB functions removed — model/ is pure types, zero `database/sql`
- [x] sftp/ uses store interface instead of `*sql.DB`
- [x] EventActor() added to WebhookEvent interface — extractActor type switch eliminated
- [x] Docker UID/GID constants deduplicated into model/

### Remaining (optional, discuss first)
- [ ] **Store method naming** — `GameserverStore.ListGameservers` could be `GameserverStore.List`. Large churn for naming preference. Current names are unambiguous and grep-able.
