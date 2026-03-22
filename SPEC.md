# Gamejanitor Specification

Current state of the project as of 2026-03-20. This document describes what exists, the invariants the system maintains, and how each user archetype interacts with it.

## Core Invariants

These are the guarantees the system maintains:

1. **Single binary, zero config for single-node.** Download, run `gamejanitor serve`, open browser. No Docker knowledge, no YAML, no CLI required for basic use.

2. **Same codebase, same binary for all deployment modes.** Standalone, controller, worker, controller+worker are all the same binary with a `--role` flag.

3. **Worker interface abstraction.** All container and volume operations go through the `Worker` interface. `LocalWorker` talks to Docker directly. `RemoteWorker` talks to a worker agent over gRPC. Services never know which one they're using.

4. **Games are data, not code.** Game definitions are YAML + shell scripts. Adding a game means adding a YAML file and scripts, not writing Go code. Embedded in the binary, overridable locally at `{dataDir}/games/{id}/`.

5. **Four shared base images.** Games don't have individual Docker images. They share `base` (Ubuntu), `steamcmd`, `java` (JDK 21), or `dotnet` (.NET 9). Game scripts are bind-mounted at runtime.

6. **Token-based auth, not user-based.** No user accounts. Scoped tokens grant permissions on specific gameservers. Businesses integrate with their own customer/billing systems.

7. **No auth by default.** Single-node users bind to localhost. Auth is opt-in. When enabled, localhost bypass is on by default. - Question: What is the bypass for?

8. **Soft resource enforcement.** Storage caps, memory caps, CPU caps, and backup limits block new operations but never kill running gameservers. - Question: Should this be configurable? Business/power user may want backups to not be blocked - handle manually, also do we need webhooks for stuff like this?

9. **Log-based ready detection.** Each game defines a `ready_pattern` regex. Status transitions from `started` to `running` when the pattern matches in container logs, or after 60s timeout. No probe-based health checks. - Question: What happens if it fails? If a user is using a custom jar or mod that has a different pattent? How do we make this fault tolerant and user choice tolerant

10. **Direct volume access with sidecar fallback.** File operations use direct filesystem access to Docker volume mount points when available (fast). Falls back to ephemeral Alpine sidecar containers when running in restricted environments. - Question: Any security concerns with our non docker storage?

11. **Streaming, not buffering.** Backup creation, restore, and file transfers stream through `io.Reader`/`io.Writer`. Multi-GB gameserver data never sits in memory.

12. **ENV overrides DB settings.** All settings follow the priority: ENV var > DB setting > default. When set via ENV, the UI marks the setting as read-only. - Question: Dont think the UI marks the setting as read only, would be a good solution though. Need to investigate this further.

13. **mTLS for multi-node gRPC.** Controller auto-generates a CA and server cert on first startup. Worker certs are generated via `gen-worker-cert` CLI command. All gRPC traffic is encrypted and mutually authenticated. Standalone mode has no gRPC. - Question whats the "add worker" flow? too complicated? Acceptable?

14. **Audit everything that mutates.** All POST/PATCH/PUT/DELETE requests are logged with action, resource, token identity, IP, and status code. Retention is configurable. - Question: Another good area for webhooks?

## Deployment Modes

| Mode | Web UI | Local Docker | gRPC Server | Connects to Controller |
|------|--------|-------------|-------------|----------------------|
| standalone | Yes | Yes | No | No |
| controller | Yes | No | Yes | No | Question: What happens if a controller tries to run a gameserver without a worker?
| worker | No | Yes | Yes | Yes |
| controller+worker | Yes | Yes | Yes | No | Question: is webui configurable? businesses wont want it (probably)

## Supported Games

Minecraft Java, Minecraft Bedrock, Rust, CS2, ARK, Valheim, 7 Days to Die, Palworld, Satisfactory, Terraria, Garry's Mod.

Each game defines: base image, recommended memory, ports (name/port/protocol), environment variables (with types, defaults, labels), ready pattern, disabled capabilities, and scripts (install, start, send-command, save, update).

## User Archetypes

### 1. Normal User (Newbie)

**Profile:** Wants to play Minecraft/Valheim/etc with friends. No sysadmin knowledge. Comfortable with a web browser.

