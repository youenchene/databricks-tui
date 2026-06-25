package job_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/youenchene/databricks-tui/internal/domain/job"
)

// stubRepo implements job.Repository for testing.
type stubRepo struct {
	jobs     []job.Job
	runs     []job.Run
	detail   *job.JobDetail
	runDetail *job.RunDetail
	err      error
}

func (s *stubRepo) List(_ context.Context) ([]job.Job, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.jobs, nil
}

func (s *stubRepo) Runs(_ context.Context, jobID int64, limit int) ([]job.Run, error) {
	if s.err != nil {
		return nil, s.err
	}
	runs := make([]job.Run, 0, limit)
	for i, r := range s.runs {
		if i >= limit {
			break
		}
		if r.JobID == jobID {
			runs = append(runs, r)
		}
	}
	return runs, nil
}

func (s *stubRepo) GetDetail(_ context.Context, jobID int64) (*job.JobDetail, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.detail, nil
}

func (s *stubRepo) GetRunDetail(_ context.Context, runID int64) (*job.RunDetail, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.runDetail, nil
}

// --- Entity tests ---

func TestJob_Summary(t *testing.T) {
	tests := []struct {
		name     string
		j        job.Job
		expected string
	}{
		{
			name:     "manual job",
			j:        job.Job{Name: "daily-etl", Schedule: ""},
			expected: "daily-etl (manual)",
		},
		{
			name:     "scheduled job",
			j:        job.Job{Name: "hourly-report", Schedule: "0 * * * *"},
			expected: "hourly-report (0 * * * *)",
		},
		{
			name:     "empty name",
			j:        job.Job{Name: "", Schedule: "0 0 * * *"},
			expected: " (0 0 * * *)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.j.Summary())
		})
	}
}

func TestRun_ZeroEndAt(t *testing.T) {
	r := job.Run{
		RunID:   123,
		JobID:   42,
		State:   job.StateRunning,
		StartAt: timeFromMillis(1000),
	}
	assert.True(t, r.EndAt.IsZero(), "running job should have zero EndAt")
}

func TestJob_LastRunTime_ZeroWhenNeverRun(t *testing.T) {
	j := job.Job{ID: 1, Name: "new-job"}
	assert.True(t, j.LastRunTime.IsZero(), "never-run job should have zero LastRunTime")
}

func TestJob_LastRunTime_Set(t *testing.T) {
	ts := time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)
	j := job.Job{ID: 1, Name: "etl", LastRunTime: ts}
	assert.Equal(t, ts, j.LastRunTime)
}

func TestState_Constants(t *testing.T) {
	tests := []struct {
		name  string
		state job.State
		want  string
	}{
		{"pending", job.StatePending, "PENDING"},
		{"running", job.StateRunning, "RUNNING"},
		{"succeeded", job.StateSucceeded, "SUCCEEDED"},
		{"failed", job.StateFailed, "FAILED"},
		{"canceled", job.StateCanceled, "CANCELED"},
		{"unknown", job.StateUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.state))
		})
	}
}

// --- Service tests ---

func TestJobService_ListAll(t *testing.T) {
	repo := &stubRepo{
		jobs: []job.Job{
			{ID: 1, Name: "etl", Schedule: "0 * * * *"},
			{ID: 2, Name: "report", Schedule: ""},
			{ID: 3, Name: "cleanup", Schedule: "0 2 * * 0"},
		},
	}
	svc := job.NewService(repo)

	jobs, err := svc.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, jobs, 3)
	assert.Equal(t, "etl", jobs[0].Name)
	assert.Equal(t, "report", jobs[1].Name)
}

