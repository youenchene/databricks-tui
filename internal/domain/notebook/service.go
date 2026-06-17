package notebook

import (
	"context"
	"fmt"
)

// Service orchestrates notebook browsing use cases.
type Service struct {
	repo Repository
}

// NewService creates a notebook service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListPath returns all entries (notebooks + directories) at a workspace path.
func (s *Service) ListPath(ctx context.Context, path string) ([]Entry, error) {
	entries, err := s.repo.List(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("notebook service: list %s: %w", path, err)
	}
	return entries, nil
}

// Get returns a single notebook by path.
func (s *Service) Get(ctx context.Context, path string) (Notebook, error) {
	nb, err := s.repo.Get(ctx, path)
	if err != nil {
		return Notebook{}, fmt.Errorf("notebook service: get %s: %w", path, err)
	}
	return nb, nil
}
