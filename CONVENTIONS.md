# Code Conventions

This document defines the canonical code patterns used across the gamejanitor codebase. All contributions must follow these conventions. When in doubt, match what the surrounding code does — consistency beats personal preference.

---

## Go Style

### Naming

- **"gameserver"** everywhere — never "server" when referring to a game server
- `camelCase` for Go identifiers, `snake_case` for JSON tags and database columns
- Constructors: `NewXxxService(deps...) *XxxService`
- Struct fields: lowercase, abbreviated where obvious (`svc`, `log`, `db`, `ctx`)
- Interfaces describe behavior: `Worker`, `BackupStore` — not `IWorker` or `WorkerInterface`
- Constants: `CamelCase` for exported, `camelCase` for unexported — no `SCREAMING_SNAKE`

### File Organization

- One primary type per file, named after the type (`gameserver.go`, `backup.go`)
- Within a file: struct → constructor → exported methods → unexported methods → helpers
- Related logic lives in the same file — don't scatter a feature across files for "separation of concerns"
- Single blank line between functions. No double blank lines.

### Comments

- **Delete any comment that restates what the code does.** `// GetUser returns the user` above `func GetUser()` is noise.
- Comment the **why**, not the what:
  - Decision rationale: why this approach over alternatives
  - Warnings/gotchas: "don't remove this, it prevents X"
  - Non-obvious behavior: "yes, this should be >= not >"
  - Workarounds: "workaround for bug in lib X"
  - External context: links to specs, RFCs, tickets
- Package-level doc comments are fine on interfaces and complex types

### Imports

Standard library first, then external deps, then internal packages. `goimports` handles this.

---

## Error Handling

### Wrapping

Always wrap errors with context using `fmt.Errorf` and `%w`:

```go
return fmt.Errorf("creating gameserver %s: %w", id, err)
```

Format: `"verb-ing noun [identifier]: %w"` — lowercase, gerund form, includes relevant ID.

### Service Errors

The service layer uses typed errors that map to HTTP status codes:

```go
return ErrNotFoundf("gameserver %s not found", id)
return ErrBadRequest("name and game_id are required")
return ErrConflict("port already in use")
```

Handlers extract the status code via `serviceErrorStatus(err)`.

### Not-Found Pattern

Model `Get*` functions return `(nil, nil)` when a record doesn't exist — not an error. Callers check:

```go
gs, err := models.GetGameserver(db, id)
if err != nil {
    return err
}
if gs == nil {
    return ErrNotFoundf("gameserver %s not found", id)
}
```

### Don't Swallow Errors

Never `_ = someFunc()` if the error matters. If you intentionally ignore an error, log it:

```go
if err := w.RemoveContainer(ctx, id); err != nil {
    s.log.Warn("failed to remove container", "id", id, "error", err)
}
```

---

## Logging

### Library

`log/slog` everywhere. Injected as `*slog.Logger` via constructor, stored as `log` field.

Exception: `db/db.go` uses package-level `slog` since it runs during startup before DI is wired.

### Format

```go
s.log.Info("gameserver started", "id", id, "container_id", containerID[:12])
s.log.Warn("failed to stop container", "id", id, "error", err)
s.log.Error("update-server failed", "id", id, "exit_code", exitCode)
s.log.Debug("no stale container to remove", "name", containerName)
```

- Messages: lowercase, no trailing period, present tense
- Keys: `snake_case`
- Always include relevant IDs and the `error` key for failures

### Levels

| Level | Use |
|-------|-----|
| `Debug` | Expected misses, skipped branches, verbose diagnostics |
| `Info` | State transitions, completed operations, significant events |
| `Warn` | Non-fatal failures, degraded behavior, things that should be investigated |
| `Error` | Critical failures, data loss risk, things that need immediate attention |

### What to Log

- State transitions: `"gameserver status changed"`, `"backup created"`, `"worker registered"`
- Operation start/end for long operations: `"migrating gameserver"` ... `"gameserver migrated"`
- Non-obvious decisions: `"worker skipped during placement"` with `"reason"`
- Don't log every function entry/exit — only operations that matter

