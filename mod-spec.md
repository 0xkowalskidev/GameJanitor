# Mod System Redesign Spec

## Overview

The mod system manages three entangled concerns: **game version**, **mod loader**, and **mods**. These are currently treated as separate systems (version/loader as env vars, mods as a standalone service) but they're one coherent configuration from the user's perspective. This redesign unifies them.

## Principles

1. **The user picks version, loader, and mods in one place.** The mods tab owns all three.
2. **The system figures out the rest.** Image variant, loader version, install paths, compatibility — all resolved automatically.
3. **No framework is sacred.** The system handles Modrinth, uMod, Workshop, Thunderstore, CurseForge, resource packs, data packs, modpacks, and anything future through the same abstractions.
4. **Simple games carry zero weight.** No `mods:` section in game YAML = no mod UI, no mod code runs.
5. **Different delivery mechanisms are modeled as different things.** File downloads, manifest IDs, and modpack bundles are three distinct delivery types.

## User Experience

### Web UI — Mods Tab

The mods tab appears only if the game YAML has a `mods` section.

```
┌──────────────────────────────────────────────────┐
│  Version: [1.21.1 ▾]       Loader: [Fabric ▾]   │
│                                                   │
│  ⚠ 2 mods are incompatible with 1.22             │  ← contextual warnings
├───────────────────────────────────────────────────┤
│  [Mods]  [Resource Packs]  [Modpacks]             │  ← from game YAML categories
├───────────────────────────────────────────────────┤
│  Installed (3)                     [Update All]   │
│  ┌──────────────────────────────────────────────┐ │
│  │ ◆ ATM10 (modpack)              [Update] [✕]  │ │
│  │   Lithium       0.13.1   ✓ up to date   [✕] │ │
│  │   Fabric API    0.97.0   ↑ 0.98.1       [✕] │ │
│  │     └ dependency of Lithium                  │ │
│  │   Starlight     1.1.2   ✓ up to date    [✕] │ │
│  │   + 147 more from ATM10                      │ │
│  │ Replay Mod      1.16.6   ✓ up to date   [✕] │ │
│  │   └ manually added                          │ │
│  └──────────────────────────────────────────────┘ │
│                                                   │
│  🔍 Search mods...                                │
│  ┌──────────────────────────────────────────────┐ │
│  │ results from all sources for this category   │ │
│  └──────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────┘
```

**Key UX decisions:**

