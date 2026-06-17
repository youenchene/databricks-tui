package cluster

import "context"

// Repository defines the port for retrieving cluster data.
// Domain packages declare this interface; the databricks adapter implements it.
type Repository interface {
	// List returns all clusters visible to the current user.
	List(ctx context.Context) ([]Cluster, error)

	// Get returns a single cluster by its ID.
	Get(ctx context.Context, id string) (Cluster, error)
}
