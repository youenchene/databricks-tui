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
	jobViewRunDetail
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

type jobRunsMsg struct {
	Runs []job.Run
	Err  error
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
	items      []JobItem
	cursor     int
	loaded     bool
	err        error
	filter     string
	filterMode bool
}

func NewJobListModel() JobListModel { return JobListModel{} }

func (m JobListModel) Init() tea.Cmd { return nil }

func (m JobListModel) filteredItems() []JobItem {
	if m.filter == "" {
		return m.items
	}
	q := strings.ToLower(m.filter)
	var out []JobItem
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Name), q) ||
			strings.Contains(fmt.Sprint(it.ID), q) ||
			strings.Contains(strings.ToLower(it.Schedule), q) {
			out = append(out, it)
		}
	}
	return out
}

func (m JobListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m JobListModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading jobs...")
	}
	if m.err != nil {
		return tea.NewView("Error loading jobs: " + m.err.Error())
	}

	s := ""
	if m.filterMode || m.filter != "" {
		s += filterBar(m.filter, m.filterMode, "jobs")
	}

	items := m.filteredItems()
	if len(items) == 0 {
		s += "No jobs found."
		return tea.NewView(s)
	}

	s += "Jobs (press enter to view details):\n\n"
	for i, item := range items {
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

func (m JobListModel) SelectedID() int64 {
	items := m.filteredItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return 0
	}
	return items[m.cursor].ID
}

// --- JobDetailModel ---

type JobDetailModel struct {
	detail   *job.JobDetail
	runs     []job.Run
	focusRuns bool // tab toggles between runs cursor and tasks cursor
	runCursor int
	taskCursor int
	loaded     bool
	err        error
	jobID      int64
}

func NewJobDetailModel(jobID int64) JobDetailModel {
	return JobDetailModel{jobID: jobID}
}

func (m JobDetailModel) Init() tea.Cmd { return nil }

