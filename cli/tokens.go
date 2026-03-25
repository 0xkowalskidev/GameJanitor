package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage auth tokens",
}

func init() {
	tokensCreateCmd.Flags().String("name", "", "Token name (required)")
	tokensCreateCmd.Flags().String("scope", "custom", "Token scope: admin or custom")
	tokensCreateCmd.Flags().StringSlice("gameserver", nil, "Scope to gameserver (repeatable, name or ID)")
	tokensCreateCmd.Flags().StringSlice("permission", nil, "Permission (repeatable: start, stop, restart, logs, commands, files, backups, configure)")
	tokensCreateCmd.Flags().String("expires-in", "", "Expiry duration (e.g. 720h, 30d)")

	tokensCmd.AddCommand(tokensListCmd, tokensCreateCmd, tokensDeleteCmd)
}

var tokensListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiGet("/api/tokens")
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var tokens []struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			Scope      string  `json:"scope"`
			CreatedAt  string  `json:"created_at"`
			LastUsedAt *string `json:"last_used_at"`
			ExpiresAt  *string `json:"expires_at"`
		}
		if err := json.Unmarshal(resp.Data, &tokens); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		if len(tokens) == 0 {
			fmt.Println("No tokens found.")
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tNAME\tSCOPE\tCREATED\tLAST USED\tEXPIRES")
		for _, t := range tokens {
			lastUsed := "-"
			if t.LastUsedAt != nil {
				lastUsed = *t.LastUsedAt
			}
			expires := "never"
			if t.ExpiresAt != nil {
				expires = *t.ExpiresAt
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				t.ID[:8], t.Name, t.Scope, t.CreatedAt, lastUsed, expires)
		}
		w.Flush()
		return nil
	},
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a token",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return exitError(fmt.Errorf("--name is required"))
		}

		scope, _ := cmd.Flags().GetString("scope")
		gameserverNames, _ := cmd.Flags().GetStringSlice("gameserver")
		permissions, _ := cmd.Flags().GetStringSlice("permission")
		expiresIn, _ := cmd.Flags().GetString("expires-in")

		// Resolve gameserver names to IDs
		var gameserverIDs []string
		for _, gs := range gameserverNames {
			id, err := resolveGameserverID(gs)
			if err != nil {
				return exitError(fmt.Errorf("resolving gameserver %q: %w", gs, err))
			}
			gameserverIDs = append(gameserverIDs, id)
		}

		body := map[string]any{
			"name":           name,
			"scope":          scope,
			"gameserver_ids": gameserverIDs,
			"permissions":    permissions,
		}
		if expiresIn != "" {
			body["expires_in"] = expiresIn
		}

		resp, err := apiPost("/api/tokens", body)
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

		fmt.Fprintf(os.Stderr, "Token %q created (id: %s)\n", result.Name, result.TokenID)
		if len(gameserverIDs) > 0 {
			fmt.Fprintf(os.Stderr, "Scoped to %d gameserver(s), permissions: %s\n", len(gameserverIDs), strings.Join(permissions, ", "))
		}
		fmt.Fprintf(os.Stderr, "Store this token — it cannot be retrieved later.\n")
		// Raw token to stdout for piping
		fmt.Println(result.Token)
		return nil
	},
}

var tokensDeleteCmd = &cobra.Command{
	Use:   "delete <token-id>",
	Short: "Delete a token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmAction(fmt.Sprintf("Delete token %s?", args[0])) {
			fmt.Println("Aborted.")
			return nil
		}

		_, err := apiDelete("/api/tokens/" + args[0])
		if err != nil {
			return exitError(err)
		}

		if !jsonOutput {
			fmt.Println("Token deleted.")
		}
		return nil
	},
}
