package job

import "context"

// Repository defines the port for retrieving job data.
type Repository interface {
	// List returns all jobs visible to the current user.
	List(ctx context.Context) ([]Job, error)

	// Runs returns recent runs for a specific job.
	Runs(ctx context.Context, jobID int64, limit int) ([]Run, error)

	// GetDetail returns a job with its full task list.
	GetDetail(ctx context.Context, jobID int64) (*JobDetail, error)

	// GetRunDetail returns a run with task executions and output.
	GetRunDetail(ctx context.Context, runID int64) (*RunDetail, error)
}
