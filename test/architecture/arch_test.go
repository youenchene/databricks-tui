// Package architecture validates hexagonal architecture constraints.
package architecture

import (
	"path/filepath"
	"testing"

	"github.com/solrac97gr/goarchtest"
	"github.com/stretchr/testify/assert"
)

func TestHexagonalArchitecture(t *testing.T) {
	projectPath, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("failed to resolve project path: %v", err)
	}

	t.Run("domain layer has no external dependencies", func(t *testing.T) {
		result := goarchtest.InPath(projectPath).
			That().ResideInNamespace("internal/domain/*").
			ShouldNot().
			HaveDependencyOn("github.com/").
			GetResult()

		if !result.IsSuccessful {
			t.Errorf("domain layer should not depend on external packages:\n%v", result.FailingTypes)
		}
	})

	t.Run("domain layer does not depend on adapters", func(t *testing.T) {
		result := goarchtest.InPath(projectPath).
			That().ResideInNamespace("internal/domain/*").
			ShouldNot().
			HaveDependencyOn("internal/adapters").
			GetResult()

		assert.True(t, result.IsSuccessful, "domain must not import adapters: %v", result.FailingTypes)
	})

	t.Run("domain layer does not depend on ports", func(t *testing.T) {
		result := goarchtest.InPath(projectPath).
			That().ResideInNamespace("internal/domain/*").
			ShouldNot().
			HaveDependencyOn("internal/ports").
			GetResult()

		assert.True(t, result.IsSuccessful, "domain must not import ports: %v", result.FailingTypes)
	})

	t.Run("domain does not import SDK libraries", func(t *testing.T) {
		result := goarchtest.InPath(projectPath).
			That().ResideInNamespace("internal/domain/*").
			ShouldNot().
			HaveDependencyOn("charm.land/").
			And().
			ShouldNot().
			HaveDependencyOn("github.com/charmbracelet/").
			And().
			ShouldNot().
			HaveDependencyOn("github.com/databricks/").
			GetResult()

		assert.True(t, result.IsSuccessful, "domain must not import UI or SDK: %v", result.FailingTypes)
	})

	t.Run("adapters import domain but not ports", func(t *testing.T) {
		result := goarchtest.InPath(projectPath).
			That().ResideInNamespace("internal/adapters/*").
			ShouldNot().
			HaveDependencyOn("internal/ports").
			GetResult()

		assert.True(t, result.IsSuccessful, "adapters must not import ports: %v", result.FailingTypes)
	})
}
