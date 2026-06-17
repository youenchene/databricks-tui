package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"github.com/youenchene/databricks-tui/internal/domain/job"
)

// --- navigation ---

const (
	jobViewList = iota
	jobViewDetail
	jobViewTaskDetail
)

// --- messages ---

type jobsMsg struct {
	Items []JobItem
	Err   error
}

type jobDetailMsg struct {
	Detail *job.JobDetail
	Err    error
}

type jobRunDetailMsg struct {
	Detail *job.RunDetail
	Err    error
}

// --- items ---

type JobItem struct {
	ID       int64
	Name     string
	Schedule string
}

// --- JobListModel ---

type JobListModel struct {
	items  []JobItem
	cursor int
	loaded bool
	err    error
}

func NewJobListModel() JobListModel {
	return JobListModel{}
}

func (m JobListModel) Init() tea.Cmd { return nil }

func (m JobListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m JobListModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading jobs...")
	}
	if m.err != nil {
		return tea.NewView("Error loading jobs: " + m.err.Error())
	}
	if len(m.items) == 0 {
		return tea.NewView("No jobs found.")
	}

	s := "Jobs (press enter to view details):\n\n"
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		sched := "manual"
		if item.Schedule != "" {
			sched = item.Schedule
		}
		s += cursor + " " + item.Name + " | " + sched + "\n"
	}
	return tea.NewView(s)
}

// SelectedID returns the ID of the currently highlighted job.
func (m JobListModel) SelectedID() int64 {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return 0
	}
	return m.items[m.cursor].ID
}

// --- JobDetailModel ---

type JobDetailModel struct {
	detail *job.JobDetail
	runs   []job.Run
	cursor int
	loaded bool
	err    error
	jobID  int64
}

func NewJobDetailModel(jobID int64) JobDetailModel {
	return JobDetailModel{jobID: jobID}
}

func (m JobDetailModel) Init() tea.Cmd { return nil }

func (m JobDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.detail.Tasks)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m JobDetailModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading job details...")
	}
	if m.err != nil {
		return tea.NewView("Error loading job: " + m.err.Error())
	}
	if m.detail == nil {
		return tea.NewView("No job data.")
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	d := m.detail

	s := title.Render("Job: "+d.Job.Name) + "\n\n"

	// metadata
	s += fmt.Sprintf("  ID:       %d\n", d.Job.ID)
	s += fmt.Sprintf("  Schedule: %s\n", scheduleOrManual(d.Job.Schedule))
	s += fmt.Sprintf("  Creator:  %s\n", d.Job.Creator)
	s += fmt.Sprintf("  Tasks:    %d\n", d.TaskCount())

	// recent runs
	if len(m.runs) > 0 {
		s += "\n" + title.Render("Recent Runs:") + "\n"
		for _, r := range m.runs {
			icon := stateIcon(r.State)
			when := "n/a"
			if !r.StartAt.IsZero() {
				when = r.StartAt.Format("2006-01-02 15:04")
			}
			s += fmt.Sprintf("  %s %s — %s\n", icon, when, string(r.State))
		}
	}

	// tasks list
	s += "\n" + title.Render("Tasks (↓↑ / enter):") + "\n"
	if d.TaskCount() == 0 {
		s += "  (no tasks)\n"
	} else {
		for i, t := range d.Tasks {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			deps := ""
			if len(t.DependsOn) > 0 {
				deps = " ← [" + strings.Join(t.DependsOn, ",") + "]"
			}
			s += fmt.Sprintf("  %s %s | %s%s\n", cursor, t.TaskKey, t.TaskType(), deps)
		}
	}

	s += "\n[esc/backspace] back to list"
	return tea.NewView(s)
}

// SelectedTaskKey returns the task key at cursor.
func (m JobDetailModel) SelectedTaskKey() string {
	if m.cursor < 0 || m.cursor >= len(m.detail.Tasks) {
		return ""
	}
	return m.detail.Tasks[m.cursor].TaskKey
}