- Version and loader pickers live on the mods tab — they're the context for everything below.
- Version/loader are still stored as env vars on the gameserver. The mods tab provides a nicer UI than the raw env editor on settings.
- Version and loader also appear on the settings page (canonical location for all config). Both read/write the same env vars — change in one place, updates the other. Settings page shows loader with a note: "Tip: manage this from the Mods tab to see compatibility with installed mods."
- Games with no mods section: version/loader only on settings page (no mods tab exists).
- Changing version: system checks all installed mods for compatibility, shows inline warning with details.
- Changing loader: warns "this will remove all installed mods" since loaders are incompatible (Fabric mods don't work on Forge).
- Disabling a mod framework (e.g., Oxide off): warns "Disabling Oxide will deactivate 20 installed plugins. They'll remain on disk but won't load. Re-enable to use them again." Don't delete — user might re-enable.
- Categories (Mods, Resource Packs, Modpacks, etc.) are tabs defined by the game YAML.
- Search merges results from all sources for that category. Source shown as small badge on each result.
- Auto-installed dependencies are visually distinct ("dependency of X").
- Modpack mods are grouped under their pack header, collapsible. Individual mods from a pack can still be updated or removed independently.
- Games with no loader choice (Rust + Oxide): show an enable/disable toggle instead of a dropdown.
- Games with no version choice (Rust, CS2): no version picker shown.
- Update checking happens on mods tab open (cheap API call, can be batched). No background polling.

### CLI

```bash
# List installed mods
gamejanitor mods list <gameserver>
gamejanitor mods list <gameserver> --category "Resource Packs"

# Search across all sources for the game
gamejanitor mods search <gameserver> "lithium"
gamejanitor mods search <gameserver> "faithful" --category "Resource Packs"
gamejanitor mods search <gameserver> "all the mods" --category "Modpacks"

# Install mod (latest compatible version, auto-installs dependencies)
gamejanitor mods install <gameserver> modrinth:gvQqBUqZ
gamejanitor mods install <gameserver> lithium          # if unambiguous across sources

# Install specific version
gamejanitor mods install <gameserver> lithium --version 0.13.1

# Install modpack
gamejanitor mods install <gameserver> modrinth:ATM10 --category Modpacks

# Update
gamejanitor mods update <gameserver>                   # update all
gamejanitor mods update <gameserver> lithium           # update one

# Check for updates without applying
gamejanitor mods check <gameserver>

# Uninstall (auto-removes orphaned dependencies)
gamejanitor mods uninstall <gameserver> <mod-id>

# Export/import mod list (future — data model supports it)
gamejanitor mods export <gameserver> > mods.json
gamejanitor mods import <gameserver> < mods.json
```

Defaults to first category (usually "Mods"). `--category` for others. Source prefix (`modrinth:`, `workshop:`, `umod:`) for explicit source targeting.

## Game YAML — Mod Configuration

### Minecraft Java (full complexity)

```yaml
mods:
  version_env: MINECRAFT_VERSION
  loader:
    env: MODLOADER
    options:
      vanilla:
        mod_sources: []    # no mods on vanilla
      fabric:
        mod_sources: [modrinth, curseforge]
        loader_id: fabric  # what the source APIs call this loader
      forge:
        mod_sources: [modrinth, curseforge]
        loader_id: forge
      neoforge:
        mod_sources: [modrinth, curseforge]
        loader_id: neoforge
      paper:
        mod_sources: [modrinth]
        loader_id: paper

  categories:
    - name: Mods
      sources:
        - name: modrinth
          delivery: file
          install_path: /data/mods
          filters:
            project_type: mod
        - name: curseforge
          delivery: file
          install_path: /data/mods
          filters:
            class_id: 6    # CurseForge class ID for mods

    - name: Resource Packs
      sources:
        - name: modrinth
          delivery: file
          install_path: /data/resourcepacks
          filters:
            project_type: resourcepack

    - name: Data Packs
      sources:
        - name: modrinth
          delivery: file
          install_path: /data/datapacks
          filters:
            project_type: datapack

    - name: Shader Packs
      sources:
        - name: modrinth
          delivery: file
          install_path: /data/shaderpacks
          filters:
            project_type: shader

    - name: Modpacks
      sources:
        - name: modrinth
          delivery: pack
          install_path: /data/mods      # where individual mod files go
          overrides_path: /data          # where server-overrides/ extracts to
          filters:
            project_type: modpack
```

### Rust (simple, framework toggle)

```yaml
mods:
  loader:
    env: OXIDE_ENABLED
    options:
      "false":
        mod_sources: []
      "true":
        mod_sources: [umod]

  categories:
    - name: Plugins
      sources:
        - name: umod
          delivery: file
          install_path: /data/oxide/plugins
          config:
            category: rust    # umod-specific: which game's plugins
```

### ARK (Workshop, manifest-based)

```yaml
mods:
  categories:
    - name: Mods
      sources:
        - name: workshop
          delivery: manifest
          config:
            app_id: 346110
            manifest_path: /data/.gamejanitor/workshop_items.json
```

### Valheim (future — BepInEx framework + Thunderstore)

```yaml
mods:
  loader:
    env: BEPINEX_ENABLED
    options:
      "false":
        mod_sources: []
      "true":
        mod_sources: [thunderstore]

  categories:
    - name: Mods
      sources:
        - name: thunderstore
          delivery: file
          install_path: /data/BepInEx/plugins
          config:
            community: valheim
```

### Game with no mods

No `mods:` section. No mod tab. No code runs.

## Image Resolution (Separate System)

Image variants are a **runtime concern**, not a mod concern. They live separately in the game YAML.

```yaml
# Minecraft — dynamic resolution from Mojang API
runtime:
  resolver: minecraft-java   # Go function in games/ package
  default_image: ghcr.io/warsmite/gamejanitor/java17
  images:
    java8: ghcr.io/warsmite/gamejanitor/java8
    java17: ghcr.io/warsmite/gamejanitor/java17
    java21: ghcr.io/warsmite/gamejanitor/java21
    java25: ghcr.io/warsmite/gamejanitor/java25

# The minecraft-java resolver:
# 1. Takes the MINECRAFT_VERSION env value
# 2. Fetches the Mojang version manifest (cached)
# 3. Reads javaVersion.majorVersion for that version
# 4. Maps to the image tag: java{majorVersion}
# 5. Falls back to default_image if version not found
```

```yaml
# Rust — static, one image
runtime:
  image: ghcr.io/warsmite/gamejanitor/steamcmd
```

The resolver is a Go function in the `games/` package, not YAML logic. The Mojang API tells us the Java version — no hardcoding. If Mojang releases a version needing Java 25, we build a java25 image and it works automatically.

The controller calls `game.ResolveImage(env)` at create and start time. Three lines of integration. The mod system doesn't know or care about image resolution.

## Architecture

### Three Systems, Three Touchpoints

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Image Resolution │     │  Mod Config     │     │   Mod Service   │
│ (games/ package) │     │ (game YAML)     │     │ (controller/mod)│
│                  │     │                  │     │                  │
│ ResolveImage(env)│     │ AvailableSources │     │ Search, Install │
│   → image string │     │   (env) → []src  │     │ Update, Compat  │
└────────┬─────────┘     └────────┬─────────┘     └────────┬────────┘
         │                        │                         │
         │ touchpoint 1           │ touchpoint 2            │ touchpoint 3
         │ create/start           │ mod tab load            │ config change
         ▼                        ▼                         ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Gameserver Service                                 │
│                                                                      │
│  Create/Start: image = game.ResolveImage(env)                        │
│  Mod Tab: sources = game.AvailableModSources(env)                    │
│  Config Change: issues = modService.CheckCompatibility(gsID, newEnv) │
└──────────────────────────────────────────────────────────────────────┘
```

### Mod Service Interfaces

```go
// ModCatalog — how we discover mods from a source.
// Each source (Modrinth, uMod, Workshop, Thunderstore, CurseForge)
// implements this interface.
type ModCatalog interface {
    // Search returns mods matching a query, filtered by game version,
    // loader, and source-specific filters from the game YAML.
    Search(ctx context.Context, query string, filters CatalogFilters) ([]ModResult, int, error)

    // GetDetails returns full info for a single mod.
    GetDetails(ctx context.Context, modID string) (*ModDetails, error)

    // GetVersions returns available versions for a mod.
    // Filtered by game version and loader for compatibility display.
    GetVersions(ctx context.Context, modID string, filters CatalogFilters) ([]ModVersion, error)

    // GetDependencies returns required and optional dependencies for a version.
    // Returns nil if the source doesn't support dependency info.
    GetDependencies(ctx context.Context, versionID string) ([]ModDependency, error)
}

// CatalogFilters carries resolved filter values for catalog queries.
// Populated from game YAML compatibility config + current server env.
type CatalogFilters struct {
    GameVersion string            // e.g., "1.21.1"
    Loader      string            // e.g., "fabric" (source's terminology, from loader_id)
    Extra       map[string]string // source-specific: project_type, category, app_id, etc.
}

// ModDependency represents a dependency relationship from a catalog.
type ModDependency struct {
    ModID     string // source's ID for the dependency
    VersionID string // specific version required (empty = latest compatible)
    Required  bool   // true = must install, false = optional/recommended
}
```

### Delivery Mechanisms

Not an interface — there are exactly three, and they're different enough to not share one.

```go
// FileDelivery downloads a file from a URL and streams it to the
// gameserver volume. Never buffers the full file in memory.
type FileDelivery struct {
    fileSvc    FileOperator
    httpClient *http.Client
}

func (d *FileDelivery) Install(ctx context.Context, gameserverID string, installPath string, downloadURL string, fileName string) error {
    // Stream: HTTP response body → fileSvc.WriteFileStream → worker volume
    // No []byte buffer. Respects MaxModDownloadBytes via LimitReader.
}

func (d *FileDelivery) Uninstall(ctx context.Context, gameserverID string, filePath string) error {
    return d.fileSvc.DeletePath(ctx, gameserverID, filePath)
}

// ManifestDelivery writes mod IDs to a JSON manifest file.
// The game server reads this manifest and downloads mods itself
// (e.g., via SteamCMD for Workshop items).
type ManifestDelivery struct {
    fileSvc FileOperator
}

func (d *ManifestDelivery) Install(ctx context.Context, gameserverID string, manifestPath string, allModIDs []string) error {
    // Write the complete list of mod IDs as JSON array to manifestPath.
    // Overwrites the entire file each time (manifest is the full truth).
}

func (d *ManifestDelivery) Uninstall(ctx context.Context, gameserverID string, manifestPath string, remainingModIDs []string) error {
    // Rewrite manifest without the removed mod. Delete file if empty.
}

// PackDelivery installs a modpack — a bundle of mods + config overrides.
// Used for Modrinth .mrpack files and potentially CurseForge modpacks.
type PackDelivery struct {
    fileSvc    FileOperator
    httpClient *http.Client
}

func (d *PackDelivery) Install(ctx context.Context, gameserverID string, packURL string, installPath string, overridesPath string) (*PackContents, error) {
    // 1. Download the .mrpack (ZIP) file — stream to temp, don't hold in memory
    // 2. Parse modrinth.index.json from the ZIP
    // 3. Filter files: keep server-side only (env.server != "unsupported")
    // 4. Download each mod file (with SHA512 hash verification)
    // 5. Stream each to installPath via fileSvc
    // 6. Validate server-overrides/ paths — reject any containing ".." or
    //    absolute paths to prevent directory traversal
    // 7. Extract server-overrides/ to overridesPath
    // 8. Return PackContents listing all installed mods + overridden files
}

// PackContents is the parsed result of installing a modpack.
type PackContents struct {
    Mods      []PackMod // each mod that was installed
    Overrides []string  // paths of config files extracted
}

type PackMod struct {
    SourceID    string // modrinth project ID (from the mrpack file hashes)
    FileName    string
    FilePath    string
    Version     string
    DownloadURL string
}
```

### Mod Service

```go
type ModService struct {
    catalogs    map[string]ModCatalog    // "modrinth", "umod", "workshop", etc.
    fileDel     *FileDelivery
    manifestDel *ManifestDelivery
    packDel     *PackDelivery
    store       ModStore
    gameStore   *games.GameStore
    broadcaster *controller.EventBus
    log         *slog.Logger
}

// Install finds the right version, resolves dependencies, and delivers
// the mod to the gameserver.
func (s *ModService) Install(ctx context.Context, gameserverID string, sourceName string, modID string, versionID string, category string) (*InstalledMod, error) {
    gs := s.getGameserver(gameserverID)
    catConfig := s.getCategoryConfig(gs.GameID, category, sourceName)
    catalog := s.catalogs[sourceName]
    filters := s.buildFilters(gs, catConfig)

    // Resolve version
    version := s.resolveVersion(ctx, catalog, modID, versionID, filters)

    // Check for duplicates
    s.checkNotInstalled(gameserverID, sourceName, modID)

    // Resolve and install dependencies (recursive, file sources only)
    s.installDependencies(ctx, gameserverID, catalog, version, catConfig, filters, 0)

    // Deliver the mod
    switch catConfig.Delivery {
    case "file":
        s.fileDel.Install(ctx, gameserverID, catConfig.InstallPath, version.DownloadURL, version.FileName)
    case "manifest":
        allIDs := s.allManifestIDs(gameserverID, sourceName, modID)
        s.manifestDel.Install(ctx, gameserverID, catConfig.ManifestPath, allIDs)
    }

    // Record in DB and publish event
    mod := s.recordInstall(gameserverID, sourceName, modID, version, catConfig, false, "")
    s.publishEvent(ctx, gameserverID, mod, controller.EventModInstalled)
    return mod, nil
}

// Uninstall removes a mod and any orphaned auto-installed dependencies.
func (s *ModService) Uninstall(ctx context.Context, gameserverID string, modID string) error {
    mod := s.store.GetInstalledMod(modID)

    // If this is a modpack, remove all mods linked to it
    if mod.Delivery == "pack" {
        s.uninstallPackMods(ctx, gameserverID, modID)
    }

    // Uninstall the mod itself
    s.deliverUninstall(ctx, gameserverID, mod)
    s.store.DeleteInstalledMod(modID)

    // Clean up orphaned dependencies (not for pack mods — those are handled above)
    if mod.PackID == "" {
        s.removeOrphanedDependencies(ctx, gameserverID, modID)
    }

    s.publishEvent(ctx, gameserverID, mod, controller.EventModUninstalled)
    return nil
}

// CheckForUpdates compares installed versions against latest from catalogs.
func (s *ModService) CheckForUpdates(ctx context.Context, gameserverID string) ([]ModUpdate, error) {
    // For each installed file-delivery mod (not auto-installed deps):
    //   - Query catalog for latest compatible version
    //   - Compare with installed version
    //   - Return list of available updates
    // For modpacks:
    //   - Check if the modpack has a newer version on the catalog
    //   - Return as a pack update (not individual mod updates)
    // Workshop/manifest mods are skipped (server downloads latest automatically)
}

// Update replaces an installed mod with a newer version.
func (s *ModService) Update(ctx context.Context, gameserverID string, modID string) (*InstalledMod, error) {
    // Get current mod → get latest version from catalog → download → replace file → update DB
    // Also updates dependencies if needed
}

// UpdateAll updates all mods that have available updates.
func (s *ModService) UpdateAll(ctx context.Context, gameserverID string) ([]ModUpdate, error) {
    // CheckForUpdates → Update each
}

// InstallPack installs a modpack — downloads all mods, extracts config overrides.
func (s *ModService) InstallPack(ctx context.Context, gameserverID string, sourceName string, packID string, versionID string) (*PackInstallResult, error) {
    gs := s.getGameserver(gameserverID)
    catConfig := s.getCategoryConfig(gs.GameID, "Modpacks", sourceName)
    catalog := s.catalogs[sourceName]
    filters := s.buildFilters(gs, catConfig)

    // Resolve pack version
    version := s.resolveVersion(ctx, catalog, packID, versionID, filters)

    // Download and parse pack via PackDelivery
    contents, err := s.packDel.Install(ctx, gameserverID, version.DownloadURL,
        catConfig.InstallPath, catConfig.OverridesPath)

    // Record the modpack itself as an installed_mod entry
    pack := s.recordInstall(gameserverID, sourceName, packID, version, catConfig, false, "")

    // Record each individual mod, linked to the pack
    for _, mod := range contents.Mods {
        // Check for duplicates — same mod already installed?
        existing := s.store.GetInstalledModBySource(gameserverID, sourceName, mod.SourceID)
        if existing != nil {
            if existing.VersionID == mod.Version {
                // Same version — skip, just link to pack
                s.store.SetPackID(existing.ID, pack.ID)
                continue
            }
            // Different version — update to pack's version, link to pack
            s.deliverUninstall(ctx, gameserverID, existing)
            s.store.DeleteInstalledMod(existing.ID)
        }
        s.recordInstall(gameserverID, sourceName, mod.SourceID, &ModVersion{
            VersionID: mod.Version, FileName: mod.FileName, DownloadURL: mod.DownloadURL,
        }, catConfig, false, pack.ID)
    }

    return &PackInstallResult{
        Pack:       pack,
        ModCount:   len(contents.Mods),
        Overrides:  contents.Overrides,
    }, nil
}

// UpdatePack updates a modpack to a newer version.
// Diffs old vs new mod lists: adds new mods, removes dropped mods,
// updates changed versions. Respects user exclusions.
func (s *ModService) UpdatePack(ctx context.Context, gameserverID string, packModID string) (*PackInstallResult, error) {
    pack := s.store.GetInstalledMod(packModID)
    catalog := s.catalogs[pack.Source]

    // Get latest pack version
    filters := s.buildFiltersForGameserver(gameserverID)
    versions := catalog.GetVersions(ctx, pack.SourceID, filters)
    latestVersion := versions[0]

    // Download and parse new pack
    catConfig := s.getCategoryConfigForPack(gameserverID, pack)
    newContents, err := s.packDel.Install(ctx, gameserverID, latestVersion.DownloadURL,
        catConfig.InstallPath, catConfig.OverridesPath)

    // Load current pack mods and exclusions
    currentMods := s.store.ListModsByPackID(gameserverID, packModID)
    exclusions := s.store.GetPackExclusions(packModID)

    // Build lookup of new pack's mods
    newModSet := make(map[string]*PackMod)
    for i := range newContents.Mods {
        newModSet[newContents.Mods[i].SourceID] = &newContents.Mods[i]
    }

    // Remove mods dropped from the pack (unless user-modified)
    for _, current := range currentMods {
        if _, inNew := newModSet[current.SourceID]; !inNew {
            // Mod removed from pack — remove it (unless user excluded it,
            // which means they removed it on purpose, so already gone)
            s.deliverUninstall(ctx, gameserverID, &current)
            s.store.DeleteInstalledMod(current.ID)
        }
    }

    // Add new mods and update changed versions (skip excluded)
    for sourceID, newMod := range newModSet {
        if exclusions[sourceID] {
            continue // user explicitly removed this mod, don't re-add
        }

        existing := s.store.GetInstalledModBySource(gameserverID, pack.Source, sourceID)
        if existing != nil && existing.VersionID == newMod.Version {
            continue // same version, skip
        }
        if existing != nil {
            // Version changed — update
            s.deliverUninstall(ctx, gameserverID, existing)
            s.store.DeleteInstalledMod(existing.ID)
        }
        // Install new/updated mod
        s.fileDel.Install(ctx, gameserverID, catConfig.InstallPath, newMod.DownloadURL, newMod.FileName)
        s.recordInstall(gameserverID, pack.Source, sourceID, &ModVersion{
            VersionID: newMod.Version, FileName: newMod.FileName, DownloadURL: newMod.DownloadURL,
        }, catConfig, false, packModID)
    }

    // Update pack record itself
    s.store.UpdateModVersion(packModID, latestVersion.VersionID, latestVersion.Version)

    return &PackInstallResult{Pack: pack, ModCount: len(newContents.Mods), Overrides: newContents.Overrides}, nil
}

// CheckCompatibility checks if installed mods are compatible with new env values.
// Called by gameserver service before applying version/loader changes.
func (s *ModService) CheckCompatibility(ctx context.Context, gameserverID string, newEnv model.Env) ([]ModIssue, error) {
    // Three cases:
    //
    // 1. Version changed (e.g., 1.21 → 1.22):
    //    For each installed mod, query catalog with new version.
    //    If no compatible version exists → ModIssue{Mod, "incompatible with 1.22"}
    //
    // 2. Loader changed (e.g., Fabric → Forge):
    //    All mods are incompatible. Return ModIssue for every installed mod.
    //    UI shows "Changing loader will remove all installed mods."
    //
    // 3. Framework disabled (e.g., OXIDE_ENABLED false):
    //    All mods deactivated but NOT removed. Return ModIssue with
    //    type "deactivated" (not "incompatible"). UI shows
    //    "Plugins will remain on disk but won't load."
}
```

### Dependency Handling

Dependencies are resolved at install time, one level at a time (recursive). Not a constraint solver. Depth capped at 10 levels (warning at 5).

```go
const maxDepDepth = 10

func (s *ModService) installDependencies(ctx context.Context, gameserverID string, catalog ModCatalog, version *ModVersion, catConfig *CategoryConfig, filters CatalogFilters, depth int) error {
    if depth >= maxDepDepth {
        return fmt.Errorf("dependency depth limit reached (%d levels)", maxDepDepth)
    }
    if depth >= 5 {
        s.log.Warn("deep dependency chain", "depth", depth, "mod", version.VersionID)
    }

    deps, err := catalog.GetDependencies(ctx, version.VersionID)
    if err != nil || deps == nil {
        return nil // source doesn't support deps, or no deps — fine
    }

    for _, dep := range deps {
        if !dep.Required {
            continue // skip optional deps
        }

        // Already installed?
        existing, _ := s.store.GetInstalledModBySource(gameserverID, catConfig.SourceName, dep.ModID)
        if existing != nil {
            continue
        }

        // Resolve the dependency's best version
        depVersion := s.resolveVersion(ctx, catalog, dep.ModID, dep.VersionID, filters)

        // Recursive: install the dependency's dependencies first
        s.installDependencies(ctx, gameserverID, catalog, depVersion, catConfig, filters, depth+1)

        // Install the dependency itself
        if catConfig.Delivery == "file" {
            s.fileDel.Install(ctx, gameserverID, catConfig.InstallPath, depVersion.DownloadURL, depVersion.FileName)
        }

        // Record as auto-installed, tracking which mod required it
        s.recordInstall(gameserverID, catConfig.SourceName, dep.ModID, depVersion, catConfig, true, "")
    }
    return nil
}
```

When uninstalling a mod, orphaned dependencies (auto-installed, no other mod depends on them) are cleaned up:

```go
func (s *ModService) removeOrphanedDependencies(ctx context.Context, gameserverID string, removedModID string) {
    // Find all auto-installed mods that depended on removedModID
    autoMods := s.store.ListAutoInstalledDependencies(gameserverID, removedModID)

    for _, dep := range autoMods {
        // Check if any OTHER installed mod still depends on this
        if s.hasOtherDependents(gameserverID, dep.ID, removedModID) {
            continue
        }

        // No other dependents — remove it
        s.deliverUninstall(ctx, gameserverID, &dep)
        s.store.DeleteInstalledMod(dep.ID)
    }
}
```

Cross-source dependencies are not supported. If a Modrinth mod declares a dependency only available on CurseForge, the system skips it and warns the user to install it manually.

### Catalog Implementation Notes

**Modrinth:**
- Search: uses facets for project_type, loader, game_version filtering
- GetVersions: filter by loader, return all game versions (UI shows compatibility matrix)
- GetDependencies: Modrinth version response includes `dependencies[]` with `project_id`, `version_id`, and `dependency_type` (required/optional)
- Supports: mods, resource packs, data packs, shader packs, **modpacks** (via project_type filter)
- Modpacks: `.mrpack` format (ZIP with `modrinth.index.json`). Each file has `env.server` field for server-side filtering. Files include SHA512 hashes for verification. `server-overrides/` folder contains config files to extract. Modpack versions specify game version + loader, so version/loader can be auto-set from the modpack.

**uMod:**
- Search: filter by game category (rust, hurtworld, etc. — from game YAML config)
- GetVersions: single version per plugin (latest only)
- GetDependencies: returns nil (uMod plugins are standalone)
- Supports: plugins only

**Workshop:**
- Search: filter by app_id. Requires Steam API key for search, paste-ID mode without.
- GetVersions: returns single entry (Workshop items don't have discrete versions)
- GetDependencies: returns nil (Steam handles deps at download time)
- Delivery: manifest (IDs written to JSON, SteamCMD downloads at server start)

**Thunderstore (future):**
- Search: filter by community (valheim, lethal-company, etc.)
- GetVersions: proper version list
- GetDependencies: Thunderstore manifests include dependency strings
- Delivery: file (zip extraction to plugin path)

**CurseForge (future):**
- Search: filter by game_id, class_id (6=mods, 12=texture packs, etc.)
- GetVersions: filter by game version and mod loader
- GetDependencies: CurseForge API includes `dependencies[]`
- Delivery: file
- Note: requires API key registration with CurseForge

## Data Model Changes

### installed_mods table

```sql
CREATE TABLE installed_mods (
    id TEXT PRIMARY KEY,
    gameserver_id TEXT NOT NULL REFERENCES gameservers(id) ON DELETE CASCADE,
    source TEXT NOT NULL,              -- "modrinth", "umod", "workshop"
    source_id TEXT NOT NULL,           -- source's unique ID for the mod
    category TEXT NOT NULL DEFAULT '', -- "Mods", "Resource Packs", "Modpacks", etc.
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    version_id TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL DEFAULT '',    -- empty for manifest and pack delivery
    file_name TEXT NOT NULL DEFAULT '',
    delivery TEXT NOT NULL DEFAULT 'file', -- "file", "manifest", or "pack"
    auto_installed BOOLEAN NOT NULL DEFAULT 0,
    depends_on TEXT,                        -- mod ID this was auto-installed for (nullable)
    pack_id TEXT,                           -- installed_mod ID of parent modpack (nullable)
    metadata JSON NOT NULL DEFAULT '{}',
    installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_installed_mods_gameserver ON installed_mods(gameserver_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_installed_mods_unique ON installed_mods(gameserver_id, source, source_id);
CREATE INDEX IF NOT EXISTS idx_installed_mods_depends ON installed_mods(depends_on) WHERE depends_on IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_installed_mods_pack ON installed_mods(pack_id) WHERE pack_id IS NOT NULL;
```

### pack_exclusions table

Tracks mods the user explicitly removed from a modpack, so pack updates don't re-add them.

```sql
CREATE TABLE pack_exclusions (
    id TEXT PRIMARY KEY,
    pack_mod_id TEXT NOT NULL REFERENCES installed_mods(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL,  -- the excluded mod's source ID
    excluded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pack_exclusions_unique ON pack_exclusions(pack_mod_id, source_id);
```

When a user removes an individual mod that has a `pack_id`, the system:
1. Removes the mod
2. Records an exclusion in `pack_exclusions` for that pack + source_id
3. On pack update, the exclusion prevents re-adding the mod

## What Gets Deleted

- `resolveFilters()` and the env var parsing dance
- `checkPreconditions()` — replaced by loader options in YAML
- Workshop special-casing in Install/Uninstall (`installWorkshop`, `uninstallWorkshop`, `writeWorkshopManifest`)
- `Download()` on the ModSource interface (never properly used)
- The loader field overloading hack (Workshop passing appID as "loader")
- In-memory download buffering (`downloadFile` reading full []byte)
- The entire current `ModSource` interface (replaced by `ModCatalog`)

## What Stays

- `installed_mods` table structure (extended, not replaced)
- Modrinth, uMod, Workshop API client code (reimplements `ModCatalog` instead of `ModSource`)
- Game YAML as the definition format (extended with `mods.categories` and `runtime`)
- File operations via worker (`FileOperator` interface)
- Event publishing for mod install/uninstall

## Migration Path

This is a pre-release redesign — no migration needed. Replace the mod system, update the game YAMLs, update the UI and API endpoints. The `installed_mods` table gets the new columns added to the initial schema. The `pack_exclusions` table is new.

## Resolved Decisions

1. **Mod update checking frequency** — Check on mods tab open (cheap API call, can be batched via Modrinth's multi-project endpoint). CLI via `gamejanitor mods check`. No background polling.

2. **Dependency depth limit** — Cap recursion at 10 levels, log warning at 5. In practice Modrinth deps are 1-2 deep. Bail with error if limit hit.

3. **Cross-source dependencies** — Don't support. If a Modrinth mod declares a dependency only available on CurseForge, skip it and tell the user to install manually.

4. **Modpacks** — First-class support via `delivery: pack`. Modrinth `.mrpack` format: ZIP containing `modrinth.index.json` (mod list with hashes and download URLs) + `server-overrides/` (config files). Individual mods from a pack are tracked separately in the DB and can be updated or removed independently. Pack updates diff old vs new mod lists and respect user exclusions.

5. **Auto-update** — User-triggered only. "Updates available" indicator on mods tab and in `gamejanitor mods check` output. No scheduled auto-update.

6. **Export/import** — Deferred but the data model supports it. `installed_mods` has all the info needed to serialize and replay. Future CLI: `gamejanitor mods export/import`.

## Implementation Notes

**Modpack duplicate handling:**
- Same mod, same version → skip, link to pack via `pack_id`.
- Same mod, different version → update to the pack's version, link to pack.
- New mod → install and link to pack.

**Modpack version/loader mismatch:**
- Don't auto-change the server's version/loader. Warn: "This modpack requires Fabric 1.20.1, your server is Fabric 1.21.1 — install anyway?"
- Let the user decide. Version downgrades can break worlds.

**Modpack update flow:**
- Diff old pack contents against new: add new mods, remove dropped mods, update version-changed mods.
- Skip mods in `pack_exclusions` (user explicitly removed them).
- Re-extract `server-overrides/` (overwrites config files — the pre-install backup prompt covers this).

**Server-overrides safety:**
- Validate all paths from `server-overrides/` — reject any containing `..` or absolute paths to prevent directory traversal.
- Before overwriting existing files, show a pre-install summary: "This modpack will overwrite 3 config files: server.properties, ..."
- Offer a one-click "[Create Backup]" button inline on the install confirmation dialog. Non-blocking — power users can skip it.

**Framework deactivation (distinct from incompatibility):**
- Disabling a mod framework (Oxide, BepInEx) doesn't delete mods.
- Mods stay on disk, just don't load (the framework isn't running).
- Compatibility checker returns issues with type "deactivated" (not "incompatible").
- UI shows: "Disabling Oxide will deactivate 20 plugins. They'll remain on disk. Re-enable to use them."
- Re-enabling the framework immediately reactivates all mods — no reinstall needed.

**Version/loader picker location:**
- Version and loader live on BOTH the settings page and the mods tab.
- Both read/write the same env vars — change in one place, updates the other.
- Settings page shows loader with a note: "Tip: manage this from the Mods tab to see compatibility with installed mods."
- Mods tab surfaces them as context for mod search and compatibility.
- Games with no mods section: version/loader only on settings page (no mods tab exists).
