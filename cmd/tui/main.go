// Package main wires dependencies and starts the TUI.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/youenchene/databricks-tui/internal/adapters/databricks"
	"github.com/youenchene/databricks-tui/internal/domain/cluster"
	"github.com/youenchene/databricks-tui/internal/domain/job"
	"github.com/youenchene/databricks-tui/internal/domain/notebook"
	"github.com/youenchene/databricks-tui/internal/ports/tui"
)

var version = "dev"

func main() {
	// setup structured logging to file (stderr messes up the TUI)
	logDir := filepath.Join(os.TempDir(), "databricks-tui")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "tui.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})))
		defer logFile.Close()
	}

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
	app := tui.NewAppModel(clusterSvc, jobSvc, notebookSvc, profile, version)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
