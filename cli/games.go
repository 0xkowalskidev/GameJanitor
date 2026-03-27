package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/warsmite/gamejanitor/config"
	"github.com/warsmite/gamejanitor/games"
	"github.com/spf13/cobra"
)

var gamesCmd = &cobra.Command{
	Use:   "games",
	Short: "List available games",
}

func init() {
	gamesCmd.AddCommand(gamesListCmd, gamesGetCmd)
}

var gamesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all games",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try API first, fall back to local game store if server isn't running
		gameList, err := getClient().Games.List(ctx())
		if err == nil {
			if jsonOutput {
				printJSON(gameList)
				return nil
			}
			w := newTabWriter()
			fmt.Fprintln(w, "ID\tNAME\tALIASES")
			for _, g := range gameList {
				aliases := ""
				if len(g.Aliases) > 0 {
					for i, a := range g.Aliases {
						if i > 0 { aliases += ", " }
						aliases += a
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", g.ID, g.Name, aliases)
			}
			w.Flush()
			return nil
		}

		// Fallback: load embedded game store locally
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		dataDir := config.DefaultConfig().DataDir
		store, storeErr := games.NewGameStore(dataDir+"/games", log)
		if storeErr != nil {
			// Return original API error if local fallback also fails
			return exitError(err)
		}

		allGames := store.ListGames()
		if jsonOutput {
			printJSON(allGames)
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tNAME\tALIASES")
		for _, g := range allGames {
			aliases := ""
			for i, a := range g.Aliases {
				if i > 0 { aliases += ", " }
				aliases += a
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", g.ID, g.Name, aliases)
		}
		w.Flush()
		return nil
	},
}

var gamesGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a game by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		game, err := getClient().Games.Get(ctx(), args[0])
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSON(game)
			return nil
		}

		fmt.Printf("ID:                  %s\n", game.ID)
		fmt.Printf("Name:                %s\n", game.Name)
		fmt.Printf("Image:               %s\n", game.BaseImage)
		fmt.Printf("Recommended Memory:  %s\n", formatMemory(game.RecommendedMemoryMB))
		return nil
	},
}
