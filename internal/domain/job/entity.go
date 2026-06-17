// Package job defines the job browsing domain.
package job

import (
	"fmt"
	"strings"
	"time"
)

// State represents a job run lifecycle state.
type State string

const (
	StatePending   State = "PENDING"
	StateRunning   State = "RUNNING"
	StateSucceeded State = "SUCCEEDED"
	StateFailed    State = "FAILED"
	StateCanceled  State = "CANCELED"
	StateUnknown   State = "UNKNOWN"
)

// Job represents a Databricks job definition.
type Job struct {
	ID        int64
	Name      string
	Schedule  string // cron expression or empty
	Creator   string
	CreatedAt time.Time
}

// Run represents a single execution of a job.
type Run struct {
	RunID   int64
	JobID   int64
	State   State
	StartAt time.Time
	EndAt   time.Time // zero if still running
}

// Task represents a single task within a job definition.
type Task struct {
	TaskKey      string
	Description  string
	DependsOn    []string
	NotebookPath string // non-empty = notebook task
	MainClassName string // non-empty = spark jar task
	PackageName  string // non-empty = python wheel task
	WarehouseID  string // non-empty = SQL task
	DbtProjectDir string // non-empty = DBT task
	PipelineID   string // non-empty = pipeline task
}

// TaskType returns a human-readable task type label.
func (t Task) TaskType() string {
	switch {
	case t.NotebookPath != "":
		return "Notebook"
	case t.MainClassName != "":
		return "SparkJar"
	case t.PackageName != "":
		return "PythonWheel"
	case t.WarehouseID != "":
		return "SQL"
	case t.DbtProjectDir != "":
		return "DBT"
	case t.PipelineID != "":
		return "Pipeline"
	default:
		return "Unknown"
	}
}

// TaskRun represents a task execution within a job run.
type TaskRun struct {
	TaskKey     string
	State       State
	StartAt     time.Time
	EndAt       time.Time
	RunDuration time.Duration
}

// JobDetail is a composite of a job and its tasks (for the detail view).
type JobDetail struct {
	Job   Job
	Tasks []Task
}

// NewJobDetail creates a JobDetail.
func NewJobDetail(j Job, tasks []Task) JobDetail {
	return JobDetail{Job: j, Tasks: tasks}
}

// TaskCount returns the number of tasks in this job.
func (jd JobDetail) TaskCount() int {
	return len(jd.Tasks)
}

// Summary returns a formatted one-line summary.
func (jd JobDetail) Summary() string {
	return fmt.Sprintf("%s — %d tasks", jd.Job.Summary(), jd.TaskCount())
}

// RunOutputInfo holds the logs and output of a job run.
type RunOutputInfo struct {
	NotebookResult string
	SQLResult      string
	Logs           string
	ErrorMsg       string
	ErrorTrace     string
}

// HasLogs returns true when there are logs available.
func (o RunOutputInfo) HasLogs() bool {
	return o.Logs != ""
}

// HasError returns true when the run had an error.
func (o RunOutputInfo) HasError() bool {
	return o.ErrorMsg != ""
}

// FirstLines returns the first N log lines, split by newlines.
func (o RunOutputInfo) FirstLines(n int) []string {
	if o.Logs == "" {
		return nil
	}
	lines := strings.SplitN(o.Logs, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// RunDetail is a composite of a run, its task executions, and output.
type RunDetail struct {
	Run    Run
	Tasks  []TaskRun
	Output RunOutputInfo
}

// NewRunDetail creates a RunDetail.
func NewRunDetail(r Run, tasks []TaskRun, output RunOutputInfo) RunDetail {
	return RunDetail{Run: r, Tasks: tasks, Output: output}
}

// Duration returns "XmYs" for completed runs or "running..." if still active.
func (rd RunDetail) Duration() string {
	if rd.Run.EndAt.IsZero() {
		return "running..."
	}
	d := rd.Run.EndAt.Sub(rd.Run.StartAt)
	return d.Truncate(time.Second).String()
}

// TaskStates returns the count of succeeded, failed, and running tasks.
func (rd RunDetail) TaskStates() (succeeded, failed, running int) {
	for _, t := range rd.Tasks {
		switch t.State {
		case StateSucceeded:
			succeeded++
		case StateFailed:
			failed++
		case StateRunning, StatePending:
			running++
		}
	}
	return
}

// Summary returns a one-line human-readable summary of a job.
func (j Job) Summary() string {
	sched := "manual"
	if j.Schedule != "" {
		sched = j.Schedule
	}
	return j.Name + " (" + sched + ")"
}
