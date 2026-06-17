package tui

import (
	"context"
	"log/slog"
	"strings"
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
	items      []ClusterItem
	cursor     int
	loaded     bool
	err        error
	filter     string
	filterMode bool
}

// NewClusterListModel creates an empty cluster list.
func NewClusterListModel() ClusterListModel {
	return ClusterListModel{}
}

func (m ClusterListModel) Init() tea.Cmd { return nil }

// filteredItems returns items matching the current filter (case-insensitive).
func (m ClusterListModel) filteredItems() []ClusterItem {
	if m.filter == "" {
		return m.items
	}
	q := strings.ToLower(m.filter)
	var out []ClusterItem
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Name), q) ||
			strings.Contains(strings.ToLower(it.ID), q) ||
			strings.Contains(strings.ToLower(it.State), q) {
			out = append(out, it)
		}
	}
	return out
}

// Update handles key navigation and filter.
func (m ClusterListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.filterMode {
			switch msg.String() {
			case "esc":
				m.filterMode = false
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.cursor = 0
				}
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				items := m.filteredItems()
				if m.cursor < len(items)-1 {
					m.cursor++
				}
			default:
				s := msg.String()
				if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
					m.filter += s
					m.cursor = 0
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.filterMode = true
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			items := m.filteredItems()
			if m.cursor < len(items)-1 {
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

	s := ""
	if m.filterMode || m.filter != "" {
		s += filterBar(m.filter, m.filterMode, "clusters")
	}

	items := m.filteredItems()
	if len(items) == 0 {
		s += "No clusters found."
		return tea.NewView(s)
	}

	s += "Clusters:\n\n"
	for i, item := range items {
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
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
