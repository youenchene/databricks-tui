package notebook

import "context"

// Repository defines the port for browsing workspace notebooks.
type Repository interface {
	// List returns all entries (notebooks + directories) at the given path.
	// Pass "/" for the workspace root.
	List(ctx context.Context, path string) ([]Entry, error)

	// Get returns the Notebook at the given path.
	// Returns an error if the path points to a directory.
	Get(ctx context.Context, path string) (Notebook, error)
}