func (m JobDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			m.focusRuns = !m.focusRuns
		case "up", "k":
			if m.focusRuns {
				if m.runCursor > 0 {
					m.runCursor--
				}
			} else {
				if m.taskCursor > 0 {
					m.taskCursor--
				}
			}
		case "down", "j":
			if m.focusRuns {
				if m.runCursor < len(m.runs)-1 {
					m.runCursor++
				}
			} else {
				if m.detail != nil && m.taskCursor < len(m.detail.Tasks)-1 {
					m.taskCursor++
				}
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
	highlight := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0A500"))
	d := m.detail

	s := title.Render("Job: "+d.Job.Name) + "\n\n"

	// metadata
	s += fmt.Sprintf("  ID:       %d\n", d.Job.ID)
	s += fmt.Sprintf("  Schedule: %s\n", scheduleOrManual(d.Job.Schedule))
	s += fmt.Sprintf("  Creator:  %s\n", d.Job.Creator)
	s += fmt.Sprintf("  Tasks:    %d\n", d.TaskCount())

	// --- runs section ---
	s += "\n"
	if m.focusRuns {
		s += highlight.Render("▶ Runs (tab to switch, enter to zoom):") + "\n"
	} else {
		s += title.Render("Runs (tab to focus):") + "\n"
	}
	if len(m.runs) == 0 {
		s += "  (no runs)\n"
	} else {
		for i, r := range m.runs {
			cursor := " "
			if m.focusRuns && m.runCursor == i {
				cursor = ">"
			}
			icon := stateIcon(r.State)
			when := "n/a"
			if !r.StartAt.IsZero() {
				when = r.StartAt.Format("2006-01-02 15:04")
			}
			s += fmt.Sprintf("  %s %s %s — %s\n", cursor, icon, when, string(r.State))
		}
	}

	// --- tasks section ---
	s += "\n"
	if !m.focusRuns {
		s += highlight.Render("▶ Tasks (enter for latest run & logs):") + "\n"
	} else {
		s += title.Render("Tasks (tab to focus):") + "\n"
	}
	if d.TaskCount() == 0 {
		s += "  (no tasks)\n"
	} else {
		for i, t := range d.Tasks {
			cursor := " "
			if !m.focusRuns && m.taskCursor == i {
				cursor = ">"
			}
			deps := ""
			if len(t.DependsOn) > 0 {
				deps = " ← [" + strings.Join(t.DependsOn, ",") + "]"
			}
			s += fmt.Sprintf("  %s %s | %s%s\n", cursor, t.TaskKey, t.TaskType(), deps)
		}
	}

	s += "\n[tab] switch runs/tasks  [enter] zoom  [esc] back to list"
	return tea.NewView(s)
}

func (m JobDetailModel) SelectedTaskKey() string {
	if m.detail == nil || m.taskCursor < 0 || m.taskCursor >= len(m.detail.Tasks) {
		return ""
	}
	return m.detail.Tasks[m.taskCursor].TaskKey
}

func (m JobDetailModel) SelectedRunID() int64 {
	if m.runCursor < 0 || m.runCursor >= len(m.runs) {
		return 0
	}
	return m.runs[m.runCursor].RunID
}

// --- RunDetailModel ---

type RunDetailModel struct {
	detail *job.RunDetail
	loaded bool
	err    error
	runID  int64
	cursor int // task cursor within the run
}

func NewRunDetailModel(runID int64) RunDetailModel {
	return RunDetailModel{runID: runID}
}

func (m RunDetailModel) Init() tea.Cmd { return nil }

func (m RunDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.detail != nil && m.cursor < len(m.detail.Tasks)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m RunDetailModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading run details...")
	}
	if m.err != nil {
		return tea.NewView("Error loading run: " + m.err.Error())
	}
	if m.detail == nil {
		return tea.NewView("No run data.")
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	highlight := lipgloss.NewStyle().Foreground(lipgloss.Color("#F0A500"))
	d := m.detail

	s := title.Render(fmt.Sprintf("Run #%d", d.Run.RunID)) + "\n\n"

	// metadata
	s += fmt.Sprintf("  Job ID:   %d\n", d.Run.JobID)
	s += fmt.Sprintf("  State:    %s\n", d.Run.State)
	s += fmt.Sprintf("  Start:    %s\n", timeOrNA(d.Run.StartAt))
	s += fmt.Sprintf("  End:      %s\n", timeOrNA(d.Run.EndAt))
	s += fmt.Sprintf("  Duration: %s\n", d.Duration())

	// stats
	succeeded, failed, running := d.TaskStates()
	s += "\n" + title.Render("Stats:") + "\n"
	s += fmt.Sprintf("  Tasks: %d total, ✓ %d succeeded, ✗ %d failed, ⏳ %d running\n",
		len(d.Tasks), succeeded, failed, running)

	// tasks in this run
	s += "\n" + highlight.Render("Tasks in this run (enter to zoom):") + "\n"
	if len(d.Tasks) == 0 {
		s += "  (no task executions)\n"
	} else {
		for i, t := range d.Tasks {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			icon := stateIcon(t.State)
			dur := t.RunDuration.Truncate(time.Second).String()
			s += fmt.Sprintf("  %s %s %s — %s (%s)\n", cursor, icon, t.TaskKey, string(t.State), dur)
		}
	}

	// logs
	s += "\n" + title.Render("Output:") + "\n"
	if d.Output.HasLogs() {
		lines := d.Output.FirstLines(15)
		for _, l := range lines {
			s += "  " + l + "\n"
		}
		if d.Output.LogTruncated {
			s += "  ... (truncated)\n"
		}
	} else if d.Output.Logs != "" {
		s += "  " + d.Output.Logs + "\n"
	} else {
		s += "  (no output)\n"
	}
	if d.Output.HasError() {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Render("Error: "+d.Output.ErrorMsg) + "\n"
		if d.Output.ErrorTrace != "" {
			s += "  " + d.Output.ErrorTrace + "\n"
		}
	}

	s += "\n[esc/backspace] back to job"
	return tea.NewView(s)
}

func (m RunDetailModel) SelectedTaskRunID() int64 {
	if m.detail == nil || m.cursor < 0 || m.cursor >= len(m.detail.Tasks) {
		return 0
	}
	return m.detail.Tasks[m.cursor].RunID
}

func (m RunDetailModel) SelectedTaskKey() string {
	if m.detail == nil || m.cursor < 0 || m.cursor >= len(m.detail.Tasks) {
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

func (m TaskDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m TaskDetailModel) View() tea.View {
	if !m.loaded {
		return tea.NewView("Loading task details...")
	}
	if m.err != nil {
		return tea.NewView("Error loading task: " + m.err.Error())
	}
	if m.detail == nil {
		return tea.NewView("No task data.")
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	d := m.detail

	s := title.Render(fmt.Sprintf("Task: %s → %s", m.jobName, m.taskKey)) + "\n\n"

	s += fmt.Sprintf("  Run ID:   %d\n", d.Run.RunID)
	s += fmt.Sprintf("  State:    %s\n", d.Run.State)
	s += fmt.Sprintf("  Start:    %s\n", timeOrNA(d.Run.StartAt))
	s += fmt.Sprintf("  End:      %s\n", timeOrNA(d.Run.EndAt))
	s += fmt.Sprintf("  Duration: %s\n", d.Duration())

	// task executions
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

	s += "\n" + title.Render("Output:") + "\n"
	if d.Output.HasLogs() {
		for _, l := range d.Output.FirstLines(20) {
			s += "  " + l + "\n"
		}
		if d.Output.LogTruncated {
			s += "  ... (truncated)\n"
		}
	} else if d.Output.Logs != "" {
		s += "  " + d.Output.Logs + "\n"
	} else {
		s += "  (no output)\n"
	}
	if d.Output.HasError() {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4672")).Render("Error: "+d.Output.ErrorMsg) + "\n"
	}

	s += "\n[esc/backspace] back"
	return tea.NewView(s)
}

// --- commands ---

func fetchJobsCmd(svc *job.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		jobs, err := svc.ListAll(ctx)
		if err != nil {
			slog.Error("fetch jobs failed", "error", err)
			return jobsMsg{Err: err}
		}
		items := make([]JobItem, len(jobs))
		for i, j := range jobs {
			items[i] = JobItem{ID: j.ID, Name: j.Name, Schedule: j.Schedule}
		}
		return jobsMsg{Items: items}
	}
}

func fetchJobDetailCmd(svc *job.Service, jobID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		jd, err := svc.GetDetail(ctx, jobID)
		if err != nil {
			slog.Error("fetch job detail failed", "jobID", jobID, "error", err)
			return jobDetailMsg{Err: err}
		}
		return jobDetailMsg{Detail: jd}
	}
}

func fetchJobRunsCmd(svc *job.Service, jobID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		runs, err := svc.RecentRuns(ctx, jobID, 10)
		if err != nil {
			slog.Error("fetch job runs failed", "jobID", jobID, "error", err)
			return jobRunsMsg{Err: err}
		}
		return jobRunsMsg{Runs: runs}
	}
}

func fetchRunDetailCmd(svc *job.Service, runID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
