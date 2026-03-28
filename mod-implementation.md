# Mod System Redesign — Implementation Plan

Reference: `mod-spec.md` in project root.

## Scope

Delete the current mod system entirely. Build new one per spec. Touch list:

| Area | Action |
|------|--------|
| `games/store.go` | Replace `ModConfig`, `ModSourceConfig`, `ModLoaderConfig` with new types. Add `RuntimeConfig`. |
| `games/registry.go` | Add `RuntimeConfig` to `GameDef.Container`. |
| `games/resolve.go` | **New file.** Image resolution logic + Mojang API resolver. |
| `controller/mod/` | **Delete all 5 files.** Rewrite from scratch: `service.go`, `catalog.go` (interface), `delivery.go`, `modrinth.go`, `umod.go`, `workshop.go`, `deps.go`, `compat.go`. |
| `model/mod.go` | Update `InstalledMod` struct with new fields. |
| `db/migrations/001_initial.sql` | Update `installed_mods` table. Add `pack_exclusions` table. |
| `store/mod.go` | Update store methods for new schema. |
| `api/handler/mods.go` | Rewrite handlers for new API shape (categories, updates, packs). |
| `api/router.go` | Update mod routes. |
| `cli/services.go` | Update `ModService` wiring. |
| `games/data/*/game.yaml` | Update 5 game YAMLs to new `mods:` format + add `runtime:` section to minecraft-java. |
| `controller/gameserver/lifecycle.go` | Call `game.ResolveImage(env)` instead of static `game.BaseImage`. |

UI changes are out of scope for this implementation — API + backend only.

## Phase 1: Foundation (types, schema, games package)

**Goal:** New types compiled and parseable. DB schema ready. No service code yet.

1. Update `games/store.go`:
   - Delete `ModSourceConfig`, `ModLoaderConfig`, `ModConfig`
   - Add new types per spec:
     ```go
     type RuntimeConfig struct {
         Image        string            `yaml:"image,omitempty"`
         Resolver     string            `yaml:"resolver,omitempty"`
         DefaultImage string            `yaml:"default_image,omitempty"`
         Images       map[string]string `yaml:"images,omitempty"`
     }

     type ModsConfig struct {
         VersionEnv string              `yaml:"version_env,omitempty"`
         Loader     *ModLoaderDef       `yaml:"loader,omitempty"`
         Categories []ModCategoryDef    `yaml:"categories,omitempty"`
     }

     type ModLoaderDef struct {
         Env     string                       `yaml:"env"`
         Options map[string]ModLoaderOption    `yaml:"options"`
     }

     type ModLoaderOption struct {
         ModSources []string `yaml:"mod_sources"`
         LoaderID   string   `yaml:"loader_id,omitempty"`
     }

     type ModCategoryDef struct {
         Name    string              `yaml:"name"`
         Sources []ModCategorySource `yaml:"sources"`
     }

     type ModCategorySource struct {
         Name          string            `yaml:"name"`
         Delivery      string            `yaml:"delivery"` // "file", "manifest", "pack"
         InstallPath   string            `yaml:"install_path,omitempty"`
         OverridesPath string            `yaml:"overrides_path,omitempty"`
         Filters       map[string]string `yaml:"filters,omitempty"`
         Config        map[string]string `yaml:"config,omitempty"`
     }
     ```
   - Add `RuntimeConfig` to `ContainerConfig`
   - Replace `ModConfig` with `ModsConfig` on `ContainerConfig`
   - Update `Game` struct: add `Runtime RuntimeConfig`, replace `Mods ModConfig` with `Mods ModsConfig`

