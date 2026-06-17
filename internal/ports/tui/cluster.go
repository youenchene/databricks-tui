package tui

import (
	"context"
	"log/slog"
	"time"
	tea "charm.land/bubbletea/v2"

	"github.com/youenchene/databricks-tui/internal/domain/cluster"
)

// clustersMsg carries the fetched cluster list or an error.
type clustersMsg struct {
	Items []ClusterItem
	Err   error
}

// ClusterItem is a display-friendly wrapper around domain.Cluster.
type ClusterItem struct {
	ID           string
	Name         string
	State        string
	SparkVersion string
	NodeTypeID   string
}

// ClusterListModel manages the cluster list view.
type ClusterListModel struct {
	items  []ClusterItem
	cursor int
	loaded bool
	err    error
}

// NewClusterListModel creates an empty cluster list.
func NewClusterListModel() ClusterListModel {
	return ClusterListModel{}
}

func (m ClusterListModel) Init() tea.Cmd { return nil }

// Update handles key navigation.
func (m ClusterListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// View renders the cluster list or error.
func (m ClusterListModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading clusters...")
	}
	if m.err != nil {
		return tea.NewView("Error loading clusters: " + m.err.Error())
	}
	if len(m.items) == 0 {
		return tea.NewView("No clusters found.")
	}

	s := "Clusters:\n\n"
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		style := stateColor(item.State)
		s += style.Render(cursor+" "+item.ID+" | "+item.Name+" | "+item.State) + "\n"
	}
	return tea.NewView(s)
}

// fetchClustersCmd returns a command that calls the cluster service.
func fetchClustersCmd(svc *cluster.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // paginated list
		defer cancel()
		clusters, err := svc.ListAll(ctx)
		if err != nil {
			slog.Error("fetch clusters failed", "error", err)
			return clustersMsg{Err: err}
		}
		items := make([]ClusterItem, len(clusters))
		for i, c := range clusters {
			items[i] = ClusterItem{
				ID:           c.ID,
				Name:         c.Name,
				State:        string(c.State),
				SparkVersion: c.SparkVersion,
				NodeTypeID:   c.NodeTypeID,
			}
		}
		return clustersMsg{Items: items}
	}
}
