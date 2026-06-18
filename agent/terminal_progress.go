package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/123100123/lanlink/agent/dashboard"
)

var previouslyActive = make(map[string]bool)

func startTerminalProgress() {
	var lastPrinted time.Time

	for {
		time.Sleep(500 * time.Millisecond)

		now := time.Now()
		if now.Sub(lastPrinted) < time.Second {
			continue
		}
		lastPrinted = now

		active := dashboard.GetActiveTransfers()
		completed := dashboard.GetRecentlyCompleted()

		currentActive := make(map[string]bool)
		for _, t := range active {
			currentActive[t.ID] = true
			renderTerminalTransfer(t)
		}

		for id := range previouslyActive {
			if !currentActive[id] {
				for _, t := range completed {
					if t.ID == id {
						fmt.Fprintf(os.Stderr, "\nSaved: %s -> %s\n", t.Filename, t.Path)
						break
					}
				}
			}
		}

		previouslyActive = currentActive
	}
}

func renderTerminalTransfer(t dashboard.Transfer) {
	name := t.Filename
	if len(name) > 30 {
		name = name[:27] + "..."
	}

	if t.Total <= 0 {
		speedStr := ""
		if t.Speed > 0 {
			speedStr = " " + formatSpeed(t.Speed)
		}
		fmt.Fprintf(os.Stderr,
			"\rReceiving: %s  %s received%s        ",
			name, formatBytes(t.Received), speedStr,
		)
		return
	}

	percent := float64(t.Received) / float64(t.Total) * 100
	bar := progressBar(percent, 20)

	speedStr := ""
	if t.Speed > 0 {
		speedStr = " " + formatSpeed(t.Speed)
	}

	etaStr := ""
	if t.Speed > 0 && t.Received < t.Total {
		remaining := float64(t.Total - t.Received)
		etaSec := remaining / float64(t.Speed)
		etaStr = " ETA " + formatETA(etaSec)
	}

	fmt.Fprintf(os.Stderr,
		"\rReceiving: %s [%s] %5.1f%%  %s / %s%s%s        ",
		name, bar, percent,
		formatBytes(t.Received), formatBytes(t.Total),
		speedStr, etaStr,
	)
}

func progressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	value := float64(b)
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", b)
	}
	return fmt.Sprintf("%.1f %s", value, units[i])
}

func formatSpeed(bps int64) string {
	if bps <= 0 {
		return ""
	}
	mbps := float64(bps) / 1024 / 1024
	return fmt.Sprintf("%.1f MB/s", mbps)
}

func formatETA(seconds float64) string {
	if seconds < 1 {
		return "<1s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", int(seconds))
	}
	mins := int(seconds) / 60
	secs := int(seconds) % 60
	return fmt.Sprintf("%dm%ds", mins, secs)
}
