# Web UI Specification

## Overview

Single-page application embedded in the gamejanitor binary. Consumes the API like any other client — no shared code with the service layer. Built with Svelte, compiled to static assets, served from `ui/dist/` via `embed.go`.

The UI adapts to the user's permissions. An admin sees everything. A scoped token sees only the gameservers and actions they have access to. This naturally supports businesses exposing the UI to customers via reverse proxy — no separate "customer panel" needed, just token scoping.

## Architecture

```
ui/
├── src/              # Svelte source
├── dist/             # built static assets (committed or built at release)
├── embed.go          # embeds dist/ and serves via Go's embed.FS
├── package.json
└── vite.config.js
```

The Go binary serves `ui/dist/` at `/` (or a configurable path). The SPA handles all routing client-side. API calls go to `/api/*` on the same origin — no CORS needed.

When `--web-ui=false` (or `web_ui: false` in config), the embed route is simply not registered. API-only mode.

## Tech Stack

- **Svelte** — compiler-based, small bundles, reactive by default (ideal for SSE streams)
- **Vite** — build tool, dev server with HMR
- **No CSS framework** — utility classes or minimal custom CSS. The UI should feel fast and light, not bloated with a framework.

## Real-Time

One SSE connection to `/api/events` per session. Events are dispatched to Svelte stores, which components subscribe to reactively.

| Data | Source | Method |
|---|---|---|
| Gameserver status changes | `status_changed` events via SSE | Real-time |
| Lifecycle progress (pulling, creating, starting) | Lifecycle events via SSE | Real-time |
| Console/log output | `/api/gameservers/{id}/logs?follow=true` | Streaming |
| Resource stats (CPU, memory, storage) | `/api/gameservers/{id}/stats` | Polling (5s) |
| Query data (players, map, version) | `/api/gameservers/{id}/query` | Polling (10s) or SSE if available |
| Worker status | `worker.connected`/`worker.disconnected` via SSE | Real-time |

## Pages

### Dashboard

The landing page. Shows all gameservers the token has access to.

- Grid of gameserver cards (game icon, name, status badge, player count)
- Status badges update in real-time via SSE
- "Create Gameserver" button (admin only)
- Filter by game, status
- If no gameservers exist: prominent CTA — "Create your first gameserver"

### Gameserver Detail

The main management page for a single gameserver. Reached by clicking a card on the dashboard.

**Header:**
- Game icon, name, status badge (real-time)
- Connection address + port (copy button)
- Quick actions: Start / Stop / Restart (permission-gated)

**Live Activity Feed:**
- Shows lifecycle events as they happen: "Pulling image...", "Container started", "Ready — accepting players"
- Driven by SSE events filtered to this gameserver
- Replaces the old approach of only showing logs in a separate console tab

**Tabs:**

| Tab | Content | Permission |
|---|---|---|
| Overview | Resource stats (CPU, memory, storage gauges), query data (players online, map, version), uptime | Access |
| Console | Live log stream + command input | `gameserver.logs` / `gameserver.command` |
| Files | File browser with edit, upload, download, rename, delete, mkdir | `gameserver.files.*` |
| Backups | Backup list with create, restore, download, delete. Shows in-progress backups with live status | `backup.*` |
| Schedules | Schedule list with create, edit, delete, toggle. Cron expression input with human-readable preview | `schedule.*` |
| Settings | Name, environment variables, resource limits, port config, node placement. Only fields the token can edit | `gameserver.edit.*` |

### Games

Grid of available games with icons, names, recommended memory. Clicking a game shows its full definition: default ports, configurable env vars with descriptions, capabilities.

Used during gameserver creation — select a game, then configure.

### Create Gameserver

