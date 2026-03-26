package cli

import (
	"fmt"
	"io/fs"
	"os/exec"
	"runtime"
	"strings"

	"github.com/warsmite/gamejanitor/config"
	"github.com/warsmite/gamejanitor/ui"
)

func isLoopback(addr string) bool {
	return addr == "127.0.0.1" || addr == "::1" || addr == "localhost"
}

// listenError wraps a listen error with a user-friendly message when the port is already in use.
func listenError(service, addr string, port int, err error) error {
	if strings.Contains(err.Error(), "address already in use") {
		return fmt.Errorf("%s server failed to start: port %d is already in use — another instance of gamejanitor or another program is using this port", service, port)
	}
	return fmt.Errorf("%s server failed to start on %s: %w", service, addr, err)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}

func webUIFS(cfg config.Config) fs.FS {
	if !cfg.WebUI {
		return nil
	}
	sub, err := fs.Sub(ui.Dist, "dist")
	if err != nil {
		return nil
	}
	return sub
}
