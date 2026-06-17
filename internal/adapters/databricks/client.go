// Package databricks implements domain repository ports using the Databricks SDK.
package databricks

import (
	"context"
	"fmt"
	"log/slog"

	sdk "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Client wraps the Databricks WorkspaceClient.
type Client struct {
	sdk *sdk.WorkspaceClient
}

// NewClient creates a client from a .databrickscfg profile.
// Pass empty string to use the DEFAULT profile.
func NewClient(profile string) (*Client, error) {
	cfg := &sdk.Config{
		HTTPTimeoutSeconds: 30, // per-request timeout (pagination uses multiple requests)
	}
	if profile != "" {
		cfg.Profile = profile
	}

	w, err := sdk.NewWorkspaceClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("databricks client: %w", err)
	}
	slog.Info("databricks client created", "profile", profile)
	return &Client{sdk: w}, nil
}

// --- Clusters ---

func (c *Client) ListClusters(ctx context.Context) ([]compute.ClusterDetails, error) {
	all, err := c.sdk.Clusters.ListAll(ctx, compute.ListClustersRequest{})
	if err != nil {
		slog.Error("list clusters", "error", err)
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	return all, nil
}

func (c *Client) GetCluster(ctx context.Context, id string) (*compute.ClusterDetails, error) {
	detail, err := c.sdk.Clusters.Get(ctx, compute.GetClusterRequest{ClusterId: id})
	if err != nil {
		slog.Error("get cluster", "id", id, "error", err)
		return nil, fmt.Errorf("get cluster %s: %w", id, err)
	}
	return detail, nil
}

// --- Jobs ---

func (c *Client) ListJobs(ctx context.Context) ([]jobs.BaseJob, error) {
	all, err := c.sdk.Jobs.ListAll(ctx, jobs.ListJobsRequest{})
	if err != nil {
		slog.Error("list jobs", "error", err)
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	return all, nil
}

func (c *Client) GetJob(ctx context.Context, jobID int64) (*jobs.Job, error) {
	j, err := c.sdk.Jobs.Get(ctx, jobs.GetJobRequest{JobId: jobID})
	if err != nil {
		slog.Error("get job", "jobID", jobID, "error", err)
		return nil, fmt.Errorf("get job %d: %w", jobID, err)
	}
	return j, nil
}

func (c *Client) ListJobRuns(ctx context.Context, jobID int64, limit int) ([]jobs.BaseRun, error) {
	all, err := c.sdk.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{JobId: jobID, Limit: limit})
	if err != nil {
		slog.Error("list job runs", "jobID", jobID, "error", err)
		return nil, fmt.Errorf("list runs for job %d: %w", jobID, err)
	}
	return all, nil
}

func (c *Client) GetRun(ctx context.Context, runID int64) (*jobs.Run, error) {
	r, err := c.sdk.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		slog.Error("get run", "runID", runID, "error", err)
		return nil, fmt.Errorf("get run %d: %w", runID, err)
	}
	return r, nil
}

func (c *Client) GetRunOutput(ctx context.Context, runID int64) (*jobs.RunOutput, error) {
	o, err := c.sdk.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: runID})
	if err != nil {
		slog.Error("get run output", "runID", runID, "error", err)
		return nil, fmt.Errorf("get run output %d: %w", runID, err)
	}
	return o, nil
}

// --- Workspace ---

func (c *Client) ListWorkspace(ctx context.Context, path string) ([]workspace.ObjectInfo, error) {
	all, err := c.sdk.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{Path: path})
	if err != nil {
		slog.Error("list workspace", "path", path, "error", err)
		return nil, fmt.Errorf("list workspace %s: %w", path, err)
	}
	return all, nil
}
