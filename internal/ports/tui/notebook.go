package tui

import (
	"context"
	"log/slog"
	"time"
	tea "charm.land/bubbletea/v2"

	"github.com/youenchene/databricks-tui/internal/domain/notebook"
)

// notebooksMsg carries the fetched notebook list or an error.
type notebooksMsg struct {
	Items []NotebookItem
	Err   error
}

// NotebookItem is a display-friendly wrapper around domain.Entry.
type NotebookItem struct {
	Path     string
	IsDir    bool
	Language string
}

// NotebookListModel manages the notebook browser view.
type NotebookListModel struct {
	items  []NotebookItem
	cursor int
	loaded bool
	err    error
}

// NewNotebookListModel creates an empty notebook list.
func NewNotebookListModel() NotebookListModel {
	return NotebookListModel{}
}

func (m NotebookListModel) Init() tea.Cmd { return nil }

// Update handles navigation.
func (m NotebookListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the notebook list or error.
func (m NotebookListModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading notebooks...")
	}
	if m.err != nil {
		return tea.NewView("Error loading notebooks: " + m.err.Error())
	}
	if len(m.items) == 0 {
		return tea.NewView("No notebooks found.")
	}

	s := "Notebooks:\n\n"
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		prefix := "[FILE]"
		if item.IsDir {
			prefix = "[DIR] "
		}
		s += cursor + " " + prefix + " " + item.Path + "\n"
	}
	return tea.NewView(s)
}

// fetchNotebooksCmd returns a command that calls the notebook service.
func fetchNotebooksCmd(svc *notebook.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // paginated list
		defer cancel()
		entries, err := svc.ListPath(ctx, "/")
		if err != nil {
			slog.Error("fetch notebooks failed", "error", err)
			return notebooksMsg{Err: err}
		}
		items := make([]NotebookItem, len(entries))
		for i, e := range entries {
			items[i] = NotebookItem{
				Path:     e.Path,
				IsDir:    e.IsDir,
				Language: string(e.Language),
			}
		}
		return notebooksMsg{Items: items}
	}
}
