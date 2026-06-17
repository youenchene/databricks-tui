package databricks

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/youenchene/databricks-tui/internal/domain/cluster"
)

// compile-time check: *ClusterRepo implements cluster.Repository
var _ cluster.Repository = (*ClusterRepo)(nil)

// ClusterRepo adapts the SDK client to the cluster.Repository port.
type ClusterRepo struct {
	client *Client
}

// NewClusterRepo creates a cluster repository adapter.
func NewClusterRepo(client *Client) *ClusterRepo {
	return &ClusterRepo{client: client}
}

// List returns all clusters, mapped to domain models.
func (r *ClusterRepo) List(ctx context.Context) ([]cluster.Cluster, error) {
	details, err := r.client.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("cluster repo: list: %w", err)
	}

	clusters := make([]cluster.Cluster, 0, len(details))
	for _, d := range details {
		clusters = append(clusters, toDomainCluster(d))
	}
	return clusters, nil
}

// Get returns a single cluster by ID.
func (r *ClusterRepo) Get(ctx context.Context, id string) (cluster.Cluster, error) {
	detail, err := r.client.GetCluster(ctx, id)
	if err != nil {
		return cluster.Cluster{}, fmt.Errorf("cluster repo: get %s: %w", id, err)
	}
	return toDomainCluster(*detail), nil
}

// toDomainCluster maps an SDK ClusterDetails to the domain model.
func toDomainCluster(d compute.ClusterDetails) cluster.Cluster {
	return cluster.Cluster{
		ID:           d.ClusterId,
		Name:         d.ClusterName,
		State:        mapClusterState(d.State),
		SparkVersion: d.SparkVersion,
		NodeTypeID:   d.NodeTypeId,
		NumWorkers:   int32(d.NumWorkers),
		Creator:      d.CreatorUserName,
		CreatedAt:    msToTime(d.StartTime),
	}
}

func mapClusterState(s compute.State) cluster.State {
	switch s {
	case compute.StatePending:
		return cluster.StatePending
	case compute.StateRunning:
		return cluster.StateRunning
	case compute.StateRestarting:
		return cluster.StateRestarting
	case compute.StateResizing:
		return cluster.StateResizing
	case compute.StateTerminating:
		return cluster.StateTerminating
	case compute.StateTerminated:
		return cluster.StateTerminated
	case compute.StateError:
		return cluster.StateError
	default:
		return cluster.StateUnknown
	}
}
