package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workerTokensCmd = &cobra.Command{
	Use:   "worker-tokens",
	Short: "Manage worker auth tokens via API (requires running server)",
}

var workerTokensListCmd = &cobra.Command{
	Use:   "list",
	Short: "List worker tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiGet("/api/worker-tokens")
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var tokens []struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			CreatedAt string  `json:"created_at"`
			ExpiresAt *string `json:"expires_at"`
		}
		if err := json.Unmarshal(resp.Data, &tokens); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		if len(tokens) == 0 {
			fmt.Println("No worker tokens found.")
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tNAME\tCREATED")
		for _, t := range tokens {
			fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID[:8], t.Name, t.CreatedAt)
		}
		w.Flush()
		return nil
	},
}

var workerTokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a worker token",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return exitError(fmt.Errorf("--name is required"))
		}

		body := map[string]any{"name": name}

		resp, err := apiPost("/api/worker-tokens", body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var result struct {
			Token   string `json:"token"`
			TokenID string `json:"token_id"`
			Name    string `json:"name"`
		}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Worker token %q created (id: %s)\n", result.Name, result.TokenID)
		fmt.Println(result.Token)
		return nil
	},
}

var workerTokensDeleteCmd = &cobra.Command{
	Use:   "delete <token-id>",
	Short: "Delete a worker token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmAction(fmt.Sprintf("Delete worker token %s?", args[0])) {
			return nil
		}

		_, err := apiDelete("/api/worker-tokens/" + args[0])
		if err != nil {
			return exitError(err)
		}

		if !jsonOutput {
			fmt.Println("Worker token deleted.")
		}
		return nil
	},
}

func init() {
	workerTokensCreateCmd.Flags().String("name", "", "Token name (required)")

	workerTokensCmd.AddCommand(workerTokensListCmd, workerTokensCreateCmd, workerTokensDeleteCmd)
}
