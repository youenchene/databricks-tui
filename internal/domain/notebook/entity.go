// Package notebook defines the notebook browsing domain.
package notebook

// Language is the notebook language.
type Language string

const (
	LangPython   Language = "PYTHON"
	LangScala    Language = "SCALA"
	LangSQL      Language = "SQL"
	LangR        Language = "R"
)

// Notebook represents a Databricks notebook in the workspace.
type Notebook struct {
	Path     string
	Language Language
	Size     int64 // bytes
}

// Directory represents a workspace directory (folder containing notebooks).
type Directory struct {
	Path string
}

// Entry is either a Notebook or a Directory.
type Entry struct {
	IsDir      bool
	Path       string
	Language   Language // only meaningful for notebooks
	Size       int64    // only meaningful for notebooks
}

// Summary returns a human-readable one-line description.
func (n Notebook) Summary() string {
	return n.Path + " (" + string(n.Language) + ")"
}