2. Add `games/resolve.go`:
   - `func (g *Game) ResolveImage(env map[string]string) string` — checks resolver, falls back to static image
   - `func (g *Game) AvailableModSources(env map[string]string) []ModCategoryDef` — filters categories by loader availability
   - Minecraft-java resolver: fetch Mojang manifest, extract `javaVersion.majorVersion`, map to image tag
   - Resolver caching (the Mojang manifest doesn't change every request)

3. Update `games/registry.go`:
   - `ContainerConfig` gets `Runtime *RuntimeConfig` and `Mods *ModsConfig`

4. Update `model/mod.go`:
   - Add `Category`, `Delivery`, `AutoInstalled`, `DependsOn`, `PackID` fields to `InstalledMod`

5. Update `db/migrations/001_initial.sql`:
   - Add new columns to `installed_mods`
   - Add `pack_exclusions` table
   - Add `idx_installed_mods_pack` index

6. Update `store/mod.go`:
   - Add new query methods: `ListModsByPackID`, `GetPackExclusions`, `SetPackID`, `CreatePackExclusion`, `UpdateModVersion`
   - Update existing methods for new columns

7. Update 5 game YAMLs to new format:
   - `minecraft-java/game.yaml` — full categories + runtime resolver
   - `rust/game.yaml` — umod with framework toggle
   - `ark-survival-evolved/game.yaml` — workshop manifest
   - `counter-strike-2/game.yaml` — workshop manifest
   - `garrys-mod/game.yaml` — workshop manifest

**Verify:** `go build ./...` passes. Game YAMLs parse correctly. Schema is valid.

## Phase 2: Catalogs (mod source API clients)

**Goal:** New `ModCatalog` interface. Modrinth, uMod, Workshop reimplemented.

1. Write `controller/mod/catalog.go`:
   - `ModCatalog` interface (Search, GetDetails, GetVersions, GetDependencies)
   - `CatalogFilters` struct
   - `ModResult`, `ModDetails`, `ModVersion`, `ModDependency` types

2. Rewrite `controller/mod/modrinth.go`:
   - Implement `ModCatalog` for Modrinth
   - Port existing API client code (response types, HTTP helpers)
   - Add `GetDependencies` (parse `dependencies[]` from version response)
   - Add `project_type` filter support (for mods vs resource packs vs modpacks)

3. Rewrite `controller/mod/umod.go`:
   - Implement `ModCatalog` for uMod
   - Port existing API client code
   - `GetDependencies` returns nil (uMod has no deps)
   - Use `config.category` from game YAML instead of hardcoded "rust"

4. Rewrite `controller/mod/workshop.go`:
   - Implement `ModCatalog` for Workshop
   - Port existing API client code
   - Use `config.app_id` from game YAML
   - `GetDependencies` returns nil

**Verify:** Each catalog can be instantiated and compiles. No service wiring yet.

## Phase 3: Delivery mechanisms

**Goal:** Three delivery types working.

1. Write `controller/mod/delivery.go`:
   - `FileDelivery` — streaming download to volume (replace the old `downloadFile` that buffered everything)
   - `ManifestDelivery` — write/rewrite JSON manifest
   - `PackDelivery` — download .mrpack, parse index, filter server-side, download mods with hash verification, extract overrides with path validation

2. Update `FileOperator` interface if needed for streaming (currently `WriteFile` takes `[]byte` — may need `WriteFileStream` that takes `io.Reader`)

**Verify:** Delivery types compile. PackDelivery path validation rejects traversal attempts.

## Phase 4: Mod Service (core logic)

**Goal:** New `ModService` with all methods. Wired into the app.

1. Write `controller/mod/service.go`:
   - Constructor: takes catalogs, delivery types, store, game store, broadcaster, logger
   - `Install` — resolve version, check duplicates, install deps, deliver, record
   - `Uninstall` — deliver uninstall, delete record, clean orphaned deps
   - `ListInstalled`
   - `GetSources` — return available categories/sources for a gameserver's current config
   - `Search` — route to correct catalog with resolved filters
   - `GetVersions` — route to correct catalog

2. Write `controller/mod/deps.go`:
   - `installDependencies` — recursive with depth limit (10, warn at 5)
   - `removeOrphanedDependencies`

3. Write `controller/mod/compat.go`:
   - `CheckCompatibility` — called before version/loader changes
   - `CheckForUpdates` — compare installed vs latest
   - Handle three cases: version change, loader change, framework deactivation

4. Write `controller/mod/pack.go`:
   - `InstallPack` — download pack, install all mods, handle duplicates, record with pack_id
   - `UpdatePack` — diff old vs new, respect exclusions
   - `uninstallPackMods` — cascade delete

5. Update `controller/mod/service.go`:
   - `Update` — single mod
   - `UpdateAll` — all mods

6. Delete old files: `mod.go`, `source.go` (replaced by new files)

7. Update `cli/services.go`:
   - Wire new `ModService` with catalogs and delivery types

8. Update `controller/gameserver/lifecycle.go`:
   - Replace `game.BaseImage` with `game.ResolveImage(gs.Env)` in Start

**Verify:** `go build ./...` passes. `go test ./...` passes. Create a gameserver, the image resolution works.

## Phase 5: API handlers + routes

**Goal:** New API endpoints matching the spec.

1. Rewrite `api/handler/mods.go`:
   - `List` — list installed mods (with category filter)
   - `Sources` — return available categories/sources
   - `Search` — search with category + source params
   - `Versions` — get versions for a mod
   - `Install` — install mod (with category, handles deps)
   - `InstallPack` — install modpack
   - `Uninstall` — uninstall (handles pack cascade + exclusions)
   - `CheckUpdates` — return available updates
   - `Update` — update single mod
   - `UpdateAll` — update all mods
   - `UpdatePack` — update modpack
   - `CheckCompatibility` — check mods against proposed env changes

2. Update `api/router.go`:
   - New routes under `/api/gameservers/{id}/mods/`

**Verify:** `go build ./...` passes. Full test suite passes. Can hit the API endpoints manually with curl.

## Phase 6: Tests

**Goal:** Meaningful test coverage for the new system.

1. Test the game YAML parsing — new ModsConfig, RuntimeConfig
2. Test image resolution — Minecraft resolver with mocked Mojang API
3. Test `AvailableModSources` — loader toggling, env filtering
4. Test compatibility checker — version change, loader change, framework off
5. Test pack duplicate handling — same mod same version, same mod different version
6. Test pack exclusions — remove mod from pack, update pack, excluded mod not re-added
7. Test dependency depth limit
8. Test path validation on pack overrides
9. Integration test: install + uninstall through the full service (with fake catalogs)

**Verify:** `go test ./...` all green.