---

## Models Layer (`internal/models/`)

### Pattern

Package-level functions taking `*sql.DB` — not methods on a repository struct:

```go
func GetGameserver(db *sql.DB, id string) (*Gameserver, error)
func ListGameservers(db *sql.DB, filter GameserverFilter) ([]Gameserver, error)
func CreateGameserver(db *sql.DB, gs *Gameserver) error
func UpdateGameserver(db *sql.DB, gs *Gameserver) error
func DeleteGameserver(db *sql.DB, id string) error
```

### Struct Tags

All model structs get JSON tags:

```go
type Gameserver struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Status string `json:"status"`
}
```

Use `json:"-"` for sensitive fields (hashed passwords, tokens).

### SQL

- Parameterized queries only — never concatenate user input
- `WHERE 1=1` pattern for dynamic filters with optional conditions
- `COALESCE` for nullable aggregates: `SELECT COALESCE(SUM(x), 0)`
- Update/Delete check `RowsAffected()` and return "not found" if 0

### Scan Helpers

When a model has JSON columns (stored as TEXT in SQLite), use a `scan*` helper:

```go
func scanGameserver(scan func(dest ...any) error) (Gameserver, error) {
    var gs Gameserver
    var portsStr, envStr string
    err := scan(&gs.ID, ..., &portsStr, &envStr, ...)
    if err != nil {
        return gs, err
    }
    gs.Ports = json.RawMessage(portsStr)
    return gs, nil
}
```

---

## Service Layer (`internal/service/`)

### Structure

Each service is a struct with injected dependencies:

```go
type BackupService struct {
    db          *sql.DB
    dispatcher  *worker.Dispatcher
    settingsSvc *SettingsService
    log         *slog.Logger
}

func NewBackupService(db *sql.DB, ...) *BackupService {
    return &BackupService{db: db, ...}
}
```

### Settings Pattern

Every setting follows: ENV override → DB value → default.

```go
func (s *SettingsService) GetMaxBackups() int {
    if v := os.Getenv("GJ_MAX_BACKUPS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    v, err := models.GetSetting(s.db, SettingMaxBackups)
    if err != nil || v == "" {
        return DefaultMaxBackups
    }
    n, err := strconv.Atoi(v)
    if err != nil {
        return DefaultMaxBackups
    }
    return n
}
```

Each setting has: `GetX()`, `SetX()`, `IsXFromEnv()`. String settings that can be cleared also have `ClearX()`.

### Status Constants

Defined in `common.go`. All lowercase strings matching the database values:

```go
const (
    StatusStopped  = "stopped"
    StatusRunning  = "running"
    StatusError    = "error"
    // ...
)
```

---

## Worker Layer (`internal/worker/`)

### Interface

`Worker` is the core interface — implemented by `LocalWorker` (same process) and `RemoteWorker` (gRPC proxy). The `Dispatcher` routes to the correct worker based on gameserver-to-node assignment.

### Type Mirroring

`worker.ContainerOptions` and `docker.ContainerOptions` are separate types. `LocalWorker` converts between them. This keeps the worker interface independent of the Docker SDK.

---

## Web Layer (`internal/web/`)

### API Handlers (`handlers/`)

JSON API handlers follow this pattern:

```go
func (h *GameserverHandlers) Get(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    gs, err := h.svc.GetGameserver(id)
    if err != nil {
        h.log.Error("getting gameserver", "id", id, "error", err)
        respondError(w, serviceErrorStatus(err), err.Error())
        return
    }
    if gs == nil {
        respondError(w, http.StatusNotFound, "gameserver "+id+" not found")
        return
    }
    respondOK(w, gs)
}
```

### Response Envelope

All API responses use the envelope format:

```json
{"status": "ok", "data": {...}}
{"status": "error", "error": "message"}
```

Via helpers: `respondOK(w, data)`, `respondCreated(w, data)`, `respondError(w, code, msg)`, `respondNoContent(w)`.

