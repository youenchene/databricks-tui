package tui

import "github.com/charmbracelet/lipgloss"

// stateColor returns a lipgloss style colored by cluster/job state.
func stateColor(state string) lipgloss.Style {
	switch state {
	case "RUNNING", "SUCCEEDED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")) // green
	case "PENDING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#F0A500")) // amber
	case "FAILED", "ERROR", "TERMINATING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")) // red
	case "TERMINATED", "CANCELED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")) // gray
	default:
		return lipgloss.NewStyle()
	}
}
