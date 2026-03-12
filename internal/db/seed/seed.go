package seed

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func SeedGames(db *sql.DB) error {
	games := []struct {
		id                   string
		name                 string
		image                string
		defaultPorts         string
		defaultEnv           string
		minMemoryMB          int
		minCPU               float64
		gsqGameSlug          string
		disabledCapabilities string
	}{
		{
			id:    "minecraft-java",
			name:  "Minecraft: Java Edition",
			image: "registry.0xkowalski.dev/gamejanitor/minecraft-java",
			defaultPorts: `[{"name":"game","port":25565,"protocol":"tcp"}]`,
			defaultEnv: `[
				{"key":"EULA","default":"false","label":"Accept Minecraft EULA","type":"boolean"},
				{"key":"GAMEMODE","default":"survival","label":"Game Mode","type":"select","options":["survival","creative","adventure","spectator"]},
				{"key":"MAX_PLAYERS","default":"20","label":"Max Players","type":"number"},
				{"key":"DIFFICULTY","default":"normal","label":"Difficulty","type":"select","options":["peaceful","easy","normal","hard"]},
				{"key":"MOTD","default":"A Gamejanitor Server","label":"Message of the Day"},
				{"key":"PVP","default":"true","label":"PvP","type":"boolean"},
				{"key":"SERVER_PORT","default":"25565","system":true}
			]`,
			minMemoryMB:          2048,
			minCPU:               1.0,
			gsqGameSlug:          "minecraft",
			disabledCapabilities: `[]`,
		},
		{
			id:    "rust",
			name:  "Rust",
			image: "registry.0xkowalski.dev/gamejanitor/rust",
			defaultPorts: `[{"name":"game","port":28015,"protocol":"udp"},{"name":"rcon","port":28016,"protocol":"tcp"}]`,
			defaultEnv: `[
				{"key":"SERVER_MAXPLAYERS","default":"50","label":"Max Players","type":"number"},
				{"key":"SERVER_HOSTNAME","default":"Gamejanitor Rust Server","label":"Server Name"},
				{"key":"SERVER_WORLDSIZE","default":"3000","label":"World Size","type":"number"},
				{"key":"RCON_PASSWORD","default":"changeme","label":"RCON Password"},
				{"key":"SERVER_PORT","default":"28015","system":true},
				{"key":"RCON_PORT","default":"28016","system":true}
			]`,
			minMemoryMB:          4096,
			minCPU:               2.0,
			gsqGameSlug:          "rust",
			disabledCapabilities: `[]`,
		},
	}

	for _, g := range games {
		result, err := db.Exec(
			`INSERT OR IGNORE INTO games (id, name, image, default_ports, default_env, min_memory_mb, min_cpu, gsq_game_slug, disabled_capabilities) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			g.id, g.name, g.image, g.defaultPorts, g.defaultEnv, g.minMemoryMB, g.minCPU, g.gsqGameSlug, g.disabledCapabilities,
		)
		if err != nil {
			return fmt.Errorf("seeding game %s: %w", g.id, err)
		}

		rows, _ := result.RowsAffected()
		if rows > 0 {
			slog.Info("seeded game", "id", g.id, "name", g.name)
		} else {
			slog.Debug("game already exists, skipping seed", "id", g.id)
		}
	}

	return nil
}
