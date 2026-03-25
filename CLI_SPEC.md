# CLI Specification

The `gamejanitor` binary is both the server and the client. `gamejanitor serve` runs the service; everything else talks to the API. This single-binary model (like consul or nomad) means one download gets you everything.

## Principles

- **Newbie-optimized defaults** — `gamejanitor serve` works out of the box with zero config
- **Gameservers are the core** — lifecycle commands (`start`, `stop`, `logs`, etc.) live at the top level, no `gameservers` prefix needed
- **Name-or-ID everywhere** — commands accept gameserver names, UUID prefixes (4+ chars), or full UUIDs
- **Consistent patterns** — same flags, output style, and ID resolution across all commands
- **`--json` on everything** — human-readable tables by default, JSON for scripting/automation
- **Interactive when helpful** — prompt for missing required fields when stdin is a TTY; never prompt in pipes or scripts
- **Cluster-aware** — client commands route to the current cluster context; defaults to localhost for newbies

## Stack

- **Cobra** — command tree, flag parsing, shell completions, man page generation
- **Lipgloss** — styled help output, colored tables, status indicators
- **Huh** — interactive forms for guided creation flows (gameserver create, cluster add, etc.)

Custom Cobra help templates replace the default sterile output with styled, persona-aware help text.

## Help Output

