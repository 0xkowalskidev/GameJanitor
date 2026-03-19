package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/api/audit"
		params := url.Values{}
		if v, _ := cmd.Flags().GetString("action"); v != "" {
			params.Set("action", v)
		}
		if v, _ := cmd.Flags().GetString("resource-type"); v != "" {
			params.Set("resource_type", v)
		}
		if v, _ := cmd.Flags().GetString("resource-id"); v != "" {
			params.Set("resource_id", v)
		}
		if v, _ := cmd.Flags().GetString("token"); v != "" {
			params.Set("token_id", v)
		}
		if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
			params.Set("limit", strconv.Itoa(v))
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		resp, err := apiGet(path)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var entries []struct {
			ID           string `json:"id"`
			Timestamp    string `json:"timestamp"`
			Action       string `json:"action"`
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
			TokenID      string `json:"token_id"`
			TokenName    string `json:"token_name"`
			IPAddress    string `json:"ip_address"`
			StatusCode   int    `json:"status_code"`
		}
		if err := json.Unmarshal(resp.Data, &entries); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		w := newTabWriter()
		fmt.Fprintln(w, "TIMESTAMP\tACTION\tRESOURCE\tTOKEN\tIP\tSTATUS")
		for _, e := range entries {
			resourceID := e.ResourceID
			if len(resourceID) > 8 {
				resourceID = resourceID[:8]
			}
			tokenName := e.TokenName
			if tokenName == "" && e.TokenID != "" {
				tokenName = e.TokenID[:min(8, len(e.TokenID))]
			}
			if tokenName == "" {
				tokenName = "-"
			}
			ts := e.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}
			resource := e.ResourceType
			if resourceID != "" {
				resource += "/" + resourceID
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n", ts, e.Action, resource, tokenName, e.IPAddress, e.StatusCode)
		}
		w.Flush()
		return nil
	},
}

func init() {
	auditListCmd.Flags().String("action", "", "Filter by action (e.g. gameserver.create)")
	auditListCmd.Flags().String("resource-type", "", "Filter by resource type (e.g. gameserver)")
	auditListCmd.Flags().String("resource-id", "", "Filter by resource ID")
	auditListCmd.Flags().String("token", "", "Filter by token ID")
	auditListCmd.Flags().Int("limit", 25, "Max entries to return")

	auditCmd.AddCommand(auditListCmd)
}
