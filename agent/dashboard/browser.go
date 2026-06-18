package dashboard

import (
	"os"
	"os/exec"
	"runtime"
)

func OpenDashboard(port string) {
	url := "http://127.0.0.1:" + port + "/ui"
	openBrowser(url)
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

	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.Start()
}

func ShouldOpenDashboard() bool {
	return os.Getenv("LANLINK_OPEN_DASHBOARD") != "false"
}