The root `gamejanitor` command (no args) shows styled help. Commands first (what returning users need), getting started last (where newbies' eyes land):

```
Gamejanitor - Game Server Manager

Gameserver Commands:
  create        Create a new gameserver
  delete        Delete a gameserver
  start         Start a gameserver
  stop          Stop a gameserver
  restart       Restart a gameserver
  status        Show gameserver or cluster status
  logs          Show gameserver or service logs
  command       Send a console command to a gameserver
  update-game   Update a gameserver's game version
  reinstall     Reinstall a gameserver
  migrate       Migrate a gameserver to another node

Resource Management:
  gameservers   Manage gameservers                    (aliases: gs)
  backups       Manage backups
  schedules     Manage scheduled tasks
  games         List available games

Administration:
  tokens        Manage auth tokens
  workers       Manage worker nodes                   (aliases: w)
  settings      View and configure settings
  webhooks      Manage webhook endpoints
  events        Query or stream events

Server:
  serve         Start the gamejanitor server
  install       Install as a system service
  update        Self-update to latest release
  init          Generate a starter config file
  cluster       Manage cluster connections             (aliases: ctx)
  completion    Generate shell completions

Get started:
  gamejanitor serve                     Start the server
  http://localhost:8080                 Open the web UI

Flags:
  --json        Output as JSON
  --yes, -y     Skip confirmation prompts
  --help, -h    Show help

Run 'gamejanitor <command> --help' for details.
```

Status colors: green=running, yellow=installing/starting, red=error, gray=stopped. Respects `NO_COLOR` env var.

---

## Server Commands

### serve

```
gamejanitor serve [flags]
```

Starts the gamejanitor server. Both controller and worker are enabled by default — a single command runs everything for newbies.

| Flag | Default | Description |
|---|---|---|
| `--config` | — | Path to YAML config file (see CONFIG_SPEC.md) |
| `--controller` | `true` | Enable the API server and orchestrator |
| `--worker` | `true` | Enable the local worker that manages containers on this machine |
| `-p, --port` | `8080` | Port for the HTTP API and web UI |
| `--bind` | `127.0.0.1` | Address to listen on (`0.0.0.0` for all interfaces) |
| `-d, --data-dir` | `/var/lib/gamejanitor` | Database, backups, and config directory |
| `--sftp-port` | `2222` | Port for the built-in SFTP file server (0 to disable) |
| `--grpc-port` | `9090` | Port for worker nodes to connect (0 to disable) |
| `--web-ui` | `true` | Serve the web dashboard (ignored when `--controller=false`) |
| `--controller-address` | — | Controller gRPC address to register with (e.g. `10.0.0.1:9090`) |
| `--worker-id` | hostname | Unique name for this worker node |
| `--worker-token` | — | Auth token for worker registration (or `GJ_WORKER_TOKEN` env) |
| `--runtime` | `auto` | Container runtime: `docker`, `podman`, `process`, `auto` |

All flags can also be set in the config file. CLI flags override config file values. Config file is optional.

**Startup validation:**
- Auth enabled but no tokens in DB → hard error: `auth is enabled but no tokens exist — create one first: gamejanitor token create --name admin --type admin`
- S3 configured but unreachable → hard error with connection details
- Docker/Podman not accessible → hard error suggesting `usermod -aG docker` or `sudo`

**Deployment modes:**
| Mode | Command | Who |
|---|---|---|
| Newbie | `gamejanitor serve` | Single machine, zero config |
| Business controller | `gamejanitor serve --config config.yaml --worker=false` | API + orchestrator only |
| Business worker | `gamejanitor serve --config config.yaml --controller=false` | Container host only |

### install

```
gamejanitor install [--systemd|--launchd]
```

Installs gamejanitor as a system service. Generates a service file, enables it, and starts it. Detects init system automatically if no flag specified.

This solves the "I closed my terminal and it stopped" problem for newbies. After install, gamejanitor starts on boot and survives terminal closure.

### update

```
gamejanitor update
```

Self-updates the gamejanitor binary to the latest GitHub release. Games are embedded in the binary, so updating also gets new game definitions.

NixOS users update via their flake instead. The command detects NixOS and tells them so rather than breaking their system.

### init

```
gamejanitor init [--profile newbie|business]
```

Generates a `gamejanitor.yaml` in the current directory.

| Profile | What it generates |
|---|---|
| `newbie` (default) | Minimal config with comments explaining each option |
| `business` | Auth enabled, resource limits on, S3 placeholders, multi-node setup comments |

---

## Cluster Context

For operators managing multiple gamejanitor deployments. Modeled after `kubectl config`.

```
gamejanitor cluster add <name> --address <url> --token <token>
gamejanitor cluster use <name>
gamejanitor cluster list
gamejanitor cluster remove <name>
gamejanitor cluster current
```

Stored in `~/.config/gamejanitor/clusters.yaml`. Token stored per-cluster. Current cluster remembered between commands. All client commands use the current cluster's address and token.

If no cluster configured, defaults to `http://localhost:8080` with no token (newbie mode).

**Override per-command:**
```
gamejanitor status --cluster production
gamejanitor --api-url http://10.0.0.5:8080 --token gj_abc123 status
```

`--cluster` selects a named cluster for one command. `--api-url` and `--token` override everything.

---

## Global Flags

Available on all client commands:

| Flag | Description |
|---|---|
| `--json` | Output as JSON (same envelope as API: `{"status":"ok","data":...}`) |
| `--yes, -y` | Skip confirmation prompts |
| `--api-url` | Override API base URL |
| `--token` | Override auth token |
| `--cluster` | Use a specific named cluster for this command |

---

## Gameserver Commands (Top-Level)

Gameservers are the core resource. The most common operations live at the root — no `gameservers` prefix needed. These are all shortcuts for `gameservers <command>`.

### create

```
gamejanitor create [flags]
```

Creates a new gameserver. Same flags as `gameservers create` (see Resource Commands below).

When required flags (`--name`, `--game`) are missing and stdin is a TTY, launches an interactive form. When piped/scripted, missing required flags are errors.

### delete

```
gamejanitor delete <name-or-id> [--force]
```

Deletes a gameserver. Requires confirmation unless `--yes` or `--force`. Stops the gameserver first if running.

### start / stop / restart

```
gamejanitor start <name-or-id>
gamejanitor stop <name-or-id>
gamejanitor restart <name-or-id>
```

Bulk operations:
```
gamejanitor start --all
gamejanitor stop --node node-2
gamejanitor restart --all
```

| Flag | Description |
|---|---|
| `--all` | Apply to all gameservers |
| `--node` | Apply to all gameservers on a specific node |

### status

```
gamejanitor status                      # cluster overview: all gameservers with status
gamejanitor status <name-or-id>         # detailed status for one gameserver
```

Without argument: table of all gameservers with name, game, status, memory/CPU usage, node.
With argument: detailed view including container info, query data (players, map), resource stats.

### logs

```
gamejanitor logs <name-or-id> [--tail N] [--follow]
gamejanitor logs --service
```

| Flag | Default | Description |
|---|---|---|
| `--tail` | `100` | Number of lines to show |
| `--follow, -f` | `false` | Stream live logs |
| `--service` | `false` | Show gamejanitor's own service logs instead of gameserver logs |

`--service` pulls from journalctl when running as a systemd unit. If gamejanitor isn't running as a service, prints a message explaining that logs went to stdout.

### command

```
gamejanitor command <name-or-id> <command...>
```

Sends a console command to a running gameserver. Prints the response.

### update-game

```
gamejanitor update-game <name-or-id>
```

Updates the gameserver's game to the latest version.

### reinstall

```
gamejanitor reinstall <name-or-id>
```

Reinstalls the gameserver (preserves user data, re-runs install). Requires confirmation.

### migrate

```
gamejanitor migrate <name-or-id> --node <target-node>
```

Migrates a gameserver to a different worker node. Requires confirmation.

---

## Resource Commands

### gameservers (aliases: gs)

All gameserver commands. Top-level shortcuts (`create`, `delete`, `start`, `stop`, `restart`, `status`, `logs`, `command`, `update-game`, `reinstall`, `migrate`) are aliases for `gameservers <command>` — both forms work.

#### list

```
gamejanitor gameservers list [--game <game-id>] [--status <status>] [--json]
```

#### get

```
gamejanitor gameservers get <name-or-id> [--json]
```

#### create

```
gamejanitor gameservers create [flags]
gamejanitor create [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--name` | required | Gameserver name |
| `--game` | required | Game ID |
| `--memory` | game default | Memory limit (e.g. `4g`, `2048`, `512m`) |
| `--cpu` | 0 (unlimited) | CPU limit in cores |
| `--port` | — | Port mapping: `name:host:container/proto` (repeatable) |
| `--env` | — | Environment variable: `KEY=VALUE` (repeatable) |
| `--node` | auto-placed | Worker node ID |
| `--auto-restart` | `false` | Auto-restart on crash |

**Interactive mode:** When `--name` or `--game` are missing and stdin is a TTY, launches an interactive form:
1. Pick a game from a list (with search)
2. Enter a name
3. Optionally configure memory, env vars, etc.
4. Confirm and create

When stdin is not a TTY (piped/scripted), missing required flags are errors. No prompts.

#### update

```
gamejanitor gameservers update <name-or-id> [flags]
```

Same flags as create, all optional. Only provided flags are changed. Gameserver must be stopped.

#### delete

```
gamejanitor gameservers delete <name-or-id> [--force]
gamejanitor delete <name-or-id> [--force]
```

Requires confirmation unless `--yes` or `--force`. Stops the gameserver first if running.

### backups

```
gamejanitor backups list <gameserver> [--json]
gamejanitor backups create <gameserver> [--name <name>]
gamejanitor backups restore <gameserver> <backup-name-or-id>
gamejanitor backups download <gameserver> <backup-name-or-id> [--output <file>]
gamejanitor backups delete <gameserver> <backup-name-or-id>
```

Create and restore return immediately (async). Both print the backup ID and tell the user to watch events for completion.

### schedules

```
gamejanitor schedules list <gameserver> [--json]
gamejanitor schedules create <gameserver> --type <type> --cron <expr> [--name <name>] [--payload <json>]
gamejanitor schedules update <gameserver> <schedule-name-or-id> [--cron <expr>] [--enabled true|false] [--name <name>]
gamejanitor schedules delete <gameserver> <schedule-name-or-id>
```

Types: `restart`, `backup`, `command`, `update`.

### games

```
gamejanitor games list [--json]
gamejanitor games get <id> [--json]
```

List returns basic info: id, name, image, recommended memory.
Get returns full definition: default ports, env vars with descriptions/options/types, ready pattern, capabilities.

Read-only for now. Custom game definitions are a future feature.

---

## Admin Commands

### tokens

API-based token management. Requires a running server and auth.

```
gamejanitor tokens list [--json]
gamejanitor tokens create --name <name> [--scope admin|custom] [--permissions <perm,...>] [--gameservers <id,...>] [--expires <duration>]
gamejanitor tokens delete <name-or-id>
```

| Scope | Behavior |
|---|---|
| `admin` | Full access to everything |
| `custom` (default) | Scoped to specific permissions and gameservers |

Permissions: `start`, `stop`, `restart`, `logs`, `commands`, `files`, `backups`, `configure`.

Shows the raw token once on creation. Warns user to save it — it cannot be retrieved later.

### token (offline bootstrap)

Direct DB access, no running server needed. Used to create the first admin token before enabling auth.

```
gamejanitor token create --name <name> --type admin|worker [-d /var/lib/gamejanitor]
gamejanitor token rotate --name <name> --type admin|worker [-d /var/lib/gamejanitor]
```

**Idempotent:** if a token with the given name already exists, exits 0 silently. Safe for `ExecStartPre` in systemd units and NixOS declarative deployments. The raw token is only printed on first creation.

### workers (aliases: w)

```
gamejanitor workers list [--json]
gamejanitor workers get <id> [--json]
gamejanitor workers set <id> [--memory <MB>] [--cpu <cores>] [--storage <MB>] [--port-range <start>-<end>] [--tags <key=value,...>]
gamejanitor workers clear <id> [--limits] [--port-range] [--tags]
gamejanitor workers cordon <id>
gamejanitor workers uncordon <id>
```

`set` is a unified command for all worker configuration. `clear` removes specific settings. `cordon`/`uncordon` prevent or allow new gameserver placement on a node.

### settings

```
gamejanitor settings [--json]
gamejanitor settings set <key> <value>
```

Without subcommand: display all settings with current values.

Keys: `connection-address`, `port-range-start`, `port-range-end`, `port-mode`, `max-backups`, `auth-enabled`, `localhost-bypass`, `webhook-enabled`, `webhook-url`, `webhook-secret`.

### webhooks

```
gamejanitor webhooks list [--json]
gamejanitor webhooks create --url <url> [--events <glob,...>] [--secret <secret>] [--description <desc>]
gamejanitor webhooks update <id> [--url <url>] [--events <glob,...>] [--secret <secret>] [--enabled true|false]
gamejanitor webhooks delete <id>
gamejanitor webhooks test <id>
gamejanitor webhooks deliveries <id> [--state pending|delivered|failed] [--limit N]
```

Default events: `*` (all). Secret is user-provided for HMAC-SHA256 signing.

### events

```
gamejanitor events [--type <glob>] [--gameserver <name-or-id>] [--limit N] [--json]
gamejanitor events --follow [--type <glob>]
```

Without `--follow`: queries event history from the database.
With `--follow`: streams live events via SSE.

Events are structured records of what happened in the cluster: `gameserver.started`, `backup.completed`, `worker.connected`, etc. This is distinct from `logs` which shows raw process stdout.

### gen-worker-cert

```
gamejanitor gen-worker-cert <worker-id> [-d /var/lib/gamejanitor]
```

Generates TLS certificate for a worker node. Outputs paths to cert, key, and CA files.

---

## Shell Completion

```
gamejanitor completion bash
gamejanitor completion zsh
gamejanitor completion fish
```

Outputs shell completion script. Installation instructions shown in `--help`. Cobra provides this with dynamic completions for gameserver names, game IDs, worker IDs, etc.

---

## Interactive Mode

Commands with required fields launch interactive forms when:
1. Required flags are missing, AND
2. Stdin is a TTY (not piped or scripted)

When stdin is not a TTY, missing required flags produce an error with usage hints.

Interactive mode uses `charmbracelet/huh` for styled form prompts with search, selection lists, and validation. This gives newbies a guided experience while power users who pass all flags never see a prompt.

Commands with interactive support:
- `gameservers create` — game picker, name input, optional config
- `cluster add` — address, name, token input
- `init` — profile selection

---

## Output

**Table format** (default): human-readable aligned columns with Lipgloss styling.
```
$ gamejanitor status
NAME          GAME              STATUS    MEMORY   CPU    NODE
my-mc-server  minecraft-java    running   4 GB     2.0    node-1
test-ark      ark-survival      stopped   8 GB     4.0    node-2
```

**JSON format** (`--json`): machine-readable, same envelope as the API.
```json
{"status":"ok","data":[...]}
```

JSON output is a stable contract for automation. Field names match the API response schema.

---

## Error Messages

Errors are clear, actionable, and suggest next steps.

```
$ gamejanitor start nonexistent
Error: gameserver "nonexistent" not found

$ gamejanitor gameservers create --name test --game fake
Error: game "fake" not found
  Run 'gamejanitor games list' to see available games.

$ gamejanitor start my-server
Error: cannot connect to gamejanitor at http://localhost:8080
  Is the server running? Start it with: gamejanitor serve
  Or set up a remote cluster: gamejanitor cluster add <name> --address <url> --token <token>
```

Connection errors always suggest both local (`serve`) and remote (`cluster add`) paths.

---

## ID Resolution

All commands that accept a resource identifier support three forms:
1. **Exact name** (case-insensitive): `my-mc-server`
2. **UUID prefix** (4+ characters): `a1b2c3d4`
3. **Full UUID**: `a1b2c3d4-e5f6-7890-abcd-ef1234567890`

Resolution is cached per CLI invocation to avoid redundant API calls when multiple operations reference the same resource.

---

## Persona Journeys

### Newbie

```bash
# Download and run
curl -fsSL https://gamejanitor.dev/install | sh
gamejanitor serve

# Open browser to http://localhost:8080
# Everything managed through web UI from here

# Later: "it stops when I close my terminal"
gamejanitor install
# Now it runs on boot
```

The CLI is their entry point but not their daily interface. The help text, error messages, and `install` command are designed to get them to the web UI as fast as possible.

### Power User

```bash
gamejanitor serve -d ~/gamejanitor

# Manage everything from CLI
gamejanitor create --name mc --game minecraft-java --memory 4g
gamejanitor start mc
gamejanitor logs mc -f
gamejanitor command mc "say hello"
gamejanitor backups create mc
gamejanitor schedules create mc --type backup --cron "0 3 * * *" --name nightly

# Scripting
gamejanitor status --json | jq '.data[] | select(.status == "running")'
```

### Business Operator

```bash
# Bootstrap controller (IaC deploys config + systemd unit)
gamejanitor token create --name admin --type admin -d /var/lib/gamejanitor
# → gj_abc123...

# On workstation: add cluster contexts
gamejanitor cluster add production --address https://gj.example.com --token gj_abc123
gamejanitor cluster add staging --address https://gj-staging.example.com --token gj_xyz789
gamejanitor cluster use production

# Manage cluster
gamejanitor status
gamejanitor workers list
gamejanitor workers cordon node-3
gamejanitor migrate mc-server --node node-1

# Switch context
gamejanitor cluster use staging
gamejanitor status
```