**Setup:** - Question: Does this script actually exist?
```sh
curl -sSL https://raw.githubusercontent.com/warsmite/gamejanitor/master/install.sh | bash
gamejanitor serve
# Open http://localhost:8080
``` 

**What they see:**
- Dashboard with their gameservers
- Click "Create" — pick a game, give it a name, click create
- Ports auto-allocated, memory/CPU set to game defaults
- Click "Start" — watch status go from pulling → starting → started → running
- Connection info shown once running (IP:port to share with friends)
- Console tab for live logs and sending commands
- Files tab for uploading world saves, editing configs
- Backups tab for manual backup/restore
- Schedules tab for automated restarts and backups

**What they don't need to know about:** - Question: They do need to know port forwarding atm, would like to fix that in future with a reverse proxy network, but thats a big ask that needs to be tied to a business usecase. For now we need to ensure we have good "onboarding" to explain how to port forward and open firewall, which we partially have.
- Docker, containers, volumes
- Port ranges, CPU limits, storage caps
- Auth, tokens, rate limiting
- CLI, API, SFTP - Question: What do we lose if the user closes gamejanitor serve? i.e, what wont work that the user would expect to work (backups, scheudles, etc)
- Multi-node anything

**Key invariants for this user:**
- Zero config. Works out of the box.
- No auth hassle on localhost.
- Status is always visible and accurate.
- File browser and console work without SSH/SFTP knowledge. 

### 2. Power User

**Profile:** Runs their own hardware. Comfortable with CLI and APIs. Might have a NAS. Wants automation and scripting.

**Setup:** - Question: Should we be setting up systemd shit aswell for them? Need good docs obviously, add a todo
```sh
gamejanitor serve --port 8080 --sftp-port 2222
# Or via NixOS module 
```

**What they use:**

- **CLI** for everything: - Question: Not actually true, *can use cli for everything* still need to support them in webui aswell. where do we fall short?
  ```sh
  gamejanitor gameservers create --name "Rust Server" --game rust --memory 4096 Question: Do we support GB here? We probably should have m or g options intead of raw mb values
  gamejanitor gameservers start <id>
  gamejanitor backups create <id>
  gamejanitor schedules create <id> --name "nightly backup" --type backup --cron "0 3 * * *"
  ```

- **API** for custom integrations: - Question: Make sure we have a todo for implementing good docs
  ```sh
  curl -H "Authorization: Bearer <token>" http://localhost:8080/api/gameservers
  ```

- **SFTP** for file management: - Question: Why 2222 and not 22?
  ```sh
  sftp -P 2222 <gameserver-sftp-user>@localhost
  ```

- **Auth** when exposing beyond localhost: - Question: anyway to warn the user when exposing server?
  ```sh
  gamejanitor settings enable-auth
  gamejanitor tokens create --name "my-token"
  ```

- **S3 backups** to their NAS:
  ```sh
  GJ_S3_BUCKET=gamejanitor-backups \
  GJ_S3_ENDPOINT=minio.local:9000 \
  GJ_S3_ACCESS_KEY=... GJ_S3_SECRET_KEY=... \
  GJ_S3_PATH_STYLE=true GJ_S3_USE_SSL=false \
  gamejanitor serve
  ```

- **Schedules** for automated maintenance: - Question: Do we setup a restart/backup schedule by default? Should we?
  - Nightly backups
  - Weekly restarts
  - Periodic update checks via command schedules - Question: What does this refer to?

- **Bulk operations:**
  ```sh
  gamejanitor gameservers stop --all - Question: Can this be specified by node?
  gamejanitor gameservers start --all - Question: Do we handle node migrations (server migrate away from node), what about node "quarantine" or whatever the correct term is (stop new gs being added), what else is "expected"
  ```

**What they value:**
- Full API coverage (everything in the UI is in the API)
- JSON output (`--json` flag on all CLI commands) - Question: Do we actually have this?
- SFTP for scripted file management
- S3-compatible backup storage
- Cron-based scheduling
- Per-gameserver resource limits - Question: Do we implement this properly? Double check.

### 3. Business (Hosting Provider)

**Profile:** Runs game server hosting as a service. Multiple machines. Customers rent gameservers. Has their own billing/customer portal.

**Setup:**

Controller node:
```sh
gamejanitor serve --role controller --port 8080 --grpc-port 9090 --sftp-port 2222 - Question: Hows our fault tolerance, would they expect multiple controllers?
```

