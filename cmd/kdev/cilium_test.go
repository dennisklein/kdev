package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCiliumCmd(t *testing.T) {
	t.Run("creates cilium command", func(t *testing.T) {
		cmd := newCiliumCmd()

		assert.Equal(t, "cilium", cmd.Use)
		assert.Contains(t, cmd.Short, "cilium")
	})
}
