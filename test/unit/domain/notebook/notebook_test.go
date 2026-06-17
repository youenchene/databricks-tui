package notebook_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/youenchene/databricks-tui/internal/domain/notebook"
)

// stubRepo implements notebook.Repository for testing.
type stubRepo struct {
	entries []notebook.Entry
	err     error
}

func (s *stubRepo) List(_ context.Context, path string) ([]notebook.Entry, error) {
	if s.err != nil {
		return nil, s.err
	}
	// Filter entries matching path prefix (simplified)
	var out []notebook.Entry
	for _, e := range s.entries {
		if len(e.Path) >= len(path) && e.Path[:len(path)] == path {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *stubRepo) Get(_ context.Context, path string) (notebook.Notebook, error) {
	if s.err != nil {
		return notebook.Notebook{}, s.err
	}
	for _, e := range s.entries {
		if !e.IsDir && e.Path == path {
			return notebook.Notebook{
				Path:     e.Path,
				Language: e.Language,
				Size:     e.Size,
			}, nil
		}
	}
	return notebook.Notebook{}, errors.New("not found")
}

// --- Entity tests ---

func TestNotebook_Summary(t *testing.T) {
	n := notebook.Notebook{
		Path:     "/Shared/my_notebook",
		Language: notebook.LangPython,
		Size:     2048,
	}
	assert.Equal(t, "/Shared/my_notebook (PYTHON)", n.Summary())
}

func TestLanguage_Constants(t *testing.T) {
	tests := []struct {
		name string
		lang notebook.Language
		want string
	}{
		{"python", notebook.LangPython, "PYTHON"},
		{"scala", notebook.LangScala, "SCALA"},
		{"sql", notebook.LangSQL, "SQL"},
		{"r", notebook.LangR, "R"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.lang))
		})
	}
}

func TestEntry_Notebook(t *testing.T) {
	e := notebook.Entry{
		IsDir:    false,
		Path:     "/Shared/analysis.py",
		Language: notebook.LangPython,
		Size:     1024,
	}
	assert.False(t, e.IsDir)
	assert.Equal(t, "/Shared/analysis.py", e.Path)
	assert.Equal(t, notebook.LangPython, e.Language)
	assert.Equal(t, int64(1024), e.Size)
}

func TestEntry_Directory(t *testing.T) {
	e := notebook.Entry{
		IsDir: true,
		Path:  "/Shared/",
	}
	assert.True(t, e.IsDir)
	assert.Equal(t, "/Shared/", e.Path)
	assert.Empty(t, e.Language) // not meaningful for directories
}

// --- Service tests ---

func TestNotebookService_ListPath(t *testing.T) {
	repo := &stubRepo{
		entries: []notebook.Entry{
			{Path: "/", IsDir: true},
			{Path: "/Shared/", IsDir: true},
			{Path: "/Users/", IsDir: true},
		},
	}
	svc := notebook.NewService(repo)

	entries, err := svc.ListPath(context.Background(), "/")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestNotebookService_ListPath_Empty(t *testing.T) {
	repo := &stubRepo{entries: []notebook.Entry{}}
	svc := notebook.NewService(repo)

	entries, err := svc.ListPath(context.Background(), "/empty")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestNotebookService_ListPath_Error(t *testing.T) {
	svc := notebook.NewService(&stubRepo{err: errors.New("permission denied")})

	_, err := svc.ListPath(context.Background(), "/restricted")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestNotebookService_Get(t *testing.T) {
	repo := &stubRepo{
		entries: []notebook.Entry{
			{Path: "/Shared/notebook.py", IsDir: false, Language: notebook.LangPython, Size: 512},
			{Path: "/Shared/folder", IsDir: true},
		},
	}
	svc := notebook.NewService(repo)

	nb, err := svc.Get(context.Background(), "/Shared/notebook.py")
	require.NoError(t, err)
	assert.Equal(t, "/Shared/notebook.py", nb.Path)
	assert.Equal(t, notebook.LangPython, nb.Language)
	assert.Equal(t, int64(512), nb.Size)
}

func TestNotebookService_Get_NotFound(t *testing.T) {
	repo := &stubRepo{entries: []notebook.Entry{}}
	svc := notebook.NewService(repo)

	_, err := svc.Get(context.Background(), "/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNotebookService_Get_IsDirectory(t *testing.T) {
	repo := &stubRepo{
		entries: []notebook.Entry{
			{Path: "/Shared/folder", IsDir: true},
		},
	}
	svc := notebook.NewService(repo)

	_, err := svc.Get(context.Background(), "/Shared/folder")
	assert.Error(t, err) // directory, not notebook
}
