package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Game struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Image                string          `json:"image"`
	DefaultPorts         json.RawMessage `json:"default_ports"`
	DefaultEnv           json.RawMessage `json:"default_env"`
	MinMemoryMB          int             `json:"min_memory_mb"`
	MinCPU               float64         `json:"min_cpu"`
	GSQGameSlug          *string         `json:"gsq_game_slug"`
	DisabledCapabilities json.RawMessage `json:"disabled_capabilities"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

func ListGames(db *sql.DB) ([]Game, error) {
	rows, err := db.Query("SELECT id, name, image, default_ports, default_env, min_memory_mb, min_cpu, gsq_game_slug, disabled_capabilities, created_at, updated_at FROM games ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("listing games: %w", err)
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.Name, &g.Image, &g.DefaultPorts, &g.DefaultEnv, &g.MinMemoryMB, &g.MinCPU, &g.GSQGameSlug, &g.DisabledCapabilities, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning game row: %w", err)
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

func GetGame(db *sql.DB, id string) (*Game, error) {
	var g Game
	err := db.QueryRow("SELECT id, name, image, default_ports, default_env, min_memory_mb, min_cpu, gsq_game_slug, disabled_capabilities, created_at, updated_at FROM games WHERE id = ?", id).
		Scan(&g.ID, &g.Name, &g.Image, &g.DefaultPorts, &g.DefaultEnv, &g.MinMemoryMB, &g.MinCPU, &g.GSQGameSlug, &g.DisabledCapabilities, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting game %s: %w", id, err)
	}
	return &g, nil
}

func CreateGame(db *sql.DB, g *Game) error {
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now

	_, err := db.Exec(
		"INSERT INTO games (id, name, image, default_ports, default_env, min_memory_mb, min_cpu, gsq_game_slug, disabled_capabilities, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.ID, g.Name, g.Image, g.DefaultPorts, g.DefaultEnv, g.MinMemoryMB, g.MinCPU, g.GSQGameSlug, g.DisabledCapabilities, g.CreatedAt, g.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating game %s: %w", g.ID, err)
	}
	return nil
}

func UpdateGame(db *sql.DB, g *Game) error {
	g.UpdatedAt = time.Now()

	result, err := db.Exec(
		"UPDATE games SET name = ?, image = ?, default_ports = ?, default_env = ?, min_memory_mb = ?, min_cpu = ?, gsq_game_slug = ?, disabled_capabilities = ?, updated_at = ? WHERE id = ?",
		g.Name, g.Image, g.DefaultPorts, g.DefaultEnv, g.MinMemoryMB, g.MinCPU, g.GSQGameSlug, g.DisabledCapabilities, g.UpdatedAt, g.ID,
	)
	if err != nil {
		return fmt.Errorf("updating game %s: %w", g.ID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for game %s: %w", g.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("game %s not found", g.ID)
	}
	return nil
}

func DeleteGame(db *sql.DB, id string) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM gameservers WHERE game_id = ?", id).Scan(&count); err != nil {
		return fmt.Errorf("checking gameserver references for game %s: %w", id, err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete game '%s': %d gameservers still reference this game. Delete or reassign them first", id, count)
	}

	result, err := db.Exec("DELETE FROM games WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting game %s: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for game %s: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("game %s not found", id)
	}
	return nil
}