// --- TaskDetailModel ---

type TaskDetailModel struct {
	jobName string
	taskKey string
	detail  *job.RunDetail
	loaded  bool
	err     error
}

func NewTaskDetailModel(jobName, taskKey string) TaskDetailModel {
	return TaskDetailModel{jobName: jobName, taskKey: taskKey}
}

func (m TaskDetailModel) Init() tea.Cmd { return nil }

func (m TaskDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m TaskDetailModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading task details...")
	}
	if m.err != nil {
		return tea.NewView("Error loading run: " + m.err.Error())
	}
	if m.detail == nil {
		return tea.NewView("No task data.")
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	d := m.detail

	s := title.Render(fmt.Sprintf("Task: %s → %s", m.jobName, m.taskKey)) + "\n\n"

	// run metadata
	s += fmt.Sprintf("  Run ID:   %d\n", d.Run.RunID)
	s += fmt.Sprintf("  State:    %s\n", d.Run.State)
	s += fmt.Sprintf("  Start:    %s\n", timeOrNA(d.Run.StartAt))
	s += fmt.Sprintf("  End:      %s\n", timeOrNA(d.Run.EndAt))
	s += fmt.Sprintf("  Duration: %s\n", d.Duration())

	// task executions in this run
	s += "\n" + title.Render("Task Executions:") + "\n"
	succeeded, failed, running := d.TaskStates()
	s += fmt.Sprintf("  ✓ %d  ✗ %d  ⏳ %d\n\n", succeeded, failed, running)

	for _, t := range d.Tasks {
		icon := stateIcon(t.State)
		s += fmt.Sprintf("  %s %s — %s", icon, t.TaskKey, string(t.State))
		if t.TaskKey == m.taskKey {
			s += " ◀"
		}
		s += "\n"
	}

	// logs/output
	if d.Output.HasLogs() {
		s += "\n" + title.Render("Logs:") + "\n"
		lines := d.Output.FirstLines(20)
		for _, l := range lines {
			s += "  " + l + "\n"
		}
	}
	if d.Output.HasError() {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Render("Error: "+d.Output.ErrorMsg) + "\n"
	}

	s += "\n[esc/backspace] back to job"
	return tea.NewView(s)
}

// --- commands ---

func fetchJobsCmd(svc *job.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // paginated list
		defer cancel()
		jobs, err := svc.ListAll(ctx)
		if err != nil {
			slog.Error("fetch jobs failed", "error", err)
			return jobsMsg{Err: err}
		}
		items := make([]JobItem, len(jobs))
		for i, j := range jobs {
			items[i] = JobItem{
				ID:       j.ID,
				Name:     j.Name,
				Schedule: j.Schedule,
			}
		}
		return jobsMsg{Items: items}
	}
}

func fetchJobDetailCmd(svc *job.Service, jobID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // paginated list
		defer cancel()
		jd, err := svc.GetDetail(ctx, jobID)
		if err != nil {
			slog.Error("fetch job detail failed", "jobID", jobID, "error", err)
			return jobDetailMsg{Err: err}
		}
		return jobDetailMsg{Detail: jd}
	}
}

func fetchRunDetailCmd(svc *job.Service, runID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // paginated list
		defer cancel()
		rd, err := svc.GetRunDetail(ctx, runID)
		if err != nil {
			slog.Error("fetch run detail failed", "runID", runID, "error", err)
			return jobRunDetailMsg{Err: err}
		}
		return jobRunDetailMsg{Detail: rd}
	}
}

// --- helpers ---

func scheduleOrManual(s string) string {
	if s == "" {
		return "manual"
	}
	return s
}

func stateIcon(s job.State) string {
	switch s {
	case job.StateSucceeded:
		return "✓"
	case job.StateFailed:
		return "✗"
	case job.StateRunning, job.StatePending:
		return "⏳"
	case job.StateCanceled:
		return "⊘"
	default:
		return "?"
	}
}

func timeOrNA(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.Format("2006-01-02 15:04:05")
}
