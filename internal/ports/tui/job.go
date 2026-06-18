package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

type clipboardMsg struct {
	ok  bool
	err error
}

// --- items ---

type JobItem struct {
	ID       int64
	Name     string
	Schedule string
}

// --- JobListModel ---

type JobListModel struct {
	items       []JobItem
	cursor      int
	loaded      bool
	err         error
	filter      string
	filterMode  bool
	favorites   map[int64]bool // job ID → is favorite
	favsOnly    bool           // show only favorites
}

func NewJobListModel() JobListModel {
	favs := loadFavorites()
	return JobListModel{
		favorites: favs,
		favsOnly:  len(favs) > 0, // default to favorites if any exist
	}
}

func (m JobListModel) Init() tea.Cmd { return nil }

func (m JobListModel) filteredItems() []JobItem {
	items := m.items
	if m.favsOnly {
		var favs []JobItem
		for _, it := range items {
			if m.favorites[it.ID] {
				favs = append(favs, it)
			}
		}
		items = favs
	}
	if m.filter == "" {
		return items
	}
	q := strings.ToLower(m.filter)
	var out []JobItem
	for _, it := range items {
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
		case "f":
			id := m.SelectedID()
			if id > 0 {
				m.favorites[id] = !m.favorites[id]
				go saveFavorites(m.favorites)
			}
			return m, nil
		case "F":
			m.favsOnly = !m.favsOnly
			m.cursor = 0
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
		fav := " "
		if m.favorites[item.ID] {
			fav = "★"
		}
		sched := "manual"
		if item.Schedule != "" {
			sched = item.Schedule
		}
		s += cursor + fav + " " + item.Name + " | " + sched + "\n"
	}
	if m.favsOnly {
		s += "\n  ★ favorites only (F to show all)"
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

// --- favorites persistence ---

func favPath() string {
	dir, _ := os.UserHomeDir()
	return filepath.Join(dir, ".databricks-tui", "favorites.json")
}

func loadFavorites() map[int64]bool {
	f := map[int64]bool{}
	data, err := os.ReadFile(favPath())
	if err != nil {
		return f
	}
	json.Unmarshal(data, &f)
	return f
}

func saveFavorites(favs map[int64]bool) {
	dir := filepath.Dir(favPath())
	os.MkdirAll(dir, 0755)
	data, _ := json.Marshal(favs)
	os.WriteFile(favPath(), data, 0644)
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
	detail    *job.RunDetail
	loaded    bool
	err       error
	runID     int64
	cursor    int // task cursor within the run
	logFocus  bool
	logOffset int
	copied    bool
	winWidth  int
	winHeight int
}

func NewRunDetailModel(runID int64, winWidth, winHeight int) RunDetailModel {
	return RunDetailModel{runID: runID, winWidth: winWidth, winHeight: winHeight}
}

func (m RunDetailModel) Init() tea.Cmd { return nil }

func (m RunDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clipboardMsg:
		m.copied = msg.ok && msg.err == nil

	case tea.WindowSizeMsg:
		m.winWidth = msg.Width
		m.winHeight = msg.Height - 2 // header + footer

	case tea.KeyPressMsg:
		m.copied = false // dismiss copy confirmation on any key

		switch msg.String() {
		case "y":
			canCopy := m.logFocus || (m.detail != nil && len(m.detail.Tasks) == 0)
			if canCopy && m.detail != nil && (m.detail.Output.Logs != "" || m.detail.Output.HasError()) {
				return m, copyLogsCmd(buildClipboardContent(m.detail.Output))
			}
		case "tab":
			if m.detail != nil && len(m.detail.Tasks) > 0 {
				m.logFocus = !m.logFocus
			}
		case "up", "k":
			if m.logFocus {
				if m.logOffset > 0 {
					m.logOffset--
				}
			} else if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.logFocus {
				m.logOffset++
			} else if m.detail != nil && m.cursor < len(m.detail.Tasks)-1 {
				m.cursor++
			}
		case "pgup":
			if m.logFocus {
				m.logOffset -= 10
				if m.logOffset < 0 {
					m.logOffset = 0
				}
			}
		case "pgdown":
			if m.logFocus {
				m.logOffset += 10
			}
		case "home":
			if m.logFocus {
				m.logOffset = 0
			}
		case "end":
			if m.logFocus {
				m.logOffset = 999999 // clamped in View()
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

	// logs — combined with errors into a single scrollable area
	if d.Output.HasLogs() || d.Output.Logs != "" || d.Output.HasError() {
		s += "\n" + title.Render("Output:") + "\n"

		// Build combined content: logs + errors
		var content strings.Builder
		if d.Output.HasLogs() {
			content.WriteString(d.Output.Logs)
			if d.Output.LogTruncated {
				content.WriteString("\n  ... (truncated by API)")
			}
		} else if d.Output.Logs != "" {
			content.WriteString(d.Output.Logs)
		}
		if d.Output.HasError() {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString("[ERROR] ")
			content.WriteString(d.Output.ErrorMsg)
			if d.Output.ErrorTrace != "" {
				content.WriteString("\n")
				content.WriteString(d.Output.ErrorTrace)
			}
		}
		rawLines := strings.Split(content.String(), "\n")
		logLines := wrapLogLines(rawLines, m.winWidth)

		// Auto-focus logs when there are no tasks to navigate
		noTasks := len(d.Tasks) == 0
		if noTasks {
			m.logFocus = true
		}

		// Calculate viewport height: use at least half the window
		linesBefore := strings.Count(s, "\n")
		afterLines := 1 // only the help line after logs
		vpHeight := m.winHeight - linesBefore - afterLines
		minVp := m.winHeight / 2
		if vpHeight < minVp {
			vpHeight = minVp
		}
		// But never overflow the window
		if linesBefore+vpHeight+afterLines > m.winHeight {
			vpHeight = m.winHeight - linesBefore - afterLines
		}
		if vpHeight < 8 {
			vpHeight = 8
		}

		// Clamp offset
		maxOffset := len(logLines) - vpHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.logOffset > maxOffset {
			m.logOffset = maxOffset
		}
		if m.logOffset < 0 {
			m.logOffset = 0
		}

		end := m.logOffset + vpHeight
		if end > len(logLines) {
			end = len(logLines)
		}

		// Focus hint
		if m.copied {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#4ECB71")).Render("  ✓ Copied to clipboard!") + "\n"
		} else if noTasks {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("  [y] copy all  [↑↓/pgup/pgdn] scroll") + "\n"
		} else if m.logFocus {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#F0A500")).Render("  ▶ scroll mode [tab] tasks  [y] copy all") + "\n"
		} else {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("  [tab] focus logs  [y] copy all") + "\n"
		}

		for i := m.logOffset; i < end; i++ {
			s += "  " + logLines[i] + "\n"
		}

		if len(logLines) > vpHeight {
			pos := fmt.Sprintf("── %d/%d ──", end, len(logLines))
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(pos) + "\n"
		}
	} else {
		s += "\n" + title.Render("Output:") + "\n"
		s += "  (no output)\n"
	}

	s += "\n[esc/backspace] back to job  [y] copy all logs"
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
	jobName   string
	taskKey   string
	detail    *job.RunDetail
	loaded    bool
	err       error
	logOffset int
	copied    bool
	winWidth  int
	winHeight int
}

func NewTaskDetailModel(jobName, taskKey string, winWidth, winHeight int) TaskDetailModel {
	return TaskDetailModel{
		jobName:   jobName,
		taskKey:   taskKey,
		winWidth:  winWidth,
		winHeight: winHeight,
	}
}

func (m TaskDetailModel) Init() tea.Cmd { return nil }

func (m TaskDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clipboardMsg:
		m.copied = msg.ok && msg.err == nil

	case tea.WindowSizeMsg:
		m.winWidth = msg.Width
		m.winHeight = msg.Height - 2 // header + footer

	case tea.KeyPressMsg:
		m.copied = false // dismiss copy confirmation on any key

		switch msg.String() {
		case "y":
			if m.detail != nil && (m.detail.Output.Logs != "" || m.detail.Output.HasError()) {
				return m, copyLogsCmd(buildClipboardContent(m.detail.Output))
			}
		case "up", "k":
			if m.logOffset > 0 {
				m.logOffset--
			}
		case "down", "j":
			m.logOffset++
		case "pgup":
			m.logOffset -= 10
			if m.logOffset < 0 {
				m.logOffset = 0
			}
		case "pgdown":
			m.logOffset += 10
		case "home":
			m.logOffset = 0
		case "end":
			m.logOffset = 999999 // clamped in View()
		}
	}
	return m, nil
}

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

	// logs — combined with errors into a single scrollable area
	if d.Output.HasLogs() || d.Output.Logs != "" || d.Output.HasError() {
		s += "\n" + title.Render("Output:") + "\n"

		var content strings.Builder
		if d.Output.HasLogs() {
			content.WriteString(d.Output.Logs)
			if d.Output.LogTruncated {
				content.WriteString("\n  ... (truncated by API)")
			}
		} else if d.Output.Logs != "" {
			content.WriteString(d.Output.Logs)
		}
		if d.Output.HasError() {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString("[ERROR] ")
			content.WriteString(d.Output.ErrorMsg)
			if d.Output.ErrorTrace != "" {
				content.WriteString("\n")
				content.WriteString(d.Output.ErrorTrace)
			}
		}
		rawLines := strings.Split(content.String(), "\n")
		logLines := wrapLogLines(rawLines, m.winWidth)

		linesBefore := strings.Count(s, "\n")
		afterLines := 1 // only the help line after logs
		vpHeight := m.winHeight - linesBefore - afterLines
		minVp := m.winHeight / 2
		if vpHeight < minVp {
			vpHeight = minVp
		}
		if vpHeight < 8 {
			vpHeight = 8
		}

		maxOffset := len(logLines) - vpHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.logOffset > maxOffset {
			m.logOffset = maxOffset
		}
		if m.logOffset < 0 {
			m.logOffset = 0
		}

		end := m.logOffset + vpHeight
		if end > len(logLines) {
			end = len(logLines)
		}

		if m.copied {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#4ECB71")).Render("  ✓ Copied to clipboard!") + "\n"
		} else {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("  [y] copy all  [↑↓/pgup/pgdn] scroll") + "\n"
		}
		for i := m.logOffset; i < end; i++ {
			s += "  " + logLines[i] + "\n"
		}

		if len(logLines) > vpHeight {
			pos := fmt.Sprintf("── %d/%d ──", end, len(logLines))
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render(pos) + "\n"
		}
	} else {
		s += "\n" + title.Render("Output:") + "\n"
		s += "  (no output)\n"
	}

	s += "\n[esc/backspace] back  [y] copy all logs"
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

// copyLogsCmd copies text to the system clipboard.
func copyLogsCmd(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return clipboardMsg{err: err}
		}
		return clipboardMsg{ok: true}
	}
}

// buildClipboardContent builds raw log content for clipboard copy.
func buildClipboardContent(o job.RunOutputInfo) string {
	var b strings.Builder
	b.WriteString(o.Logs)
	if o.HasError() {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(o.ErrorMsg)
		if o.ErrorTrace != "" {
			b.WriteString("\n")
			b.WriteString(o.ErrorTrace)
		}
	}
	return b.String()
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
