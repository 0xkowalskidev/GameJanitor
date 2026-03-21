package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var workersCmd = &cobra.Command{
	Use:     "workers",
	Aliases: []string{"w"},
	Short:   "Manage remote workers",
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
			ID                string   `json:"id"`
			LanIP             string   `json:"lan_ip"`
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
			status := wk.Status
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
			PortRangeStart    *int     `json:"port_range_start"`
			PortRangeEnd      *int     `json:"port_range_end"`
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
		status := wk.Status
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

		if wk.PortRangeStart != nil && wk.PortRangeEnd != nil {
			fmt.Fprintf(w, "Port Range:\t%d-%d\n", *wk.PortRangeStart, *wk.PortRangeEnd)
		} else {
			fmt.Fprintf(w, "Port Range:\tdefault\n")
		}
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

var workersSetPortRangeCmd = &cobra.Command{
	Use:   "set-port-range <worker-id>",
	Short: "Set a custom port range for a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start, _ := cmd.Flags().GetInt("start")
		end, _ := cmd.Flags().GetInt("end")
		if start == 0 || end == 0 {
			return exitError(fmt.Errorf("--start and --end are required"))
		}

		body := map[string]any{
			"port_range_start": start,
			"port_range_end":   end,
		}

		resp, err := apiPatch("/api/workers/"+args[0]+"/port-range", body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Port range set to %d-%d for worker %s.\n", start, end, args[0])
		return nil
	},
}

var workersClearPortRangeCmd = &cobra.Command{
	Use:   "clear-port-range <worker-id>",
	Short: "Clear custom port range (revert to global default)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := apiDelete("/api/workers/" + args[0] + "/port-range")
		if err != nil {
			return exitError(err)
		}
		if !jsonOutput {
			fmt.Printf("Port range cleared for worker %s.\n", args[0])
		}
		return nil
	},
}

var workersSetLimitsCmd = &cobra.Command{
	Use:   "set-limits <worker-id>",
	Short: "Set resource limits for a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body := make(map[string]any)

		if cmd.Flags().Changed("max-memory") {
			v, _ := cmd.Flags().GetInt("max-memory")
			body["max_memory_mb"] = v
		}
		if cmd.Flags().Changed("max-cpu") {
			v, _ := cmd.Flags().GetFloat64("max-cpu")
			body["max_cpu"] = v
		}
		if cmd.Flags().Changed("max-storage") {
			v, _ := cmd.Flags().GetInt("max-storage")
			body["max_storage_mb"] = v
		}

		if len(body) == 0 {
			return exitError(fmt.Errorf("at least one of --max-memory, --max-cpu, or --max-storage is required"))
		}

		resp, err := apiPatch("/api/workers/"+args[0]+"/limits", body)
		if err != nil {
			return exitError(err)
		}

		if jsonOutput {
			printJSONResponse(resp)
			return nil
		}

		fmt.Printf("Limits updated for worker %s.\n", args[0])
		return nil
	},
}

var workersClearLimitsCmd = &cobra.Command{
	Use:   "clear-limits <worker-id>",
	Short: "Remove resource limits from a worker",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := apiDelete("/api/workers/" + args[0] + "/limits")
		if err != nil {
			return exitError(err)
		}
		if !jsonOutput {
			fmt.Printf("Limits cleared for worker %s.\n", args[0])
		}
		return nil
	},
}

var workersCordonCmd = &cobra.Command{
	Use:   "cordon <worker-id>",
	Short: "Cordon a worker (prevent new gameserver placements)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiPost("/api/workers/"+args[0]+"/cordon", nil)
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
	Short: "Uncordon a worker (allow new gameserver placements)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := apiDelete("/api/workers/" + args[0] + "/cordon")
		if err != nil {
			return exitError(err)
		}
		if !jsonOutput {
			fmt.Printf("Worker %s uncordoned.\n", args[0])
		}
		return nil
	},
}

func init() {
	workersSetPortRangeCmd.Flags().Int("start", 0, "Port range start (required)")
	workersSetPortRangeCmd.Flags().Int("end", 0, "Port range end (required)")

	workersSetLimitsCmd.Flags().Int("max-memory", 0, "Max memory in MB (0 to clear)")
	workersSetLimitsCmd.Flags().Float64("max-cpu", 0, "Max CPU cores (0 to clear)")
	workersSetLimitsCmd.Flags().Int("max-storage", 0, "Max storage in MB (0 to clear)")

	workersCmd.AddCommand(
		workersListCmd, workersGetCmd,
		workersSetPortRangeCmd, workersClearPortRangeCmd,
		workersSetLimitsCmd, workersClearLimitsCmd,
		workersCordonCmd, workersUncordonCmd,
	)
}
