package cli

import (
	"encoding/json"
	"fmt"

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
		resp, err := apiGet("/api/games")
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var games []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Image string `json:"image"`
		}
		if err := json.Unmarshal(resp.Data, &games); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tNAME\tIMAGE")
		for _, g := range games {
			fmt.Fprintf(w, "%s\t%s\t%s\n", g.ID, g.Name, g.Image)
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
		resp, err := apiGet("/api/games/" + args[0])
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var game struct {
			ID                  string `json:"id"`
			Name                string `json:"name"`
			Image               string `json:"image"`
			RecommendedMemoryMB int    `json:"recommended_memory_mb"`
		}
		if err := json.Unmarshal(resp.Data, &game); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		fmt.Printf("ID:                  %s\n", game.ID)
		fmt.Printf("Name:                %s\n", game.Name)
		fmt.Printf("Image:               %s\n", game.Image)
		fmt.Printf("Recommended Memory:  %s\n", formatMemory(game.RecommendedMemoryMB))
		return nil
	},
}
