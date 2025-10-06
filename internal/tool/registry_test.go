//nolint:testpackage // internal functions require same package
package tool

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	t.Run("creates registry with all tools", func(t *testing.T) {
		var buf bytes.Buffer

		registry := NewRegistry(&buf)

		require.NotNil(t, registry)
		require.NotNil(t, registry.tools)

		kubectl := registry.Get("kubectl")
		require.NotNil(t, kubectl)
		assert.Equal(t, "kubectl", kubectl.Name)

		kind := registry.Get("kind")
		require.NotNil(t, kind)
		assert.Equal(t, "kind", kind.Name)
	})

	t.Run("creates registry with nil progress writer", func(t *testing.T) {
		registry := NewRegistry(nil)

		require.NotNil(t, registry)

		kubectl := registry.Get("kubectl")
		require.NotNil(t, kubectl)
		assert.Nil(t, kubectl.ProgressWriter)

		kind := registry.Get("kind")
		require.NotNil(t, kind)
		assert.Nil(t, kind.ProgressWriter)
	})
}

func TestRegistryGet(t *testing.T) {
	t.Run("returns tool when found", func(t *testing.T) {
		registry := NewRegistry(nil)

		tool := registry.Get("kubectl")
		require.NotNil(t, tool)
		assert.Equal(t, "kubectl", tool.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		registry := NewRegistry(nil)

		tool := registry.Get("nonexistent")
		assert.Nil(t, tool)
	})
}

func TestRegistryAll(t *testing.T) {
	t.Run("returns all tool names sorted alphabetically", func(t *testing.T) {
		registry := NewRegistry(nil)

		names := registry.All()
		require.Len(t, names, 3)

		// Names should be sorted alphabetically: cilium, kind, kubectl
		assert.Equal(t, "cilium", names[0])
		assert.Equal(t, "kind", names[1])
		assert.Equal(t, "kubectl", names[2])
	})
}

func TestRegistryAllTools(t *testing.T) {
	t.Run("returns all tool instances sorted by name", func(t *testing.T) {
		registry := NewRegistry(nil)

		tools := registry.AllTools()
		require.Len(t, tools, 3)

		// Tools should be sorted alphabetically: cilium, kind, kubectl
		assert.Equal(t, "cilium", tools[0].Name)
		assert.Equal(t, "kind", tools[1].Name)
		assert.Equal(t, "kubectl", tools[2].Name)
	})
}
