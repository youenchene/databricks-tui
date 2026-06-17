package cluster_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/youenchene/databricks-tui/internal/domain/cluster"
)

// stubRepo implements cluster.Repository for testing.
type stubRepo struct {
	clusters []cluster.Cluster
	err      error
}

func (s *stubRepo) List(_ context.Context) ([]cluster.Cluster, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.clusters, nil
}

func (s *stubRepo) Get(_ context.Context, id string) (cluster.Cluster, error) {
	if s.err != nil {
		return cluster.Cluster{}, s.err
	}
	for _, c := range s.clusters {
		if c.ID == id {
			return c, nil
		}
	}
	return cluster.Cluster{}, errors.New("not found")
}

func TestCluster_IsAlive(t *testing.T) {
	tests := []struct {
		name  string
		state cluster.State
		want  bool
	}{
		{"running is alive", cluster.StateRunning, true},
		{"pending is alive", cluster.StatePending, true},
		{"terminated is not alive", cluster.StateTerminated, false},
		{"error is not alive", cluster.StateError, false},
		{"unknown is not alive", cluster.StateUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cluster.Cluster{State: tt.state}
			assert.Equal(t, tt.want, c.IsAlive())
		})
	}
}

func TestCluster_Summary(t *testing.T) {
	c := cluster.Cluster{Name: "my-cluster", State: cluster.StateRunning}
	assert.Equal(t, "my-cluster (RUNNING)", c.Summary())
}

func TestService_ListAll(t *testing.T) {
	repo := &stubRepo{
		clusters: []cluster.Cluster{
			{ID: "c1", Name: "cluster-1", State: cluster.StateRunning},
			{ID: "c2", Name: "cluster-2", State: cluster.StateTerminated},
		},
	}
	svc := cluster.NewService(repo)

	clusters, err := svc.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, clusters, 2)
	assert.Equal(t, "cluster-1", clusters[0].Name)
}

func TestService_ListAll_Error(t *testing.T) {
	repo := &stubRepo{err: errors.New("network failure")}
	svc := cluster.NewService(repo)

	_, err := svc.ListAll(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network failure")
}

func TestService_GetByID(t *testing.T) {
	repo := &stubRepo{
		clusters: []cluster.Cluster{
			{ID: "c1", Name: "cluster-1"},
		},
	}
	svc := cluster.NewService(repo)

	c, err := svc.GetByID(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, "cluster-1", c.Name)
}
