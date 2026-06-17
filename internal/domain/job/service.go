package job

import (
	"context"
	"fmt"
)

// Service orchestrates job use cases.
type Service struct {
	repo Repository
}

// NewService creates a job service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAll returns every job.
func (s *Service) ListAll(ctx context.Context) ([]Job, error) {
	jobs, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("job service: list: %w", err)
	}
	return jobs, nil
}

// RecentRuns returns the last N runs for a job.
func (s *Service) RecentRuns(ctx context.Context, jobID int64, limit int) ([]Run, error) {
	runs, err := s.repo.Runs(ctx, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("job service: runs %d: %w", jobID, err)
	}
	return runs, nil
}

// GetDetail returns a job with its tasks.
func (s *Service) GetDetail(ctx context.Context, jobID int64) (*JobDetail, error) {
	jd, err := s.repo.GetDetail(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("job service: detail %d: %w", jobID, err)
	}
	return jd, nil
}

// GetRunDetail returns a run with task executions and output.
func (s *Service) GetRunDetail(ctx context.Context, runID int64) (*RunDetail, error) {
	rd, err := s.repo.GetRunDetail(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("job service: run detail %d: %w", runID, err)
	}
	return rd, nil
}