Worker nodes:
```sh
# On controller: generate worker cert
gamejanitor gen-worker-cert worker-1
# Copy ca.crt, worker-1.crt, worker-1.key to worker node Question: Do we need certs and a key?

# On worker:
GJ_GRPC_CA=/etc/gamejanitor/ca.crt \
GJ_GRPC_CERT=/etc/gamejanitor/worker-1.crt \
GJ_GRPC_KEY=/etc/gamejanitor/worker-1.key \
GJ_WORKER_TOKEN=<token> \
gamejanitor serve --role worker --controller 10.0.0.1:9090 --grpc-port 9090
```

**How they integrate with their billing system:**

1. Customer buys a "Minecraft 4GB" plan
2. Billing system calls API: - Question: is it a problem that these are two seperate commands? Is it a problem that the gameserver has to come first if not?
   ```
   POST /api/gameservers
   {"name": "Customer's MC Server", "game": "minecraft-java", "memory_limit_mb": 4096}
   ```
3. Billing system creates a scoped token for the customer:
   ```
   POST /api/tokens
   {"name": "customer-123", "gameserver_ids": ["<gs-id>"], "permissions": ["start","stop","restart","console","files","backups"]}
   ```
4. Customer uses the token in their panel (web UI, CLI, SFTP, or custom frontend hitting the API) Question: Customer would not have direct control over the token right? Or are you just saying that the hosts backend will use the token to interact with the server? 
5. Per-gameserver caps enforce plan limits:
   - `max_memory_mb` — customer can't increase beyond plan
   - `max_storage_mb` — disk usage capped
   - `max_backups` — backup count limited
   - `max_cpu` — CPU allocation capped

Question: How hard would it be to allow of control panel to be used by businesses? How extensible would it need to be?

**Multi-node management:**

- **Node capacity:** Per-worker limits on total memory and gameserver count. Placement algorithm picks worker with most headroom. - Question: No storage/cpu limits? What other limits should it have
- **Per-node port ranges:** Each worker gets its own port range to avoid conflicts. - Question: How would businesses expect to handle this, would they bring IPs and proxy or something, would they want domains, do we need to do more to support?
- **Per-node disk visibility:** Settings page shows disk total/available per worker. - Question: Webui would be used here? Or just api with businesses home made ui? Seems like this would be more for power users? 
- **Gameserver migration:** Move a gameserver between nodes via API when rebalancing.
- **mTLS:** All gRPC traffic between controller and workers is encrypted with auto-generated certificates.

**What they value:**
- API-first (their customer portal drives everything)
- Scoped tokens with per-gameserver permissions (no user model to manage) - Question: whats the flow for a customer upgrading their server (more memory/storage etc)
- Resource caps that enforce plan limits without killing running servers
- Multi-node with automatic placement - Question: Do we need better/more placement strategies?
- S3 backup storage (shared across nodes, no single point of failure) - Question: does this go from worker -> s3 or worker -> controller -> s3?
- Audit logging (who did what, when, from where)
- Rate limiting (protect API from abuse)
- SFTP per-gameserver credentials (customers get direct file access) - Question: Verify security is good here
- mTLS on gRPC (secure worker communication)

## Feature Matrix

| Feature | Newbie | Power User | Business |
|---------|--------|-----------|----------|
| Web UI | Primary interface | Occasionally | Admin/monitoring | Question: Do we really support admin/monitoring for business scale? + Its only on control node, businesses will have multiple clusters right?
| CLI | Not needed | Primary interface | Automation scripts |
| API | Not needed | Custom integrations | Customer portal backend |
| SFTP | Not needed | File management | Customer file access |
| Auth | Off (localhost) | On when exposed | Always on | Question: Do we have a good flow for token refresh? Will businesses expect to reguraly refresh user tokens?
| S3 backups | Not needed | NAS backup | Multi-node shared storage | Question: Where are local backups stored? datadir?
| Multi-node | No | No | Yes |
| Resource caps | Not needed | Self-imposed | Enforce plan limits |
| Schedules | Optional | Automation | Part of hosting plans |
| Audit log | Not needed | Optional | Compliance/support |
| Rate limiting | Not needed | Optional | Protect from abuse |
| mTLS | N/A | N/A | Required for multi-node |

