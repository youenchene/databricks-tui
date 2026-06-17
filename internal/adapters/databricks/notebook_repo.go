package databricks

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/youenchene/databricks-tui/internal/domain/notebook"
)

// compile-time check: *NotebookRepo implements notebook.Repository
var _ notebook.Repository = (*NotebookRepo)(nil)

// NotebookRepo adapts the SDK client to the notebook.Repository port.
type NotebookRepo struct {
	client *Client
}

// NewNotebookRepo creates a notebook repository adapter.
func NewNotebookRepo(client *Client) *NotebookRepo {
	return &NotebookRepo{client: client}
}

// List returns all entries at a workspace path.
func (r *NotebookRepo) List(ctx context.Context, path string) ([]notebook.Entry, error) {
	entries, err := r.client.ListWorkspace(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("notebook repo: list %s: %w", path, err)
	}

	out := make([]notebook.Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, notebook.Entry{
			IsDir:    e.ObjectType == workspace.ObjectTypeDirectory,
			Path:     e.Path,
			Language: mapLanguage(e.Language),
			Size:     e.Size,
		})
	}
	return out, nil
}

// Get returns a notebook at the given path.
func (r *NotebookRepo) Get(ctx context.Context, path string) (notebook.Notebook, error) {
	entries, err := r.List(ctx, path)
	if err != nil {
		return notebook.Notebook{}, fmt.Errorf("notebook repo: get %s: %w", path, err)
	}
	for _, e := range entries {
		if !e.IsDir && e.Path == path {
			return notebook.Notebook{
				Path:     e.Path,
				Language: e.Language,
				Size:     e.Size,
			}, nil
		}
	}
	return notebook.Notebook{}, fmt.Errorf("notebook repo: not found: %s", path)
}

func mapLanguage(l workspace.Language) notebook.Language {
	switch l {
	case workspace.LanguagePython:
		return notebook.LangPython
	case workspace.LanguageScala:
		return notebook.LangScala
	case workspace.LanguageSql:
		return notebook.LangSQL
	case workspace.LanguageR:
		return notebook.LangR
	default:
		return notebook.LangPython
	}
}
