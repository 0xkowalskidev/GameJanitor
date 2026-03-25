package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhook endpoints",
}

func init() {
	webhooksCreateCmd.Flags().String("url", "", "Webhook URL (required)")
	webhooksCreateCmd.Flags().StringSlice("events", []string{"*"}, "Event patterns (comma-separated, default: all)")
	webhooksCreateCmd.Flags().String("secret", "", "HMAC-SHA256 signing secret")
	webhooksCreateCmd.Flags().String("description", "", "Description")
	webhooksCreateCmd.MarkFlagRequired("url")

	webhooksUpdateCmd.Flags().String("url", "", "Webhook URL")
	webhooksUpdateCmd.Flags().StringSlice("events", nil, "Event patterns")
	webhooksUpdateCmd.Flags().String("secret", "", "Signing secret")
	webhooksUpdateCmd.Flags().Bool("enabled", false, "Enable or disable")
	webhooksUpdateCmd.Flags().String("description", "", "Description")

	webhooksDeliveriesCmd.Flags().String("state", "", "Filter by state: pending, delivered, failed")
	webhooksDeliveriesCmd.Flags().Int("limit", 50, "Number of deliveries to show")

	webhooksCmd.AddCommand(webhooksListCmd, webhooksCreateCmd, webhooksUpdateCmd, webhooksDeleteCmd, webhooksTestCmd, webhooksDeliveriesCmd)
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiGet("/api/webhooks")
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var webhooks []struct {
			ID          string   `json:"id"`
			URL         string   `json:"url"`
			Description string   `json:"description"`
			Events      []string `json:"events"`
			Enabled     bool     `json:"enabled"`
		}
		if err := json.Unmarshal(resp.Data, &webhooks); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		if len(webhooks) == 0 {
			fmt.Println("No webhooks configured.")
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tURL\tEVENTS\tENABLED\tDESCRIPTION")
		for _, wh := range webhooks {
			enabled := "yes"
			if !wh.Enabled {
				enabled = "no"
			}
			events := strings.Join(wh.Events, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wh.ID[:8], wh.URL, events, enabled, wh.Description)
		}
		w.Flush()
		return nil
	},
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		events, _ := cmd.Flags().GetStringSlice("events")
		secret, _ := cmd.Flags().GetString("secret")
		description, _ := cmd.Flags().GetString("description")

		body := map[string]any{
			"url":         url,
			"events":      events,
			"description": description,
		}
		if secret != "" {
			body["secret"] = secret
		}

		resp, err := apiPost("/api/webhooks", body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var wh struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal(resp.Data, &wh); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		fmt.Printf("Webhook created: %s (%s)\n", wh.URL, wh.ID[:8])
		return nil
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{}

		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			body["url"] = v
		}
		if cmd.Flags().Changed("events") {
			v, _ := cmd.Flags().GetStringSlice("events")
			body["events"] = v
		}
		if cmd.Flags().Changed("secret") {
			v, _ := cmd.Flags().GetString("secret")
			body["secret"] = v
		}
		if cmd.Flags().Changed("enabled") {
			v, _ := cmd.Flags().GetBool("enabled")
			body["enabled"] = v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			body["description"] = v
		}

		if len(body) == 0 {
			return exitError(fmt.Errorf("no update flags specified"))
		}

		resp, err := apiPatch("/api/webhooks/"+args[0], body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Webhook %s updated.\n", args[0][:8])
		return nil
	},
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmAction(fmt.Sprintf("Delete webhook %s?", args[0][:8])) {
			fmt.Println("Aborted.")
			return nil
		}

		_, err := apiDelete("/api/webhooks/" + args[0])
		if err != nil {
			return exitError(err)
		}

		if !jsonOutput {
			fmt.Printf("Webhook %s deleted.\n", args[0][:8])
		}
		return nil
	},
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Send a test delivery to a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiPost("/api/webhooks/"+args[0]+"/test", nil)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Test delivery sent to webhook %s.\n", args[0][:8])
		return nil
	},
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries <id>",
	Short: "List deliveries for a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		params := fmt.Sprintf("?limit=%d", mustGetInt(cmd, "limit"))
		if v, _ := cmd.Flags().GetString("state"); v != "" {
			params += "&state=" + v
		}

		resp, err := apiGet("/api/webhooks/" + args[0] + "/deliveries" + params)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var deliveries []struct {
			ID         string `json:"id"`
			State      string `json:"state"`
			StatusCode int    `json:"status_code"`
			CreatedAt  string `json:"created_at"`
			EventType  string `json:"event_type"`
		}
		if err := json.Unmarshal(resp.Data, &deliveries); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		if len(deliveries) == 0 {
			fmt.Println("No deliveries found.")
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tEVENT\tSTATE\tHTTP\tTIME")
		for _, d := range deliveries {
			httpStatus := fmt.Sprintf("%d", d.StatusCode)
			if d.StatusCode == 0 {
				httpStatus = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.ID[:8], d.EventType, d.State, httpStatus, d.CreatedAt)
		}
		w.Flush()
		return nil
	},
}

func mustGetInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}