## Resource Limits

### Per-Gameserver (admin-set, for hosting plans)
- `max_memory_mb` — memory allocation cap
- `max_cpu` — CPU allocation cap
- `max_backups` — backup count limit Question: Do we have a global default setting? Or does every one have to be configured
- `max_storage_mb` — disk usage cap (enforced before file writes, uploads, and backup creation) Question: Is it enforced? I thought we went soft enforce? (again, web hooks for businesses?)

### Per-Worker Node (multi-node)
- `max_memory_mb` — total memory allocation for the node Question: CPU? Storage? Does the controller know what the node can support? 
- `max_gameservers` — gameserver count limit Question: Why?

### Global Settings
- `max_backups` — default backup limit (per-gameserver override takes priority)
- Port range (start/end, overridable per-node)

## Backup Storage

Two backends, selected at startup:

- **Local** (default): files at `{dataDir}/backups/{gameserverID}/{backupID}.tar.gz` Question: will default dataDir allow this on all OS? Do we show dataDIr path on settings (readonly)? How is newbie meant to know where they are stored (even if they shouldnt care)
- **S3** (when `GJ_S3_BUCKET` set): objects at `backups/{gameserverID}/{backupID}.tar.gz` in the configured bucket. Works with any S3-compatible endpoint (AWS, MinIO, R2, Backblaze B2).

## Status Lifecycle Question: We should make "display statuses" (Running, Pulling, Starting, etc)

```
stopped → pulling → starting → started → running
                                              ↓
                                           stopping → stopped
                                              ↓
                                            error
```
Question: Any other forseeable statuses?
- `pulling`: Docker image being pulled
- `starting`: container created and starting
- `started`: container running, waiting for ready pattern
- `running`: ready pattern matched (or 60s timeout)
- `stopping`: graceful shutdown in progress (save-server called first if game supports it)
- `error`: unexpected failure, `error_reason` field has details

## Security Model

1. **No auth by default** — localhost-only, no tokens needed
2. **Opt-in auth** — enable via settings, generates admin token
3. **Localhost bypass** — even with auth on, 127.0.0.1 requests skip auth (configurable) Question: Why?
4. **Scoped tokens** — admin creates tokens with specific gameserver + permission grants
5. **CSRF protection** — on all HTML form submissions (not API) - Question: Do we have security in place for all of the big 10?  Add a todo to do a full security audit
6. **Rate limiting** — three-tier: per-IP, per-token, per-login (disabled by default)
7. **Audit logging** — all mutations logged with identity and IP Question: What is "identify", we just going to show peoples tokens? Should we have token labels?
8. **mTLS on gRPC** — auto-generated CA, mutual certificate verification for all worker traffic
9. **Bcrypt token hashing** — tokens stored as hashes, never plaintext - Question: how are rcon/sftp passwords and other "soft" secrets stored?
10. **Path traversal protection** — all file operations validated to stay within /data

## Worker SFTP

Workers can run their own SFTP server (set `GJ_SFTP_PORT` on the worker). When a worker runs SFTP, it validates credentials by calling back to the controller via gRPC. File operations use direct volume access on the worker — no gRPC round-trip for file data. On the controller, SFTP routes through the dispatcher to the correct worker. - Question: Is this optional? Should it be? Does it still use dispatcher and not go direct to worker?

## Query Polling (GJQ)

Games that support query (most do) get polled every 5 seconds via the GJQ library. Provides: `PlayersOnline`, `MaxPlayers`, `Players[]` (names), `Map`, `Version`. Displayed in the web UI on the gameserver detail page. Query polling only runs for gameservers in `running` status.

## Status Recovery Question: Do we need to think more about recovery/fault tolerance?

On startup, `RecoverOnStartup()` reconciles DB state with Docker reality. For each non-terminal gameserver:
- Container running >60s → mark `running`
- Container recently started (<60s) → mark `started` (await ready pattern)
- Container exited/dead → mark `stopped`
- Multi-node: gameservers on offline workers are skipped until the worker re-registers

