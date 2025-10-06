package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKindCmd(t *testing.T) {
	t.Run("creates kind command", func(t *testing.T) {
		cmd := newKindCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "kind", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("disables flag parsing for passthrough", func(t *testing.T) {
		cmd := newKindCmd()

		// Should disable flag parsing to pass all flags to kind
		assert.True(t, cmd.DisableFlagParsing)
	})
}

// Note: We cannot test runKind() execution because it calls syscall.Exec()
// which replaces the current process. This would terminate the test.
// The actual kind execution is tested through integration tests.
