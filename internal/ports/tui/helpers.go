package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// wrapLogLines wraps log lines to fit within the terminal width.
// Each line receives a "  " prefix during rendering, so the effective width is winWidth - 2.
// If winWidth is too small (< 20), lines are returned unchanged.
func wrapLogLines(lines []string, winWidth int) []string {
	wrapWidth := winWidth - 2
	if wrapWidth < 20 {
		return lines
	}
	var wrapped []string
	for _, line := range lines {
		if len(line) <= wrapWidth {
			wrapped = append(wrapped, line)
			continue
		}
		for len(line) > wrapWidth {
			wrapped = append(wrapped, line[:wrapWidth])
			line = line[wrapWidth:]
		}
		wrapped = append(wrapped, line)
	}
	return wrapped
}

// stateColor returns a lipgloss style colored by cluster/job state.
func stateColor(state string) lipgloss.Style {
	switch state {
	case "RUNNING", "SUCCEEDED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	case "PENDING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F0A500"))
	case "FAILED", "ERROR", "TERMINATING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672"))
	case "TERMINATED", "CANCELED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	default:
		return lipgloss.NewStyle()
	}
}

// filterBar renders the search/filter input bar.
func filterBar(filter string, active bool, label string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#3C3C3C")).
		Padding(0, 1)

	prompt := "/"
	if !active && filter != "" {
		prompt = ""
	}
	cursor := ""
	if active {
		cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render("│")
	}
	return style.Render(fmt.Sprintf("search %s: %s%s%s", label, prompt, filter, cursor)) + "\n\n"
}