## Container Lifecycle
Question: is docker restart policy actually disabled? Why?
Gamejanitor manages the full container lifecycle — Docker restart policy is disabled. Containers get:
- Volume mounted at `/data` Question: We could probably display this better, (/ in file manager instead of /data, some games use /data/server, could just be / probably)
- Game scripts bind-mounted at `/scripts` (read-only) Question: Are they readable? They shouldent be in path users have access to? Also why is .Installed in /data? Can it be one dir up so user cant delete it? Or do we want them to be able to? it would lose their data if they did (reinstall) right? Do we even need .Installed?
- Port bindings (host:container, tcp/udp)
- Memory and CPU limits
- Environment variables from game definition + user overrides - Question: Game env vars should provide "defaults" but how do we handle users getting confused cuz these defaults overwrite the file manager changes?

## SteamCMD Retry

Games using SteamCMD install/update through a `steamcmd-retry` wrapper that retries up to 5 times with 10-second delays. Detects failures by checking both exit code AND stdout for "ERROR!" — SteamCMD often returns exit code 0 on failure.

## Connection Address
Question: How reliable is this data? What about if running in docker? Also add a todo to get gamejanitor working in a dockerfile
`netinfo.Detect()` runs once at startup. Detects LAN IP (prefers 192.168.x.x and 172.16-31.x.x over 10.x.x.x to avoid VPN ranges) and external IP (via icanhazip.com). Skips virtual interfaces (tun, tap, wg, docker, veth). In multi-node, each worker reports its own LAN/external IP. Global `connection_address` setting overrides all detection.

## API Surface

All routes return JSON. Auth via `Authorization: Bearer <token>` header.

### Games
- `GET /api/games` — list all
- `GET /api/games/{id}` — get one - Question: Do we need this? Should we have it? Do we use it?

### Gameservers
- `GET /api/gameservers` — list (filters: `?game=`, `?status=`, `?node=`)
- `POST /api/gameservers` — create
- `GET /api/gameservers/{id}` — get
- `PATCH /api/gameservers/{id}` — update (partial)
- `DELETE /api/gameservers/{id}` — delete
- `POST /api/gameservers/{id}/start`
- `POST /api/gameservers/{id}/stop`
- `POST /api/gameservers/{id}/restart`
- `POST /api/gameservers/{id}/update-game`
- `POST /api/gameservers/{id}/reinstall`
- `POST /api/gameservers/{id}/migrate` — move between nodes
- `POST /api/gameservers/bulk` — mass start/stop/restart
- `GET /api/gameservers/{id}/status`
- `GET /api/gameservers/{id}/stats` — CPU, memory, disk usage
- `GET /api/gameservers/{id}/logs`
- `POST /api/gameservers/{id}/command`
Question: Should we have a GET /api/gameservers/id/query that calls gjq for user? Dont we already have that? How does the webui get the query?
### Files
- `GET /api/gameservers/{id}/files?path=` — list directory
- `GET /api/gameservers/{id}/files/content?path=` — read file
- `PUT /api/gameservers/{id}/files/content?path=` — write file
- `DELETE /api/gameservers/{id}/files?path=` — delete
- `POST /api/gameservers/{id}/files/mkdir?path=` — create directory
- `POST /api/gameservers/{id}/files/rename` — rename
- `POST /api/gameservers/{id}/files/upload` — upload
Question: What else do we need for a full featured firle viewer, no download? zipunzip? mv?(dragdrop)

### Backups
- `GET /api/gameservers/{id}/backups`
- `POST /api/gameservers/{id}/backups`
- `POST /api/gameservers/{id}/backups/{backupId}/restore`
- `DELETE /api/gameservers/{id}/backups/{backupId}`

### Schedules
- `GET /api/gameservers/{id}/schedules`
- `POST /api/gameservers/{id}/schedules`
- `GET /api/gameservers/{id}/schedules/{scheduleId}`
- `PATCH /api/gameservers/{id}/schedules/{scheduleId}`
- `DELETE /api/gameservers/{id}/schedules/{scheduleId}`

### Settings (admin only)
- `GET /api/settings`
- `PATCH /api/settings` — partial update

### Tokens (admin only)
- `GET /api/tokens` / `POST /api/tokens` / `DELETE /api/tokens/{id}`
- `GET /api/worker-tokens` / `POST /api/worker-tokens` / `DELETE /api/worker-tokens/{id}`

