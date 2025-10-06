package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKubectlCmd(t *testing.T) {
	t.Run("creates kubectl command", func(t *testing.T) {
		cmd := newKubectlCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "kubectl", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("disables flag parsing for passthrough", func(t *testing.T) {
		cmd := newKubectlCmd()

		// Should disable flag parsing to pass all flags to kubectl
		assert.True(t, cmd.DisableFlagParsing)
	})
}

// Note: We cannot test runKubectl() execution because it calls syscall.Exec()
// which replaces the current process. This would terminate the test.
// The actual kubectl execution is tested through integration tests.
