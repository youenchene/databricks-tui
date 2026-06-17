package cluster

import (
	"context"
	"fmt"
)

// Service orchestrates cluster use cases.
// It depends only on the Repository port (interface), never on adapters.
type Service struct {
	repo Repository
}

// NewService creates a cluster service with the given repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListAll returns every cluster. Returns an empty slice on error.
func (s *Service) ListAll(ctx context.Context) ([]Cluster, error) {
	clusters, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("cluster service: list: %w", err)
	}
	return clusters, nil
}

// GetByID returns a single cluster.
func (s *Service) GetByID(ctx context.Context, id string) (Cluster, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Cluster{}, fmt.Errorf("cluster service: get %s: %w", id, err)
	}
	return c, nil
}
