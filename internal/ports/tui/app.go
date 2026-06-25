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
	screenJobList = iota
	screenClusterList
	screenNotebookList
)

// AppModel is the root Bubble Tea model for the application.
type AppModel struct {
	width  int
	height int

	currentScreen int

	// job navigation
	jobView        int // jobViewList, jobViewDetail, jobViewRunDetail, jobViewTaskDetail
	jobDetail      JobDetailModel
	runDetail      RunDetailModel
	taskDetail     TaskDetailModel
	taskOrigin     int    // jobViewDetail or jobViewRunDetail — where we came from before task detail
	pendingTaskKey string // task key waiting for run detail to resolve

	// domain services
	clusterSvc  *cluster.Service
	jobSvc      *job.Service
	notebookSvc *notebook.Service

	// child models
	clusterList  ClusterListModel
	jobList      JobListModel
	notebookList NotebookListModel

	userProfile string
	version     string
	ready       bool
}

// NewAppModel creates the root application model.
func NewAppModel(
	clusterSvc *cluster.Service,
	jobSvc *job.Service,
	notebookSvc *notebook.Service,
	profile string,
	version string,
) AppModel {
	return AppModel{
		currentScreen: screenJobList,
		jobView:       jobViewList,
		clusterSvc:    clusterSvc,
		jobSvc:        jobSvc,
		notebookSvc:   notebookSvc,
		clusterList:   NewClusterListModel(),
		jobList:       NewJobListModel(),
		notebookList:  NewNotebookListModel(),
		userProfile:   profile,
		version:       version,
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
		// quit
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		// tab switching via Code (rune) — reliable across terminals
		switch msg.Code {
		case '1':
			m.currentScreen = screenJobList
			m.jobView = jobViewList
		case '2':
			m.currentScreen = screenClusterList
		case '3':
			m.currentScreen = screenNotebookList
		}

		// navigation keys
		switch msg.String() {
		case "enter":
			if m.currentScreen == screenJobList {
				switch m.jobView {
				case jobViewList:
					id := m.jobList.SelectedID()
					if id > 0 {
						m.jobDetail = NewJobDetailModel(id)
						m.jobView = jobViewDetail
						// fetch job detail + recent runs
						return m, tea.Batch(
							fetchJobDetailCmd(m.jobSvc, id),
							fetchJobRunsCmd(m.jobSvc, id),
						)
					}
				case jobViewDetail:
					if m.jobDetail.focusRuns {
						// enter on a run → open run detail
						id := m.jobDetail.SelectedRunID()
						if id > 0 {
							m.runDetail = NewRunDetailModel(id, m.width, m.height-2)
							m.jobView = jobViewRunDetail
							return m, fetchRunDetailCmd(m.jobSvc, id)
						}
					} else {
						// enter on a task → fetch latest run, then get task-specific output
						if m.jobDetail.detail != nil {
							key := m.jobDetail.SelectedTaskKey()
							runID := m.jobDetail.SelectedRunID()
							if key != "" && runID > 0 {
								m.pendingTaskKey = key
								m.taskDetail = NewTaskDetailModel(m.jobDetail.detail.Job.Name, key, m.width, m.height-2)
								m.taskOrigin = jobViewDetail
								m.jobView = jobViewTaskDetail
								return m, fetchRunDetailCmd(m.jobSvc, runID)
							}
						}
					}
				case jobViewRunDetail:
					// enter on a task in run detail → fetch task-specific output
					key := m.runDetail.SelectedTaskKey()
					taskRunID := m.runDetail.SelectedTaskRunID()
					if key != "" && m.runDetail.detail != nil {
					m.taskDetail = NewTaskDetailModel(
						fmt.Sprintf("run %d", m.runDetail.detail.Run.RunID), key, m.width, m.height-2)
						m.taskOrigin = jobViewRunDetail
						m.jobView = jobViewTaskDetail
						// fetch task-specific output
						return m, fetchRunDetailCmd(m.jobSvc, taskRunID)
					}
				}
			}

		case "esc", "backspace":
			if m.currentScreen == screenJobList {
				switch m.jobView {
				case jobViewDetail:
					m.jobView = jobViewList
				case jobViewRunDetail:
					m.jobView = jobViewDetail
				case jobViewTaskDetail:
					m.jobView = m.taskOrigin
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

	case refreshJobDetailMsg:
		id := m.jobDetail.jobID
		return m, tea.Batch(
			fetchJobDetailCmd(m.jobSvc, id),
			fetchJobRunsCmd(m.jobSvc, id),
		)

	case jobDetailMsg:
		m.jobDetail.loaded = true
		m.jobDetail.detail = msg.Detail
		m.jobDetail.err = msg.Err
		return m, nil

	case jobRunsMsg:
		m.jobDetail.runs = msg.Runs
		if msg.Err == nil && len(msg.Runs) > 0 {
			return m, fetchJobLatestRunTasksCmd(m.jobSvc, msg.Runs[0].RunID)
		}
		m.jobDetail.refreshing = false
		return m, nil

	case jobRunDetailMsg:
		// route to correct target based on current view
		if m.jobView == jobViewRunDetail {
			m.runDetail.loaded = true
			m.runDetail.detail = msg.Detail
			m.runDetail.err = msg.Err
		} else if m.pendingTaskKey != "" && msg.Detail != nil {
			// coming from jobDetail task selection — find task run ID and fetch its output
			for _, t := range msg.Detail.Tasks {
				if t.TaskKey == m.pendingTaskKey && t.RunID > 0 {
					m.pendingTaskKey = ""
					return m, fetchRunDetailCmd(m.jobSvc, t.RunID)
				}
			}
			// task not found in this run — show what we have
			m.taskDetail.loaded = true
			m.taskDetail.detail = msg.Detail
			m.taskDetail.err = msg.Err
			m.pendingTaskKey = ""
		} else {
			m.taskDetail.loaded = true
			m.taskDetail.detail = msg.Detail
			m.taskDetail.err = msg.Err
		}
		return m, nil

	case jobLatestRunTasksMsg:
		m.jobDetail.refreshing = false
		if msg.Err == nil {
			m.jobDetail.taskStatuses = msg.Tasks
		}
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
		case jobViewRunDetail:
			newModel, listCmd := m.runDetail.Update(msg)
			m.runDetail = newModel.(RunDetailModel)
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

	header := headerView(m.currentScreen, m.userProfile, m.version, m.width)
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
		case jobViewRunDetail:
			content = m.runDetail.View().Content
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

func headerView(screen int, profile string, version string, width int) string {
	tabs := map[int]string{
		screenJobList:      "[1] Jobs",
		screenClusterList:  "[2] Clusters",
		screenNotebookList: "[3] Notebooks",
	}

	base := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))

	// left side
	left := base.Render(" Databricks TUI v" + version)

	// center (tabs)
	center := base.Render(" " + tabs[screen] + " ")

	// right side (shortcuts)
	profileLabel := ""
	if profile != "" && profile != "DEFAULT" {
		profileLabel = " profile:" + profile + " "
	}
	right := base.Render(profileLabel + "[/] search  [q] quit ")

	// assemble bar
	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		left,
		lipgloss.NewStyle().Background(lipgloss.Color("#7D56F4")).Width(1).Render(""),
		center,
	)
	// push right side to end
	rightWidth := lipgloss.Width(right)
	spacer := lipgloss.NewStyle().Background(lipgloss.Color("#7D56F4")).Width(width - lipgloss.Width(bar) - rightWidth).Render(" ")
	bar += spacer + right

	return bar
}

func footerView(width int) string {
	help := "[/] search  [f] toggle fav  [F] favs only  [↑/↓] navigate  [enter] select  [esc] back  [q] quit"
	style := lipgloss.NewStyle().
		Width(width).
		Foreground(lipgloss.Color("#626262"))

	return style.Render(help)
}
