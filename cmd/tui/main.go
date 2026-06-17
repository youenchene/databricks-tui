// Package main wires dependencies and starts the TUI.
package main

import (
	"fmt"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/youenchene/databricks-tui/internal/adapters/databricks"
	"github.com/youenchene/databricks-tui/internal/domain/cluster"
	"github.com/youenchene/databricks-tui/internal/domain/job"
	"github.com/youenchene/databricks-tui/internal/domain/notebook"
	"github.com/youenchene/databricks-tui/internal/ports/tui"
)

func main() {
	// setup structured logging (Bubble Tea captures stdout, slog goes to stderr)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	profile := "DEFAULT"
	if len(os.Args) > 1 {
		profile = os.Args[1]
	}

	// Initialize adapter
	client, err := databricks.NewClient(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Databricks client: %v\n", err)
		os.Exit(1)
	}

	// Wire dependencies (manual DI)
	clusterRepo := databricks.NewClusterRepo(client)
	clusterSvc := cluster.NewService(clusterRepo)

	jobRepo := databricks.NewJobRepo(client)
	jobSvc := job.NewService(jobRepo)

	notebookRepo := databricks.NewNotebookRepo(client)
	notebookSvc := notebook.NewService(notebookRepo)

	// Launch TUI
	app := tui.NewAppModel(clusterSvc, jobSvc, notebookSvc, profile)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
