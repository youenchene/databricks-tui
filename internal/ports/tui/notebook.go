package tui

import (
	"context"
	"log/slog"
	"strings"
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
	items      []NotebookItem
	cursor     int
	loaded     bool
	err        error
	filter     string
	filterMode bool
}

// NewNotebookListModel creates an empty notebook list.
func NewNotebookListModel() NotebookListModel {
	return NotebookListModel{}
}

func (m NotebookListModel) Init() tea.Cmd { return nil }

func (m NotebookListModel) filteredItems() []NotebookItem {
	if m.filter == "" {
		return m.items
	}
	q := strings.ToLower(m.filter)
	var out []NotebookItem
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Path), q) ||
			strings.Contains(strings.ToLower(it.Language), q) {
			out = append(out, it)
		}
	}
	return out
}

// Update handles navigation and filter.
func (m NotebookListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the notebook list or error.
func (m NotebookListModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading notebooks...")
	}
	if m.err != nil {
		return tea.NewView("Error loading notebooks: " + m.err.Error())
	}

	s := ""
	if m.filterMode || m.filter != "" {
		s += filterBar(m.filter, m.filterMode, "notebooks")
	}

	items := m.filteredItems()
	if len(items) == 0 {
		s += "No notebooks found."
		return tea.NewView(s)
	}

	s += "Notebooks:\n\n"
	for i, item := range items {
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
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
