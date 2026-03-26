package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var workersCmd = &cobra.Command{
	Use:     "workers",
	Aliases: []string{"w"},
	Short:   "Manage worker nodes",
}

func init() {
	workersSetCmd.Flags().Int("memory", 0, "Max memory in MB (0 to clear)")
	workersSetCmd.Flags().Float64("cpu", 0, "Max CPU cores (0 to clear)")
	workersSetCmd.Flags().Int("storage", 0, "Max storage in MB (0 to clear)")
	workersSetCmd.Flags().StringSlice("tags", nil, "Worker labels (key=value, comma-separated)")
	workersSetCmd.Flags().Int("port-range-start", 0, "Port range start (0 to clear)")
	workersSetCmd.Flags().Int("port-range-end", 0, "Port range end (0 to clear)")

	workersClearCmd.Flags().Bool("limits", false, "Clear all resource limits")
	workersClearCmd.Flags().Bool("tags", false, "Clear all tags")

	workersCmd.AddCommand(workersListCmd, workersGetCmd, workersSetCmd, workersClearCmd, workersCordonCmd, workersUncordonCmd)
}

var workersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiGet("/api/workers")
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var workers []struct {
			ID                string  `json:"id"`
			LanIP             string  `json:"lan_ip"`
			CPUCores          int64   `json:"cpu_cores"`
			MemoryTotalMB     int64   `json:"memory_total_mb"`
			MemoryAvailableMB int64   `json:"memory_available_mb"`
			GameserverCount   int     `json:"gameserver_count"`
			Cordoned          bool    `json:"cordoned"`
			Status            string  `json:"status"`
		}
		if err := json.Unmarshal(resp.Data, &workers); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		if len(workers) == 0 {
			fmt.Println("No workers connected.")
			return nil
		}

		w := newTabWriter()
		fmt.Fprintln(w, "ID\tLAN IP\tCPU\tMEMORY\tGAMESERVERS\tSTATUS")
		for _, wk := range workers {
			memory := fmt.Sprintf("%s / %s", formatMemory(int(wk.MemoryAvailableMB)), formatMemory(int(wk.MemoryTotalMB)))
			status := colorStatus(wk.Status)
			if wk.Cordoned {
				status += " (cordoned)"
			}
			fmt.Fprintf(w, "%s\t%s\t%d cores\t%s\t%d\t%s\n",
				wk.ID, wk.LanIP, wk.CPUCores, memory, wk.GameserverCount, status)
		}
		w.Flush()
		return nil
	},
}

var workersGetCmd = &cobra.Command{
	Use:   "get <worker-id>",
	Short: "Get details for a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiGet("/api/workers/" + args[0])
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		var wk struct {
			ID                string   `json:"id"`
			LanIP             string   `json:"lan_ip"`
			ExternalIP        string   `json:"external_ip"`
			CPUCores          int64    `json:"cpu_cores"`
			MemoryTotalMB     int64    `json:"memory_total_mb"`
			MemoryAvailableMB int64    `json:"memory_available_mb"`
			GameserverCount   int      `json:"gameserver_count"`
			AllocatedMemoryMB int      `json:"allocated_memory_mb"`
			AllocatedCPU      float64  `json:"allocated_cpu"`
			MaxMemoryMB       *int     `json:"max_memory_mb"`
			MaxCPU            *float64 `json:"max_cpu"`
			MaxStorageMB      *int     `json:"max_storage_mb"`
			Cordoned          bool     `json:"cordoned"`
			Status            string   `json:"status"`
			LastSeen          string   `json:"last_seen"`
		}
		if err := json.Unmarshal(resp.Data, &wk); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		w := newTabWriter()
		fmt.Fprintf(w, "ID:\t%s\n", wk.ID)
		status := colorStatus(wk.Status)
		if wk.Cordoned {
			status += " (cordoned)"
		}
		fmt.Fprintf(w, "Status:\t%s\n", status)
		fmt.Fprintf(w, "LAN IP:\t%s\n", wk.LanIP)
		if wk.ExternalIP != "" {
			fmt.Fprintf(w, "External IP:\t%s\n", wk.ExternalIP)
		}
		fmt.Fprintf(w, "CPU:\t%d cores\n", wk.CPUCores)
		fmt.Fprintf(w, "Memory:\t%s / %s available\n", formatMemory(int(wk.MemoryAvailableMB)), formatMemory(int(wk.MemoryTotalMB)))
		fmt.Fprintf(w, "Gameservers:\t%d\n", wk.GameserverCount)
		fmt.Fprintf(w, "Allocated Memory:\t%s\n", formatMemory(wk.AllocatedMemoryMB))
		fmt.Fprintf(w, "Allocated CPU:\t%.1f\n", wk.AllocatedCPU)
		if wk.MaxMemoryMB != nil {
			fmt.Fprintf(w, "Max Memory:\t%s\n", formatMemory(*wk.MaxMemoryMB))
		}
		if wk.MaxCPU != nil {
			fmt.Fprintf(w, "Max CPU:\t%.1f\n", *wk.MaxCPU)
		}
		if wk.MaxStorageMB != nil {
			fmt.Fprintf(w, "Max Storage:\t%s\n", formatMemory(*wk.MaxStorageMB))
		}
		fmt.Fprintf(w, "Last Seen:\t%s\n", wk.LastSeen)
		w.Flush()
		return nil
	},
}

