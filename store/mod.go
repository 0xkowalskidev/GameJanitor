package store

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/warsmite/gamejanitor/model"
)

type ModStore struct {
	db *sql.DB
}

func NewModStore(db *sql.DB) *ModStore {
	return &ModStore{db: db}
}

const modColumns = "id, gameserver_id, source, source_id, category, name, version, version_id, file_path, file_name, delivery, auto_installed, depends_on, pack_id, metadata, installed_at"

func scanMod(scanner interface{ Scan(...any) error }) (*model.InstalledMod, error) {
	var m model.InstalledMod
	err := scanner.Scan(
		&m.ID, &m.GameserverID, &m.Source, &m.SourceID, &m.Category,
		&m.Name, &m.Version, &m.VersionID, &m.FilePath, &m.FileName,
		&m.Delivery, &m.AutoInstalled, &m.DependsOn, &m.PackID,
		&m.Metadata, &m.InstalledAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *ModStore) ListInstalledMods(gameserverID string) ([]model.InstalledMod, error) {
	rows, err := s.db.Query(
		"SELECT "+modColumns+" FROM installed_mods WHERE gameserver_id = ? ORDER BY installed_at DESC",
		gameserverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mods []model.InstalledMod
	for rows.Next() {
		m, err := scanMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, *m)
	}
	if mods == nil {
		mods = []model.InstalledMod{}
	}
	return mods, rows.Err()
}

func (s *ModStore) GetInstalledMod(id string) (*model.InstalledMod, error) {
	m, err := scanMod(s.db.QueryRow(
		"SELECT "+modColumns+" FROM installed_mods WHERE id = ?", id,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *ModStore) GetInstalledModBySource(gameserverID, source, sourceID string) (*model.InstalledMod, error) {
	m, err := scanMod(s.db.QueryRow(
		"SELECT "+modColumns+" FROM installed_mods WHERE gameserver_id = ? AND source = ? AND source_id = ?",
		gameserverID, source, sourceID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *ModStore) CreateInstalledMod(m *model.InstalledMod) error {
	_, err := s.db.Exec(
		"INSERT INTO installed_mods (id, gameserver_id, source, source_id, category, name, version, version_id, file_path, file_name, delivery, auto_installed, depends_on, pack_id, metadata, installed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.GameserverID, m.Source, m.SourceID, m.Category,
		m.Name, m.Version, m.VersionID, m.FilePath, m.FileName,
		m.Delivery, m.AutoInstalled, m.DependsOn, m.PackID,
		m.Metadata, m.InstalledAt,
	)
	return err
}

func (s *ModStore) DeleteInstalledMod(id string) error {
	_, err := s.db.Exec("DELETE FROM installed_mods WHERE id = ?", id)
	return err
}

func (s *ModStore) ListModsByPackID(gameserverID, packID string) ([]model.InstalledMod, error) {
	rows, err := s.db.Query(
		"SELECT "+modColumns+" FROM installed_mods WHERE gameserver_id = ? AND pack_id = ? ORDER BY name",
		gameserverID, packID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mods []model.InstalledMod
	for rows.Next() {
		m, err := scanMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, *m)
	}
	return mods, rows.Err()
}

func (s *ModStore) GetPackExclusions(packModID string) (map[string]bool, error) {
	rows, err := s.db.Query(
		"SELECT source_id FROM pack_exclusions WHERE pack_mod_id = ?",
		packModID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exclusions := make(map[string]bool)
	for rows.Next() {
		var sourceID string
		if err := rows.Scan(&sourceID); err != nil {
			return nil, err
		}
		exclusions[sourceID] = true
	}
	return exclusions, rows.Err()
}

func (s *ModStore) CreatePackExclusion(e *model.PackExclusion) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO pack_exclusions (id, pack_mod_id, source_id, excluded_at) VALUES (?, ?, ?, ?)",
		e.ID, e.PackModID, e.SourceID, e.ExcludedAt,
	)
	return err
}

func (s *ModStore) SetModPackID(modID, packID string) error {
	_, err := s.db.Exec("UPDATE installed_mods SET pack_id = ? WHERE id = ?", packID, modID)
	return err
}

func (s *ModStore) UpdateModVersion(modID, versionID, version string) error {
	_, err := s.db.Exec("UPDATE installed_mods SET version_id = ?, version = ? WHERE id = ?", versionID, version, modID)
	return err
}