### Page Handlers

Page handlers build a `map[string]any` data map and call `renderer.Render()`:

```go
func (h *PageDashboardHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
    // ... build data ...
    h.renderer.Render(w, r, "dashboard", data)
}
```

The renderer injects common data (CSRF token, auth state, network info) automatically.

### Middleware

Chi middleware pattern. Auth, rate limiting, audit logging, CSRF are all middleware. Middleware that needs services takes them as constructor args:

```go
func AuthMiddleware(authSvc *service.AuthService, settingsSvc *service.SettingsService) func(http.Handler) http.Handler {
```

### Routing

- `chi.Router` with nested `Route()` for grouping
- API routes under `/api/` — JSON, no CSRF, Bearer auth
- Page routes at root — HTML, CSRF protected, cookie auth
- Permission middleware applied per-route: `r.With(requireAdmin).Post(...)`

---

## CLI Layer (`cmd/cli/`)

### Pattern

Cobra commands with `RunE`. Each resource gets its own file (`gameservers.go`, `backups.go`).

```go
var gameserversListCmd = &cobra.Command{
    Use:   "list",
    Short: "List gameservers",
    RunE: func(cmd *cobra.Command, args []string) error {
        resp, err := apiGet("/api/gameservers")
        if err != nil {
            return exitError(err)
        }
        // ...
    },
}
```

### API Client

`client.go` provides `apiGet`, `apiPost`, `apiPatch`, `apiDelete` — all return `(*apiResponse, error)`.

### Output

- `--json` flag: `printJSONResponse(resp)` — raw API response
- Human output: `tabwriter` for tables, `fmt.Fprintf(w, "Key:\tValue\n")` for key-value

### Error Display

`exitError(err)` wraps errors for user-facing display. Don't expose internal details.

---

## Database

### SQLite

- WAL mode, 5s busy timeout, foreign keys ON
- Single writer (MaxOpenConns=1) — SQLite doesn't support concurrent writes
- Pure Go driver (`modernc.org/sqlite`) — no CGO dependency
- Migrations in `internal/db/migrations/` — numbered SQL files, applied in order

### Schema Changes

Pre-release: modify the initial migration directly (`001_initial.sql`). No new migration files until after release.

---

## Templates

### Engine

Go `html/template` with embedded filesystem (`embed.FS`). Layout + partials + page templates.

### HTMX

Page routes support full-page and partial (HTMX) rendering. The renderer checks `HX-Request` header and renders only the `content` block for HTMX requests.

### Template Functions

Defined in `render.go`: `statusColor`, `formatTime`, `formatBytes`, `rawJS`, `jsonPretty`, `formatMB`, etc. `rawJS` only receives `json.Marshal` output — safe from XSS.

---

## Security

- **Input validation at boundaries**: API handlers validate before calling service layer
- **Parameterized SQL**: Always `?` placeholders, never string concatenation
- **CSRF**: gorilla/csrf on all page routes, token in `X-CSRF-Token` header
- **Auth**: Bearer token (API) or `_token` cookie (web), bcrypt hashed in DB
- **Security headers**: X-Frame-Options, X-Content-Type-Options, CSP, Referrer-Policy on all responses
- **SFTP passwords**: bcrypt hashed, show-once on create, regenerate endpoint
- **ENV override for secrets**: Sensitive settings can be set via environment variables to avoid DB storage

---

## Testing

Tests go in `*_test.go` files alongside the code they test. No separate test directories.

---

## Dependencies

- **Web**: chi (router), gorilla/csrf, htmx (frontend)
- **Docker**: docker/docker SDK
- **Database**: modernc.org/sqlite (pure Go)
- **CLI**: spf13/cobra
- **gRPC**: google.golang.org/grpc + protobuf
- **Crypto**: golang.org/x/crypto/bcrypt
- **Build**: Nix flake

Minimize dependencies. Don't add a framework when a library suffices. Don't add a library when the standard library works.