### Workers (admin only)
- `GET /api/workers` / `GET /api/workers/{id}`
- `PATCH /api/workers/{id}/port-range` / `DELETE /api/workers/{id}/port-range` — DELETE reverts to global default
- `PATCH /api/workers/{id}/limits` / `DELETE /api/workers/{id}/limits` — DELETE reverts to global default

### Schedules (continued)
- `POST /api/gameservers/{id}/schedules/{scheduleId}/toggle` — enable/disable - Question: Should this just be a put on the schedule itself? (i.e, no /toggle just put /id)

### System
- `GET /api/events` — SSE stream of status changes - Question: is this the best way of doing this?
- `GET /api/audit` — audit log (filters: action, resource_type, resource_id) Question: this isnt the token audit is it? But the log stream? Right? Wait no thats below, what is this?
- `GET /api/status` — system overview
- `GET /api/logs?tail=100` — system log lines
- `GET /health` — always 200 OK, no auth

### Page-only (no API equivalent) Question ah, this should be in api no? Or is there some reason it cant -  10MB limit on download but nothing on upload? Numbers to aggresive? (mc java is more then 10mb im pretty sure), configurable?
- `GET /gameservers/{id}/files/download?path=` — file download (10MB limit)
- `POST /gameservers/{id}/files/upload` — file upload (multipart form)

## Token Creation Details

`POST /api/tokens` accepts:
- `name` (string, required)
- `scope` ("admin" or "scoped")
- `gameserver_ids` (string array, for scoped tokens)
- `permissions` (string array: start, stop, restart, console, files, backups, settings, delete) Question: Anything missing here? Console is both logs and sending commands, should probs split it into two? settings is edit? confusing name scheme?
- `expires_in` (duration string e.g. "720h", optional)

Returns the raw token once — it's bcrypt-hashed in the DB and can never be retrieved again.

## To Investigate

Known rough edges, missing pieces, and things worth examining before release:

1. **File download/upload only via page routes, not API.** The web file browser has download and upload, but the REST API has no equivalent. A business building a custom panel would need these as proper API endpoints.

2. **File read size limit is 10MB.** `ReadFile` on the worker caps at 10MB. Large config files or world data can't be read/downloaded through the web interface. SFTP has no such limit.

3. **Rate limit state is in-memory only.** All rate limit counters are lost on restart. A restart resets all limits. Probably fine — rate limiting is about burst protection, not long-term tracking.

4. **No backup download endpoint.** You can create, restore, and delete backups, but you can't download a backup file directly. A business might want to let customers download their own backups. - Question: Need this!

5. **gRPC auth interceptor is unary only.** The `WorkerAuthInterceptor` is registered as a `UnaryInterceptor`. Streaming RPCs (backup/restore volume, container logs) don't go through token validation. Token auth happens on Register, and the controller dials back to the worker — so this may be fine in practice since the controller initiates all streaming calls. - Question: Need to talk more about this! 

6. **Worker disconnect doesn't failover.** When a worker times out, its gameservers are marked unreachable but not migrated. Manual intervention required to migrate gameservers to another node. Question: Talk about this

7. **Connection address detected once at startup.** If the machine's IP changes (DHCP, VPN), the detected address is stale until restart. The global `connection_address` setting is the intended fix for dynamic environments. Question: talk about this

8. **No HTTPS on the web UI.** The HTTP server is plain. Expected to sit behind a reverse proxy (nginx, Caddy) for TLS in production. The `trust_proxy_headers` setting exists for this. Question: talk more on this

9. **CSRF key regenerated if missing.** The CSRF key is stored at `{dataDir}/csrf.key`. If the data dir is ephemeral, all browser sessions are invalidated on restart. Question: Talk more on this

10. **No streaming gRPC auth interceptor for mTLS.** Same as point 5 but for mTLS — the TLS handshake covers all RPCs (unary and streaming), so mTLS compensates for the missing streaming auth interceptor. But without mTLS (if someone somehow bypasses it), streaming RPCs are unprotected. Question: Talk about

11. **Scoped token scope naming inconsistency.** API accepts `scope: "scoped"` but internally stores `scope: "gameserver"`. Works but confusing if someone reads the DB directly. Question: Talk more on this

12. **No webhook/callback on status change.** Events are available via SSE (`GET /api/events`), but there's no outbound webhook. A business would need to maintain an SSE connection to react to status changes rather than receiving push notifications. Question: yes!