Form page (or modal from dashboard):
1. Select game (grid with icons)
2. Name
3. Memory limit (prefilled with game's recommended)
4. Optional: CPU limit, storage limit, port mode, node tags, env overrides
5. Create → redirect to gameserver detail, watch install in real-time

Advanced fields collapsed by default. Newbie sees: game, name, memory. Power user expands for more.

### Settings (Admin)

Only visible to admin tokens (or when auth is disabled).

- **Connection address** — auto-detected vs manual override
- **Port range** — start/end
- **Port mode** — auto/manual default
- **Auth** — enable/disable, localhost bypass
- **Rate limiting** — enable/disable, per-IP/per-token/login thresholds
- **Backups** — global max backups
- **Resource requirements** — require memory/CPU/storage limits on create
- **Event retention** — days to keep event history
- **Proxy headers** — trust X-Forwarded-For

All settings editable in-place. No "save" button — changes apply immediately via `PATCH /api/settings`.

### Workers (Admin)

Only visible when multi-node is active (workers connected).

- List of worker nodes with status (connected/slow/stale), resource allocation bars, gameserver counts
- Per-worker actions: set limits, set port range, set tags, cordon/uncordon
- Real-time status updates via SSE

### Tokens (Admin)

- List of API tokens with name, scope, last used, created at, expiry
- Create token form: name, scope (admin/custom/worker), permissions checkboxes, gameserver access list
- Show-once token display on creation with copy button
- Delete confirmation

### Webhooks (Admin)

- List of webhook endpoints with URL, enabled status, event filter
- Create/edit form: URL, secret, event filter (glob patterns), enabled toggle
- Test button — sends test payload, shows response status
- Delivery history per endpoint with state, attempts, errors

## Permission-Aware Rendering

The UI fetches the current token's permissions on load (or infers from 403 responses). Components conditionally render based on permissions:

- **No token in context** (auth disabled or localhost bypass): show everything, assume admin
- **Admin token**: show everything
- **Custom token**: hide sections the token can't access

Examples:
- No `gameserver.start` permission → Start button hidden
- No `settings.view` permission → Settings nav item hidden
- No `tokens.manage` permission → Tokens nav item hidden
- `gameserver_ids` scoped → dashboard only shows those gameservers

This is purely cosmetic — the API enforces permissions regardless. The UI just avoids showing actions that will fail.

## Auth UX

**Auth disabled (newbie default):** UI just works. No login, no token needed.

**Auth enabled:**
- If no token in cookie/storage → show token entry page: "Paste your API token to continue"
- Token stored in `_token` cookie (same as current auth middleware expects)
- On first auth enable via UI: auto-generate admin token, display it prominently, store in cookie automatically so user doesn't immediately get locked out
- "Emergency recovery" documented: `gamejanitor token create --type admin` always works offline via direct DB access

## Onboarding

No wizard. No setup flow. Dashboard is the first thing users see.

- First visit with no gameservers: empty dashboard with "Create your first gameserver" CTA
- First visit with gameservers (e.g. created via CLI before opening UI): populated dashboard

The firewall/port forwarding problem (users can't connect to their gameserver from outside their LAN) is not solvable in onboarding. Document it in the help/docs, possibly show a hint on the gameserver detail page if connection_address is not configured.

## Embedding

The built UI assets are embedded in the Go binary:

```go
package ui

import "embed"

//go:embed dist/*
var Assets embed.FS
```

The router serves these at `/` with SPA fallback (all non-API routes serve `index.html`). The API lives at `/api/*` and is unaffected.

Build process:
- Development: `cd ui && npm run dev` (Vite dev server proxying API to the Go backend)
- Production: `cd ui && npm run build` → outputs to `ui/dist/` → Go embeds it

## What the UI is NOT

- **Not a customer portal** — it's an admin/management interface. Businesses who want customer-facing panels build their own on top of the API.
- **Not a replacement for the CLI** — power users and businesses may prefer CLI/API for automation. The UI is for visual management.
- **Not a monitoring dashboard** — it shows current state, not historical trends. No graphs over time, no alerting. Use Grafana/Prometheus for that (gamejanitor could expose a `/metrics` endpoint in the future).

However, because the UI respects token permissions, businesses *can* expose it to customers if they want. A customer with a scoped token only sees their gameservers and allowed actions. This isn't a designed feature — it's a natural consequence of permission-aware rendering. No extra work required.
