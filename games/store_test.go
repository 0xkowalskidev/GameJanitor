package games

import (
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestGameStore_LoadsAllGames(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	games := store.ListGames()
	assert.GreaterOrEqual(t, len(games), 10, "should load at least 10 embedded games")
}

func TestGameStore_GetGame_ReturnsCorrectFields(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	game := store.GetGame("minecraft-java")
	require.NotNil(t, game, "minecraft-java should exist")
	assert.Equal(t, "minecraft-java", game.ID)
	assert.NotEmpty(t, game.Name)
	assert.NotEmpty(t, game.BaseImage)
	assert.NotEmpty(t, game.DefaultPorts)
}

func TestGameStore_GetGame_NotFound(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	game := store.GetGame("nonexistent-game")
	assert.Nil(t, game)
}

func TestGameStore_AllGames_HaveRequiredFields(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	for _, game := range store.ListGames() {
		t.Run(game.ID, func(t *testing.T) {
			assert.NotEmpty(t, game.ID, "game must have an ID")
			assert.NotEmpty(t, game.Name, "game must have a name")
			assert.NotEmpty(t, game.BaseImage, "game must have a base_image")
			assert.NotEmpty(t, game.DefaultPorts, "game must have at least one port")
		})
	}
}

func TestGameStore_AllGames_ReadyPatternsCompile(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	for _, game := range store.ListGames() {
		if game.ReadyPattern == "" {
			continue
		}
		t.Run(game.ID, func(t *testing.T) {
			_, err := regexp.Compile(game.ReadyPattern)
			assert.NoError(t, err, "ready_pattern should be a valid regex")
		})
	}
}

func TestGameStore_AllGames_NoDuplicatePortsWithinGame(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	for _, game := range store.ListGames() {
		t.Run(game.ID, func(t *testing.T) {
			type portKey struct {
				Port     int
				Protocol string
			}
			seen := make(map[portKey]bool)
			for _, p := range game.DefaultPorts {
				key := portKey{p.Port, p.Protocol}
				assert.False(t, seen[key], "duplicate port %d/%s in game %s", p.Port, p.Protocol, game.ID)
				seen[key] = true
			}
		})
	}
}

func TestGameStore_AllGames_ValidEnvTypes(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	validTypes := map[string]bool{"": true, "text": true, "number": true, "boolean": true, "select": true}

	for _, game := range store.ListGames() {
		for _, env := range game.DefaultEnv {
			t.Run(game.ID+"/"+env.Key, func(t *testing.T) {
				assert.True(t, validTypes[env.Type], "env var %s has invalid type %q", env.Key, env.Type)
			})
		}
	}
}

func TestGameStore_AllGames_SelectEnvHaveOptions(t *testing.T) {
	t.Parallel()
	store, err := NewGameStore("", testLogger())
	require.NoError(t, err)

	for _, game := range store.ListGames() {
		for _, env := range game.DefaultEnv {
			if env.Type != "select" {
				continue
			}
			t.Run(game.ID+"/"+env.Key, func(t *testing.T) {
				hasOptions := len(env.Options) > 0 || env.DynamicOptions != nil
				assert.True(t, hasOptions, "select env var %s must have options or dynamic_options", env.Key)
			})
		}
	}
}

func TestGameStore_LocalOverride(t *testing.T) {
	t.Parallel()

	// Create a temp dir with a custom game
	dir := t.TempDir()
	gameDir := dir + "/custom-game"
	os.MkdirAll(gameDir, 0755)
	os.WriteFile(gameDir+"/game.yaml", []byte(`
id: custom-game
name: "Custom Game"
base_image: alpine:latest
ports:
  - name: game
    port: 9999
    protocol: tcp
`), 0644)

	store, err := NewGameStore(dir, testLogger())
	require.NoError(t, err)

	game := store.GetGame("custom-game")
	require.NotNil(t, game, "custom game should be loaded")
	assert.Equal(t, "Custom Game", game.Name)
	assert.Equal(t, "alpine:latest", game.BaseImage)
}
