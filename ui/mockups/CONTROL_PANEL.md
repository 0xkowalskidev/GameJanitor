# Control Panel Mockup Spec

The control panel is the detail/management page for a single gameserver. Reached by clicking a gameserver on the dashboard. Multiple routes, one visual context.

## Design Context

- Import `tokens.css` for shared styles (warm dark theme, copper-orange accent)
- Same nav bar as other pages, breadcrumb: "← Gameservers" back to dashboard
- Max width 1000px centered
- The panel should feel like a live control surface, not a settings page
- Orange = operator controls, Green = server health status

## Header (persistent across all tabs)

Always visible at the top of every tab. This is the gameserver's identity.

- Game icon + server name + game name
- Status pill (running/stopped/starting/error) with real-time pulse
- Connection address — prominent, monospace, copy button (same as dashboard hero)
- SFTP info — sftp://username@host:port — also copyable, secondary to game connection
- Action buttons: Start (when stopped), Stop + Restart (when running), Update Game, Reinstall (with confirmation)
- Update Game and Reinstall could be in a "..." overflow menu to keep the header clean

## Tabs / Routes

Each tab is its own route: `/gameservers/{id}`, `/gameservers/{id}/console`, etc.
Tab bar sits below the header.

### Overview (`/gameservers/{id}`)

The default tab. At-a-glance health and info.

- **Resource gauges**: Memory (used/limit), CPU (%), Storage (used/limit if set)
  - Visual bars like the dashboard hero telemetry cells
- **Query data** (if game supports it): Players online / max, player list, current map, server version
- **Live activity feed**: Recent lifecycle events for this gameserver streamed via SSE
  - "Image pulling...", "Container started", "Ready — accepting players"
  - Shows the last ~20 events, newest at top
- **Uptime**: time since last start (or "Stopped" if not running)

### Console (`/gameservers/{id}/console`)

Terminal-style live log viewer with command input.

- Dark background (#08080a), monospace font, auto-scrolling
- Log lines stream in real-time
- Command input at the bottom — full-width, monospace, enter to send
- Should feel like a real terminal, not a text area
- "Clear" button to clear the view (doesn't clear server logs, just the UI buffer)

### Files (`/gameservers/{id}/files`)

File browser for the gameserver's volume.

- Directory tree or breadcrumb path navigation
- File list: name, size, modified date, permissions
- Actions per file: edit (opens in-browser editor for text files), download, rename, delete
- Actions per directory: enter, rename, delete
- Global actions: upload file, create directory
- In-browser text editor for config files (server.properties, etc.) — save button
- The 10MB download limit should be reflected in the UI

### Backups (`/gameservers/{id}/backups`)

Backup management.

- List of backups: name, size, status (completed/in_progress/failed), created date
- In-progress backups show a progress indicator (animated, SSE-driven status update)
- Actions per backup: restore (with confirmation — destructive, no rollback), download, delete
- "Create Backup" button at the top
- Failed backups show error reason

### Schedules (`/gameservers/{id}/schedules`)

Cron-based recurring tasks.

- List of schedules: name, type (restart/backup/command/update), cron expression, human-readable next run, enabled toggle
- Create form: name, type select, cron expression input with human-readable preview ("Every day at 3 AM"), payload (command text if type is "command")
- Edit inline or in a modal
- Delete with confirmation
- Toggle enabled/disabled inline

### Settings (`/gameservers/{id}/settings`)

Server configuration. Only shows fields the current token has permission to edit.

- **General**: Server name (editable), game info (read-only)
- **Environment variables**: Same grouped layout as the create form (Server group, Gameplay group). Edit values, save.
- **Resources**: Memory slider, CPU/storage if require_* settings are enabled
- **Ports**: Current port allocations (read-only for most users), port mode
- **SFTP**: Username (read-only), regenerate password button (shows new password once)
- **Danger zone**: Delete gameserver (with confirmation modal — type server name to confirm), Reinstall (wipe all data)
- **Placement** (admin, multi-node only): Current node, migrate to another node

## Mockup Files

Create one HTML file per major view:
- `control-overview.html` — header + overview tab
- `control-console.html` — header + console tab
- `control-files.html` — header + file browser tab
- `control-backups.html` — header + backups tab

Schedules and Settings can wait — they're forms, similar patterns to create gameserver.

## Mock Data

Use "survival-smp" Minecraft Java server, running, 12/20 players:
- Connection: 192.168.1.10:27015
- SFTP: sftp://survival-smp-a1b2c3@192.168.1.10:2222
- Memory: 2.1 / 4 GB
- CPU: 34%
- Status: running
- Uptime: 3 days, 14 hours
- Players: Steve, Alex, Notch, jeb_, Dinnerbone, C418, Jappa, slicedlime, kingbdogz, Ulraf, Chi, cojomax99
- Map: world
- Version: 1.21.4
