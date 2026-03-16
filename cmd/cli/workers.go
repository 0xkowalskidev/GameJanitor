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
			ID                string `json:"id"`
			LanIP             string `json:"lan_ip"`
			CPUCores          int64  `json:"cpu_cores"`
			MemoryTotalMB     int64  `json:"memory_total_mb"`
			MemoryAvailableMB int64  `json:"memory_available_mb"`
			GameserverCount   int    `json:"gameserver_count"`
			AllocatedMemoryMB int    `json:"allocated_memory_mb"`
			MaxMemoryMB       *int   `json:"max_memory_mb"`
			MaxGameservers    *int   `json:"max_gameservers"`
			Status            string `json:"status"`
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
			memory := fmt.Sprintf("%d/%d MB", wk.MemoryAvailableMB, wk.MemoryTotalMB)

			gs := fmt.Sprintf("%d", wk.GameserverCount)
			if wk.MaxGameservers != nil {
				gs = fmt.Sprintf("%d/%d", wk.GameserverCount, *wk.MaxGameservers)
			}

			fmt.Fprintf(w, "%s\t%s\t%d cores\t%s\t%s\t%s\n",
				wk.ID, wk.LanIP, wk.CPUCores, memory, gs, wk.Status)
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
			ID                string `json:"id"`
			LanIP             string `json:"lan_ip"`
			ExternalIP        string `json:"external_ip"`
			CPUCores          int64  `json:"cpu_cores"`
			MemoryTotalMB     int64  `json:"memory_total_mb"`
			MemoryAvailableMB int64  `json:"memory_available_mb"`
			GameserverCount   int    `json:"gameserver_count"`
			AllocatedMemoryMB int    `json:"allocated_memory_mb"`
			PortRangeStart    *int   `json:"port_range_start"`
			PortRangeEnd      *int   `json:"port_range_end"`
			MaxMemoryMB       *int   `json:"max_memory_mb"`
			MaxGameservers    *int   `json:"max_gameservers"`
			Status            string `json:"status"`
			LastSeen          string `json:"last_seen"`
		}
		if err := json.Unmarshal(resp.Data, &wk); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}

		w := newTabWriter()
		fmt.Fprintf(w, "ID:\t%s\n", wk.ID)
		fmt.Fprintf(w, "Status:\t%s\n", wk.Status)
		fmt.Fprintf(w, "LAN IP:\t%s\n", wk.LanIP)
		if wk.ExternalIP != "" {
			fmt.Fprintf(w, "External IP:\t%s\n", wk.ExternalIP)
		}
		fmt.Fprintf(w, "CPU:\t%d cores\n", wk.CPUCores)
		fmt.Fprintf(w, "Memory:\t%d / %d MB available\n", wk.MemoryAvailableMB, wk.MemoryTotalMB)
		fmt.Fprintf(w, "Gameservers:\t%d\n", wk.GameserverCount)
		fmt.Fprintf(w, "Allocated Memory:\t%d MB\n", wk.AllocatedMemoryMB)

		if wk.PortRangeStart != nil && wk.PortRangeEnd != nil {
			fmt.Fprintf(w, "Port Range:\t%d-%d\n", *wk.PortRangeStart, *wk.PortRangeEnd)
		} else {
			fmt.Fprintf(w, "Port Range:\tdefault\n")
		}
		if wk.MaxGameservers != nil {
			fmt.Fprintf(w, "Max Gameservers:\t%d\n", *wk.MaxGameservers)
		}
		if wk.MaxMemoryMB != nil {
			fmt.Fprintf(w, "Max Memory:\t%d MB\n", *wk.MaxMemoryMB)
		}
		fmt.Fprintf(w, "Last Seen:\t%s\n", wk.LastSeen)
		w.Flush()
		return nil
	},
}

func init() {
	workersCmd.AddCommand(workersListCmd, workersGetCmd)
}
