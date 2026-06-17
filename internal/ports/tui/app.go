// Package tui provides the Bubble Tea terminal UI.
//
// This is the presentation layer (port) of the hexagonal architecture.
// Views delegate to domain services, never to adapters directly.
package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/youenchene/databricks-tui/internal/domain/cluster"
	"github.com/youenchene/databricks-tui/internal/domain/job"
	"github.com/youenchene/databricks-tui/internal/domain/notebook"
)

// top-level screens
const (
	screenClusterList = iota
	screenJobList
	screenNotebookList
)

// AppModel is the root Bubble Tea model for the application.
type AppModel struct {
	width  int
	height int

	currentScreen int

	// job navigation
	jobView    int // jobViewList, jobViewDetail, jobViewTaskDetail
	jobDetail  JobDetailModel
	taskDetail TaskDetailModel

	// domain services
	clusterSvc  *cluster.Service
	jobSvc      *job.Service
	notebookSvc *notebook.Service

	// child models
	clusterList  ClusterListModel
	jobList      JobListModel
	notebookList NotebookListModel

	userProfile string
	ready       bool
}

// NewAppModel creates the root application model.
func NewAppModel(
	clusterSvc *cluster.Service,
	jobSvc *job.Service,
	notebookSvc *notebook.Service,
	profile string,
) AppModel {
	return AppModel{
		currentScreen: screenClusterList,
		jobView:       jobViewList,
		clusterSvc:    clusterSvc,
		jobSvc:        jobSvc,
		notebookSvc:   notebookSvc,
		clusterList:   NewClusterListModel(),
		jobList:       NewJobListModel(),
		notebookList:  NewNotebookListModel(),
		userProfile:   profile,
	}
}

// Init returns the initial command: fetch clusters and jobs.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		fetchClustersCmd(m.clusterSvc),
		fetchJobsCmd(m.jobSvc),
	)
}

// Update handles messages and delegates to child views.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.currentScreen = screenClusterList
		case "2":
			m.currentScreen = screenJobList
			m.jobView = jobViewList
		case "3":
			m.currentScreen = screenNotebookList

		// job navigation
		case "enter":
			if m.currentScreen == screenJobList {
				switch m.jobView {
				case jobViewList:
					id := m.jobList.SelectedID()
					if id > 0 {
						m.jobDetail = NewJobDetailModel(id)
						m.jobView = jobViewDetail
						return m, fetchJobDetailCmd(m.jobSvc, id)
					}
				case jobViewDetail:
					if m.jobDetail.detail != nil {
						key := m.jobDetail.SelectedTaskKey()
						if key != "" {
							m.taskDetail = NewTaskDetailModel(m.jobDetail.detail.Job.Name, key)
							m.jobView = jobViewTaskDetail
							return m, fetchRunDetailCmd(m.jobSvc, 0) // TODO: real run ID
						}
					}
				}
			}

		case "esc", "backspace":
			if m.currentScreen == screenJobList {
				switch m.jobView {
				case jobViewDetail:
					m.jobView = jobViewList
				case jobViewTaskDetail:
					m.jobView = jobViewDetail
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// cluster messages
	case clustersMsg:
		m.clusterList.loaded = true
		m.ready = true
		m.clusterList.items = msg.Items
		m.clusterList.err = msg.Err
		return m, nil

	// job messages
	case jobsMsg:
		m.jobList.loaded = true
		m.ready = true
		m.jobList.items = msg.Items
		m.jobList.err = msg.Err
		return m, nil

	case jobDetailMsg:
		m.jobDetail.loaded = true
		m.jobDetail.detail = msg.Detail
		m.jobDetail.err = msg.Err
		return m, nil

	case jobRunDetailMsg:
		m.taskDetail.loaded = true
		m.taskDetail.detail = msg.Detail
		m.taskDetail.err = msg.Err
		return m, nil

	// notebook messages
	case notebooksMsg:
		m.notebookList.loaded = true
		m.notebookList.items = msg.Items
		m.notebookList.err = msg.Err
		return m, nil
	}

	// Delegate to current screen
	switch m.currentScreen {
	case screenClusterList:
		newModel, listCmd := m.clusterList.Update(msg)
		m.clusterList = newModel.(ClusterListModel)
		cmd = listCmd
	case screenJobList:
		switch m.jobView {
		case jobViewList:
			newModel, listCmd := m.jobList.Update(msg)
			m.jobList = newModel.(JobListModel)
			cmd = listCmd
		case jobViewDetail:
			newModel, listCmd := m.jobDetail.Update(msg)
			m.jobDetail = newModel.(JobDetailModel)
			cmd = listCmd
		case jobViewTaskDetail:
			newModel, listCmd := m.taskDetail.Update(msg)
			m.taskDetail = newModel.(TaskDetailModel)
			cmd = listCmd
		}
	case screenNotebookList:
		newModel, listCmd := m.notebookList.Update(msg)
		m.notebookList = newModel.(NotebookListModel)
		cmd = listCmd
	}

	return m, cmd
}

// View renders the full UI.
func (m AppModel) View() tea.View {
	if !m.ready {
		return tea.NewView("Loading...")
	}

	header := headerView(m.currentScreen, m.userProfile)
	footer := footerView(m.width)
	content := ""

	switch m.currentScreen {
	case screenClusterList:
		content = m.clusterList.View().Content
	case screenJobList:
		switch m.jobView {
		case jobViewList:
			content = m.jobList.View().Content
		case jobViewDetail:
			content = m.jobDetail.View().Content
		case jobViewTaskDetail:
			content = m.taskDetail.View().Content
		}
	case screenNotebookList:
		content = m.notebookList.View().Content
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Top,
		header,
		content,
		footer,
	))
	v.AltScreen = true
	return v
}

// --- header / footer ---

func headerView(screen int, profile string) string {
	tabs := map[int]string{
		screenClusterList:  "[1] Clusters",
		screenJobList:      "[2] Jobs",
		screenNotebookList: "[3] Notebooks",
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	profileLabel := ""
	if profile != "" && profile != "DEFAULT" {
		profileLabel = fmt.Sprintf("  profile: %s", profile)
	}

	return style.Render(" databricks-tui " + tabs[screen] + profileLabel + " ")
}

func footerView(width int) string {
	help := "[1/2/3] switch view  [↑/↓] navigate  [enter] select  [esc] back  [q] quit"
	style := lipgloss.NewStyle().
		Width(width).
		Foreground(lipgloss.Color("#626262"))

	return style.Render(help)
}
