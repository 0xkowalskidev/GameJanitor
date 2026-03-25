# Bugs Found During Testing

Bugs discovered while building the test suite. Each entry has a corresponding skipped test that asserts the correct behavior.

Format:
```
## <Short description>
- **Test:** `TestName` in `path/to/file_test.go`
- **Expected:** what should happen
- **Actual:** what happens instead
- **Severity:** blocks release / should fix / cosmetic
- **Notes:** additional context, links to MEMORY.md entries, etc.
```

---

## PortMode defaults to empty string, skipping auto-allocation
- **Test:** `TestGameserver_Create_PortModeDefaultShouldBeAuto` in `service/gameserver_test.go` (to be written in Phase 3)
- **Expected:** When a gameserver is created without explicitly setting `port_mode`, ports should be auto-allocated from the configured port range. The DB default is `'auto'` but Go struct zero value is `""`.
- **Actual:** `CreateGameserver` checks `gs.PortMode == "auto"` — empty string skips allocation entirely. Ports fall through to `applyGameDefaults` which uses the game's raw default ports (e.g., 27015). Two gameservers of the same game get identical host ports and will conflict on start.
- **Severity:** blocks release
- **Notes:** The API handler (`api/handlers/gameservers.go`) never sets `PortMode`. The DB column default `'auto'` only applies on SQL INSERT, but port allocation runs before the INSERT. Fix is likely: treat empty `PortMode` as `"auto"` in `CreateGameserver`, or have the API handler set it explicitly.

## GameserverIDFromContainerName fails to reject update/fileops containers
- **Test:** `TestNaming_GameserverIDFromContainerName_RejectsUpdateContainer` and `_RejectsFileopsContainer` in `naming/naming_test.go`
- **Expected:** `GameserverIDFromContainerName("gamejanitor-update-abc-123")` should return `("", false)` since it's an update container, not a gameserver.
- **Actual:** Returns `("update-abc-123", true)`. After stripping `gamejanitor-` prefix, the remainder is `update-abc-123`. The check `strings.Contains(id, "-update-")` looks for `-update-` with a leading dash, but the remainder starts with `update-` (no leading dash). Same issue for `-fileops-`, `-reinstall-`, `-backup-`.
- **Severity:** should fix
- **Notes:** `naming/naming.go:34`. The StatusManager uses this to map container events to gameservers. Misidentifying update containers could cause spurious status changes. Fix: use `strings.HasPrefix(id, "update-")` instead of `strings.Contains(id, "-update-")`.

## runBackup panics on nil worker (missing nil check)
- **Test:** Discovered via `TestBackup_Create_ReturnsInProgressRecord` flaky panic in `service/backup_test.go`
- **Expected:** `runBackup` should handle the case where `dispatcher.WorkerFor()` returns nil (e.g., worker disconnected between CreateBackup and the goroutine starting).
- **Actual:** `runBackup` at `service/backup.go:129` calls `w.BackupVolume()` on a nil `w`, causing a panic. The goroutine runs on `context.Background()` and can outlive the request that created it.
- **Severity:** should fix
- **Notes:** `WorkerFor` returns nil when the DB lookup fails or worker isn't registered. The fix is a nil check on `w` before using it, calling `failBackup` if nil.

---

# API Surface Issues

Things that aren't bugs but caused confusion during test development. These signal unclear interfaces that could be improved.

## ValidateToken returns a single value, not (token, error)
- **Location:** `service/auth.go:47`
- **Issue:** `ValidateToken(rawToken string) *models.Token` returns nil for both "invalid token" and "expired token" — the caller can't distinguish between the two, and the lack of an error return is unusual for a Go function that can fail.
- **Suggestion:** Consider `(token, error)` return to distinguish invalid vs expired vs DB failure. Or at minimum, this is worth a doc comment explaining that nil means "not valid for any reason."

## HasPermission and IsAdmin are package-level functions, not AuthService methods
- **Location:** `service/auth.go:255`, `service/auth.go:262`
- **Issue:** `HasPermission(token, gameserverID, permission)` and `IsAdmin(token)` are standalone functions, not methods on `AuthService`. Every other auth operation is on `AuthService`. A caller naturally writes `svc.AuthSvc.HasPermission(...)` and gets a compile error.
- **Suggestion:** Either make them methods on `AuthService` for consistency, or document the split (pure functions vs stateful operations).

## Error messages use env var labels, not keys
- **Location:** `service/gameserver.go` — `validateRequiredEnv`
- **Issue:** When a required env var is missing, the error says `"Required Variable is required"` (using the `Label` field) not `"REQUIRED_VAR is required"` (using the `Key` field). The label is user-friendly for the UI but confusing in API errors and logs where the caller works with keys.
- **Suggestion:** Include both: `"REQUIRED_VAR (Required Variable) is required"` — key for programmatic use, label for human context.
