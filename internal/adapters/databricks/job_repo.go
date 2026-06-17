package databricks

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/youenchene/databricks-tui/internal/domain/job"
)

// compile-time check: *JobRepo implements job.Repository
var _ job.Repository = (*JobRepo)(nil)

// JobRepo adapts the SDK client to the job.Repository port.
type JobRepo struct {
	client *Client
}

// NewJobRepo creates a job repository adapter.
func NewJobRepo(client *Client) *JobRepo {
	return &JobRepo{client: client}
}

// List returns all jobs mapped to domain models.
func (r *JobRepo) List(ctx context.Context) ([]job.Job, error) {
	jobsSDK, err := r.client.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("job repo: list: %w", err)
	}

	jobs := make([]job.Job, 0, len(jobsSDK))
	for _, j := range jobsSDK {
		jobs = append(jobs, job.Job{
			ID:        j.JobId,
			Name:      jobSettingsName(j.Settings),
			Schedule:  jobSettingsCron(j.Settings),
			Creator:   j.CreatorUserName,
			CreatedAt: msToTime(j.CreatedTime),
		})
	}
	return jobs, nil
}

// Runs returns recent runs for a job.
func (r *JobRepo) Runs(ctx context.Context, jobID int64, limit int) ([]job.Run, error) {
	runsSDK, err := r.client.ListJobRuns(ctx, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("job repo: runs %d: %w", jobID, err)
	}

	runs := make([]job.Run, 0, len(runsSDK))
	for _, run := range runsSDK {
		runs = append(runs, job.Run{
			RunID:   run.RunId,
			JobID:   jobID,
			State:   mapJobRunState(run.State),
			StartAt: msToTime(run.StartTime),
			EndAt:   msToTime(run.EndTime),
		})
	}
	return runs, nil
}

// GetDetail returns a job with all its tasks.
func (r *JobRepo) GetDetail(ctx context.Context, jobID int64) (*job.JobDetail, error) {
	j, err := r.client.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job repo: detail %d: %w", jobID, err)
	}

	tasks := make([]job.Task, 0, len(j.Settings.Tasks))
	for _, t := range j.Settings.Tasks {
		tasks = append(tasks, mapSDKTask(t))
	}

	jd := job.NewJobDetail(job.Job{
		ID:        j.JobId,
		Name:      jobSettingsName(j.Settings),
		Schedule:  jobSettingsCron(j.Settings),
		Creator:   j.CreatorUserName,
		CreatedAt: msToTime(j.CreatedTime),
	}, tasks)
	return &jd, nil
}

// GetRunDetail returns a run with task executions and output.
func (r *JobRepo) GetRunDetail(ctx context.Context, runID int64) (*job.RunDetail, error) {
	run, err := r.client.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("job repo: run detail %d: %w", runID, err)
	}

	tasks := make([]job.TaskRun, 0, len(run.Tasks))
	for _, t := range run.Tasks {
		tasks = append(tasks, mapSDKRunTask(t))
	}

	output, _ := r.client.GetRunOutput(ctx, runID)

	rd := job.NewRunDetail(job.Run{
		RunID:   run.RunId,
		JobID:   run.JobId,
		State:   mapJobRunState(run.State),
		StartAt: msToTime(run.StartTime),
		EndAt:   msToTime(run.EndTime),
	}, tasks, mapSDKRunOutput(output))
	return &rd, nil
}

// --- SDK field helpers ---

func jobSettingsName(s *jobs.JobSettings) string {
	if s == nil {
		return ""
	}
	return s.Name
}

func jobSettingsCron(s *jobs.JobSettings) string {
	if s == nil || s.Schedule == nil {
		return ""
	}
	return s.Schedule.QuartzCronExpression
}

func mapJobRunState(s *jobs.RunState) job.State {
	if s == nil {
		return job.StateUnknown
	}
	switch s.LifeCycleState {
	case jobs.RunLifeCycleStatePending:
		return job.StatePending
	case jobs.RunLifeCycleStateRunning:
		return job.StateRunning
	case jobs.RunLifeCycleStateTerminated, jobs.RunLifeCycleStateTerminating:
		switch s.ResultState {
		case jobs.RunResultStateSuccess:
			return job.StateSucceeded
		case jobs.RunResultStateFailed:
			return job.StateFailed
		case jobs.RunResultStateCanceled:
			return job.StateCanceled
		default:
			return job.StateUnknown
		}
	default:
		return job.StateUnknown
	}
}

// mapSDKTask maps an SDK Task to the domain Task.
func mapSDKTask(t jobs.Task) job.Task {
	return job.Task{
		TaskKey:      t.TaskKey,
		Description:  t.Description,
		DependsOn:    taskDependsOn(t.DependsOn),
		NotebookPath: taskNotebookPath(t.NotebookTask),
		MainClassName: taskMainClass(t.SparkJarTask),
		PackageName:  taskPackageName(t.PythonWheelTask),
		WarehouseID:  taskWarehouseID(t.SqlTask),
		DbtProjectDir: taskDbtDir(t.DbtTask),
		PipelineID:   taskPipelineID(t.PipelineTask),
	}
}

// mapSDKRunTask maps an SDK RunTask to domain TaskRun.
func mapSDKRunTask(t jobs.RunTask) job.TaskRun {
	return job.TaskRun{
		TaskKey:     t.TaskKey,
		State:       mapJobRunState(t.State),
		StartAt:     msToTime(t.StartTime),
		EndAt:       msToTime(t.EndTime),
		RunDuration: time.Duration(t.RunDuration) * time.Millisecond,
	}
}

// mapSDKRunOutput maps an SDK RunOutput to domain RunOutputInfo.
func mapSDKRunOutput(o *jobs.RunOutput) job.RunOutputInfo {
	if o == nil {
		return job.RunOutputInfo{}
	}
	return job.RunOutputInfo{
		NotebookResult: runOutputNotebook(o),
		SQLResult:      runOutputSQL(o),
		Logs:           o.Logs,
		ErrorMsg:       o.Error,
		ErrorTrace:     o.ErrorTrace,
	}
}

// --- sub-field extractors ---

func taskDependsOn(deps []jobs.TaskDependency) []string {
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		out = append(out, d.TaskKey)
	}
	return out
}

func taskNotebookPath(t *jobs.NotebookTask) string {
	if t == nil {
		return ""
	}
	return t.NotebookPath
}

func taskMainClass(t *jobs.SparkJarTask) string {
	if t == nil {
		return ""
	}
	return t.MainClassName
}

func taskPackageName(t *jobs.PythonWheelTask) string {
	if t == nil {
		return ""
	}
	return t.PackageName
}

func taskWarehouseID(t *jobs.SqlTask) string {
	if t == nil {
		return ""
	}
	return t.WarehouseId
}

func taskDbtDir(t *jobs.DbtTask) string {
	if t == nil {
		return ""
	}
	return t.ProjectDirectory
}

func taskPipelineID(t *jobs.PipelineTask) string {
	if t == nil {
		return ""
	}
	return t.PipelineId
}

func runOutputNotebook(o *jobs.RunOutput) string {
	if o.NotebookOutput == nil {
		return ""
	}
	return o.NotebookOutput.Result
}

func runOutputSQL(o *jobs.RunOutput) string {
	if o.SqlOutput == nil || o.SqlOutput.QueryOutput == nil {
		return ""
	}
	return o.SqlOutput.QueryOutput.OutputLink
}

// msToTime converts a Unix epoch in milliseconds to time.Time.
func msToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