func TestJobService_ListAll_Empty(t *testing.T) {
	svc := job.NewService(&stubRepo{jobs: []job.Job{}})

	jobs, err := svc.ListAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

func TestJobService_ListAll_Error(t *testing.T) {
	svc := job.NewService(&stubRepo{err: errors.New("timeout")})

	_, err := svc.ListAll(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestJobService_RecentRuns(t *testing.T) {
	repo := &stubRepo{
		runs: []job.Run{
			{RunID: 10, JobID: 1, State: job.StateSucceeded},
			{RunID: 11, JobID: 1, State: job.StateFailed},
			{RunID: 12, JobID: 1, State: job.StateRunning},
			{RunID: 20, JobID: 2, State: job.StateSucceeded}, // different job
		},
	}
	svc := job.NewService(repo)

	runs, err := svc.RecentRuns(context.Background(), 1, 5)
	require.NoError(t, err)
	assert.Len(t, runs, 3)
	assert.Equal(t, int64(10), runs[0].RunID)
	assert.Equal(t, job.StateSucceeded, runs[0].State)
}

func TestJobService_RecentRuns_Limit(t *testing.T) {
	repo := &stubRepo{
		runs: []job.Run{
			{RunID: 1, JobID: 1}, {RunID: 2, JobID: 1},
			{RunID: 3, JobID: 1}, {RunID: 4, JobID: 1},
			{RunID: 5, JobID: 1},
		},
	}
	svc := job.NewService(repo)

	runs, err := svc.RecentRuns(context.Background(), 1, 3)
	require.NoError(t, err)
	assert.Len(t, runs, 3)
}

func TestJobService_RecentRuns_NoMatches(t *testing.T) {
	repo := &stubRepo{
		runs: []job.Run{{RunID: 1, JobID: 1}},
	}
	svc := job.NewService(repo)

	runs, err := svc.RecentRuns(context.Background(), 999, 10)
	require.NoError(t, err)
	assert.Empty(t, runs)
}

func TestJobService_RecentRuns_Error(t *testing.T) {
	svc := job.NewService(&stubRepo{err: errors.New("forbidden")})

	_, err := svc.RecentRuns(context.Background(), 1, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}

// timeFromMillis is a helper to create time.Time from milliseconds.
func timeFromMillis(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// --- Task tests ---

func TestTask_TaskType(t *testing.T) {
	tests := []struct {
		name     string
		t        job.Task
		wantType string
	}{
		{"notebook task", job.Task{TaskKey: "nb", NotebookPath: "/Shared/nb.py"}, "Notebook"},
		{"spark jar task", job.Task{TaskKey: "jar", MainClassName: "com.Main"}, "SparkJar"},
		{"python wheel task", job.Task{TaskKey: "whl", PackageName: "my-pkg"}, "PythonWheel"},
		{"sql task", job.Task{TaskKey: "sql", WarehouseID: "wh-1"}, "SQL"},
		{"dbt task", job.Task{TaskKey: "dbt", DbtProjectDir: "/dbt"}, "DBT"},
		{"pipeline task", job.Task{TaskKey: "pipe", PipelineID: "p-1"}, "Pipeline"},
		{"empty is unknown", job.Task{TaskKey: "unk"}, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.t.TaskType())
		})
	}
}

// --- JobDetail tests ---

func TestJobDetail_TaskCount(t *testing.T) {
	jd := job.JobDetail{
		Job:   job.Job{ID: 1, Name: "my-job"},
		Tasks: []job.Task{{TaskKey: "a"}, {TaskKey: "b"}, {TaskKey: "c"}},
	}
	assert.Equal(t, 3, jd.TaskCount())
}

func TestJobDetail_TaskCount_Empty(t *testing.T) {
	jd := job.JobDetail{Job: job.Job{ID: 1}}
	assert.Equal(t, 0, jd.TaskCount())
}

func TestJobDetail_Summary(t *testing.T) {
	jd := job.JobDetail{
		Job:   job.Job{ID: 1, Name: "my-etl", Schedule: "0 * * * *"},
		Tasks: []job.Task{{TaskKey: "ingest"}, {TaskKey: "transform"}},
	}
	assert.Contains(t, jd.Summary(), "my-etl")
	assert.Contains(t, jd.Summary(), "2 tasks")
}

// --- RunDetail tests ---

func TestRunDetail_Duration(t *testing.T) {
	rd := job.RunDetail{
		Run: job.Run{
			RunID:   1,
			JobID:   1,
			State:   job.StateSucceeded,
			StartAt: timeFromMillis(1700000000000),
			EndAt:   timeFromMillis(1700000005000),
		},
	}
	assert.Equal(t, "5s", rd.Duration())
}

func TestRunDetail_Duration_StillRunning(t *testing.T) {
	rd := job.RunDetail{
		Run: job.Run{
			RunID:   1,
			State:   job.StateRunning,
			StartAt: timeFromMillis(1700000000000),
		},
	}
	assert.Equal(t, "running...", rd.Duration())
}

func TestRunDetail_TaskStates(t *testing.T) {
	rd := job.RunDetail{
		Tasks: []job.TaskRun{
			{TaskKey: "a", State: job.StateSucceeded},
			{TaskKey: "b", State: job.StateSucceeded},
			{TaskKey: "c", State: job.StateFailed},
		},
	}
	succeeded, failed, running := rd.TaskStates()
	assert.Equal(t, 2, succeeded)
	assert.Equal(t, 1, failed)
	assert.Equal(t, 0, running)
}

// --- RunOutputInfo tests ---

func TestRunOutputInfo_HasLogs(t *testing.T) {
	o := job.RunOutputInfo{Logs: "some logs here"}
	assert.True(t, o.HasLogs())
	assert.False(t, job.RunOutputInfo{}.HasLogs())
}

func TestRunOutputInfo_HasError(t *testing.T) {
	o := job.RunOutputInfo{ErrorMsg: "division by zero"}
	assert.True(t, o.HasError())
	assert.False(t, job.RunOutputInfo{}.HasError())
}

func TestRunOutputInfo_FirstLines(t *testing.T) {
	o := job.RunOutputInfo{
		Logs: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11",
	}
	lines := o.FirstLines(5)
	assert.Len(t, lines, 5)
	assert.Equal(t, "line1", lines[0])
	assert.Equal(t, "line5", lines[4])
}

func TestRunOutputInfo_FirstLines_Empty(t *testing.T) {
	assert.Empty(t, job.RunOutputInfo{}.FirstLines(10))
}

// --- Entity creation helpers ---

func TestNewJobDetail(t *testing.T) {
	tasks := []job.Task{
		{TaskKey: "a", NotebookPath: "/nb.py"},
		{TaskKey: "b", MainClassName: "com.Main"},
	}
	jd := job.NewJobDetail(job.Job{ID: 1, Name: "test"}, tasks)
	assert.Equal(t, int64(1), jd.Job.ID)
	assert.Len(t, jd.Tasks, 2)
}

func TestNewJobDetail_SortsByDependencies(t *testing.T) {
	// c depends on b, b depends on a, d is independent
	tasks := []job.Task{
		{TaskKey: "c", DependsOn: []string{"b"}},
		{TaskKey: "b", DependsOn: []string{"a"}},
		{TaskKey: "d"},
		{TaskKey: "a"},
	}
	jd := job.NewJobDetail(job.Job{ID: 1, Name: "pipeline"}, tasks)
	require.Len(t, jd.Tasks, 4)

	// dependencies must appear before dependents
	keys := make([]string, len(jd.Tasks))
	for i, t := range jd.Tasks {
		keys[i] = t.TaskKey
	}
	idx := func(k string) int {
		for i, key := range keys {
			if key == k {
				return i
			}
		}
		return -1
	}
	assert.True(t, idx("a") < idx("b"), "a must come before b")
	assert.True(t, idx("b") < idx("c"), "b must come before c")
}

func TestNewJobDetail_SortIndependentTasks(t *testing.T) {
	tasks := []job.Task{
		{TaskKey: "c"},
		{TaskKey: "a"},
		{TaskKey: "b"},
	}
	jd := job.NewJobDetail(job.Job{ID: 1, Name: "test"}, tasks)
	require.Len(t, jd.Tasks, 3)
	// independent tasks are sorted alphabetically
	assert.Equal(t, "a", jd.Tasks[0].TaskKey)
	assert.Equal(t, "b", jd.Tasks[1].TaskKey)
	assert.Equal(t, "c", jd.Tasks[2].TaskKey)
}

func TestNewJobDetail_SingleTask(t *testing.T) {
	jd := job.NewJobDetail(job.Job{ID: 1}, []job.Task{{TaskKey: "only"}})
	assert.Len(t, jd.Tasks, 1)
	assert.Equal(t, "only", jd.Tasks[0].TaskKey)
}

func TestNewRunDetail(t *testing.T) {
	r := job.Run{RunID: 42, State: job.StateSucceeded}
	tasks := []job.TaskRun{{TaskKey: "a", State: job.StateSucceeded}}
	rd := job.NewRunDetail(r, tasks, job.RunOutputInfo{Logs: "ok"})
	assert.Equal(t, int64(42), rd.Run.RunID)
	assert.Len(t, rd.Tasks, 1)
	assert.Equal(t, "ok", rd.Output.Logs)
}

// --- Service tests for GetDetail/GetRunDetail ---

func TestJobService_GetDetail(t *testing.T) {
	expected := job.NewJobDetail(
		job.Job{ID: 1, Name: "my-job"},
		[]job.Task{{TaskKey: "a"}, {TaskKey: "b"}},
	)
	repo := &stubRepo{detail: &expected}
	svc := job.NewService(repo)

	jd, err := svc.GetDetail(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "my-job", jd.Job.Name)
	assert.Equal(t, 2, jd.TaskCount())
}

func TestJobService_GetDetail_Error(t *testing.T) {
	svc := job.NewService(&stubRepo{err: errors.New("not found")})

	_, err := svc.GetDetail(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestJobService_GetRunDetail(t *testing.T) {
	expected := job.NewRunDetail(
		job.Run{RunID: 42, State: job.StateSucceeded},
		[]job.TaskRun{{TaskKey: "a", State: job.StateSucceeded}},
		job.RunOutputInfo{Logs: "done"},
	)
	repo := &stubRepo{runDetail: &expected}
	svc := job.NewService(repo)

	rd, err := svc.GetRunDetail(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), rd.Run.RunID)
	assert.Equal(t, "done", rd.Output.Logs)
}

func TestJobService_GetRunDetail_Error(t *testing.T) {
	svc := job.NewService(&stubRepo{err: errors.New("forbidden")})

	_, err := svc.GetRunDetail(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
}
