package gamejanitor

import (
	"context"
	"fmt"
	"net/url"
)

// ModService handles mod management API calls for gameservers.
type ModService struct {
	client *Client
}

// Config returns the full mods tab configuration: version picker, loader picker,
// and available categories for a gameserver.
func (s *ModService) Config(ctx context.Context, gameserverID string) (*ModTabConfig, error) {
	var config ModTabConfig
	if err := s.client.get(ctx, "/api/gameservers/"+gameserverID+"/mods/config", &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// List returns all installed mods for a gameserver.
func (s *ModService) List(ctx context.Context, gameserverID string) ([]InstalledMod, error) {
	var mods []InstalledMod
	if err := s.client.get(ctx, "/api/gameservers/"+gameserverID+"/mods", &mods); err != nil {
		return nil, err
	}
	return mods, nil
}

// Search searches for mods within a category.
func (s *ModService) Search(ctx context.Context, gameserverID, category, query string, offset, limit int) (*SearchResults, error) {
	v := url.Values{}
	v.Set("category", category)
	if query != "" {
		v.Set("q", query)
	}
	if offset > 0 {
		v.Set("offset", fmt.Sprintf("%d", offset))
	}
	if limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", limit))
	}

	var results SearchResults
	if err := s.client.get(ctx, "/api/gameservers/"+gameserverID+"/mods/search?"+v.Encode(), &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// Versions returns available versions for a specific mod.
func (s *ModService) Versions(ctx context.Context, gameserverID, category, source, sourceID string) ([]ModVersion, error) {
	v := url.Values{}
	v.Set("category", category)
	v.Set("source", source)
	v.Set("source_id", sourceID)

	var versions []ModVersion
	if err := s.client.get(ctx, "/api/gameservers/"+gameserverID+"/mods/versions?"+v.Encode(), &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

// CheckUpdates returns available updates for installed mods.
func (s *ModService) CheckUpdates(ctx context.Context, gameserverID string) ([]ModUpdate, error) {
	var updates []ModUpdate
	if err := s.client.get(ctx, "/api/gameservers/"+gameserverID+"/mods/updates", &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// Install installs a mod on a gameserver.
func (s *ModService) Install(ctx context.Context, gameserverID string, req *InstallModRequest) (*InstalledMod, error) {
	var mod InstalledMod
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods", req, &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}

// InstallPack installs a modpack on a gameserver.
func (s *ModService) InstallPack(ctx context.Context, gameserverID string, req *InstallPackRequest) error {
	return s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/pack", req, nil)
}

// Uninstall removes a mod from a gameserver.
func (s *ModService) Uninstall(ctx context.Context, gameserverID, modID string) error {
	return s.client.delete(ctx, "/api/gameservers/"+gameserverID+"/mods/"+modID)
}

// Update updates a single mod to the latest compatible version.
func (s *ModService) Update(ctx context.Context, gameserverID, modID string) (*InstalledMod, error) {
	var mod InstalledMod
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/"+modID+"/update", nil, &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}

// UpdateAll updates all mods to their latest compatible versions.
func (s *ModService) UpdateAll(ctx context.Context, gameserverID string) ([]ModUpdate, error) {
	var updates []ModUpdate
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/update-all", nil, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// UpdatePack updates a modpack to a newer version.
func (s *ModService) UpdatePack(ctx context.Context, gameserverID, modID string) error {
	return s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/"+modID+"/update-pack", nil, nil)
}

// CheckCompatibility checks if installed mods are compatible with proposed env changes.
func (s *ModService) CheckCompatibility(ctx context.Context, gameserverID string, env map[string]string) ([]ModIssue, error) {
	var issues []ModIssue
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/check-compatibility", map[string]any{"env": env}, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// Scan cross-references files on disk with the DB.
func (s *ModService) Scan(ctx context.Context, gameserverID string) (*ScanResult, error) {
	var result ScanResult
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/scan", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TrackFile creates a DB record for an untracked file found on disk.
func (s *ModService) TrackFile(ctx context.Context, gameserverID string, req *TrackFileRequest) (*InstalledMod, error) {
	var mod InstalledMod
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/track", req, &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}

// InstallFromURL installs a mod from a plain URL.
func (s *ModService) InstallFromURL(ctx context.Context, gameserverID string, req *InstallURLRequest) (*InstalledMod, error) {
	var mod InstalledMod
	if err := s.client.post(ctx, "/api/gameservers/"+gameserverID+"/mods/url", req, &mod); err != nil {
		return nil, err
	}
	return &mod, nil
}
