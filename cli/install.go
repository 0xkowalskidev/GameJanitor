package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install as a system service",
	Long:  "Installs gamejanitor as a systemd service so it starts on boot and restarts on crash.",
	RunE:  runInstall,
}

func init() {
	installCmd.Flags().String("runtime", "auto", "Container runtime: docker, process, auto")
	installCmd.Flags().String("data-dir", "/var/lib/gamejanitor", "Data directory")
}

func runInstall(cmd *cobra.Command, args []string) error {
	if os.Getuid() != 0 {
		return exitError(fmt.Errorf("installing a systemd service requires root\n  Run: sudo gamejanitor install"))
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return exitError(fmt.Errorf("systemd not found — gamejanitor install requires systemd"))
	}

	runtimeFlag, _ := cmd.Flags().GetString("runtime")
	dataDir, _ := cmd.Flags().GetString("data-dir")

	return installSystemd(runtimeFlag, dataDir)
}

func installSystemd(runtime, dataDir string) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Build the ExecStart command with flags
	execStart := fmt.Sprintf("%s serve -d %s", binPath, dataDir)
	if runtime != "" && runtime != "auto" {
		execStart += " --runtime " + runtime
	}

	// Only depend on Docker if using container runtime
	afterUnits := "network-online.target"
	if runtime == "docker" || runtime == "auto" || runtime == "" {
		afterUnits += " docker.service"
	}

	unitContent := fmt.Sprintf(`[Unit]
Description=Gamejanitor - Game Server Manager
After=%s
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5

NoNewPrivileges=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, afterUnits, execStart, dataDir)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	unitPath := "/etc/systemd/system/gamejanitor.service"
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("writing service file: %w", err)
	}
	fmt.Printf("Service file written to %s\n", unitPath)

	for _, cmdArgs := range [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "gamejanitor"},
		{"systemctl", "start", "gamejanitor"},
	} {
		c := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("running %s: %w", strings.Join(cmdArgs, " "), err)
		}
	}

	fmt.Println("Gamejanitor installed and started.")
	fmt.Printf("  Data dir: %s\n", dataDir)
	fmt.Println("  Status:   systemctl status gamejanitor")
	fmt.Println("  Logs:     journalctl -u gamejanitor -f")
	fmt.Println("  Stop:     sudo systemctl stop gamejanitor")
	fmt.Println("  Remove:   sudo systemctl disable gamejanitor && sudo rm " + unitPath)
	return nil
}