var workersSetCmd = &cobra.Command{
	Use:   "set <worker-id>",
	Short: "Configure worker limits and tags",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := make(map[string]any)

		if cmd.Flags().Changed("memory") {
			v, _ := cmd.Flags().GetInt("memory")
			if v == 0 {
				body["max_memory_mb"] = nil
			} else {
				body["max_memory_mb"] = v
			}
		}
		if cmd.Flags().Changed("cpu") {
			v, _ := cmd.Flags().GetFloat64("cpu")
			if v == 0 {
				body["max_cpu"] = nil
			} else {
				body["max_cpu"] = v
			}
		}
		if cmd.Flags().Changed("storage") {
			v, _ := cmd.Flags().GetInt("storage")
			if v == 0 {
				body["max_storage_mb"] = nil
			} else {
				body["max_storage_mb"] = v
			}
		}
		if cmd.Flags().Changed("tags") {
			v, _ := cmd.Flags().GetStringSlice("tags")
			tags := make(map[string]string, len(v))
			for _, entry := range v {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) != 2 {
					return exitError(fmt.Errorf("invalid tag %q: must be key=value", entry))
				}
				tags[parts[0]] = parts[1]
			}
			body["tags"] = tags
		}
		if cmd.Flags().Changed("port-range-start") {
			v, _ := cmd.Flags().GetInt("port-range-start")
			if v == 0 {
				body["port_range_start"] = nil
			} else {
				body["port_range_start"] = v
			}
		}
		if cmd.Flags().Changed("port-range-end") {
			v, _ := cmd.Flags().GetInt("port-range-end")
			if v == 0 {
				body["port_range_end"] = nil
			} else {
				body["port_range_end"] = v
			}
		}

		if len(body) == 0 {
			return exitError(fmt.Errorf("at least one flag is required"))
		}

		resp, err := apiPatch("/api/workers/"+args[0], body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Worker %s updated.\n", args[0])
		return nil
	},
}

var workersClearCmd = &cobra.Command{
	Use:   "clear <worker-id>",
	Short: "Clear worker limits or tags",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := make(map[string]any)

		clearLimits, _ := cmd.Flags().GetBool("limits")
		clearTags, _ := cmd.Flags().GetBool("tags")

		if !clearLimits && !clearTags {
			return exitError(fmt.Errorf("specify --limits and/or --tags"))
		}

		if clearLimits {
			body["max_memory_mb"] = nil
			body["max_cpu"] = nil
			body["max_storage_mb"] = nil
		}
		if clearTags {
			body["tags"] = map[string]string{}
		}

		resp, err := apiPatch("/api/workers/"+args[0], body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Worker %s cleared.\n", args[0])
		return nil
	},
}

var workersCordonCmd = &cobra.Command{
	Use:   "cordon <worker-id>",
	Short: "Prevent new gameserver placement on a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{"cordoned": true}
		resp, err := apiPatch("/api/workers/"+args[0], body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Worker %s cordoned.\n", args[0])
		return nil
	},
}

var workersUncordonCmd = &cobra.Command{
	Use:   "uncordon <worker-id>",
	Short: "Allow gameserver placement on a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := map[string]any{"cordoned": false}
		resp, err := apiPatch("/api/workers/"+args[0], body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Worker %s uncordoned.\n", args[0])
		return nil
	},
}
