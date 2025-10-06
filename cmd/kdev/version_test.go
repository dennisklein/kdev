package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersionCmd(t *testing.T) {
	t.Run("creates version command", func(t *testing.T) {
		cmd := newVersionCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, "version", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("executes version command", func(t *testing.T) {
		cmd := newVersionCmd()

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		// Version output should contain something (even if it's "(devel)")
		assert.NotEmpty(t, output)
	})
}
