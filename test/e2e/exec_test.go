// SPDX-FileCopyrightText: 2025 GSI Helmholtzzentrum für Schwerionenforschung GmbH
//
// SPDX-License-Identifier: MPL-2.0

//go:build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests verify that kdev correctly execs tools using syscall.Exec.
// Uses mock tools placed at expected cache locations to avoid real downloads.
// The version is fetched from the real endpoint to ensure we place the mock
// at the correct version path that kdev expects.

func TestMain(m *testing.M) {
	// Build kdev binary before running tests
	if err := buildKdev(); err != nil {
		panic("failed to build kdev: " + err.Error())
	}

	// Build mock tool binary
	if err := buildMockTool(); err != nil {
		panic("failed to build mock tool: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	_ = os.Remove(kdevBinary())
	_ = os.Remove(mockToolBinary())

	os.Exit(code)
}

func buildKdev() error {
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", kdevBinary(), "../../cmd/kdev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func buildMockTool() error {
	cmd := exec.Command("go", "build", "-o", mockToolBinary(), "./mock/tool.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func kdevBinary() string {
	wd, _ := os.Getwd()

	return filepath.Join(wd, "kdev-e2e")
}

func mockToolBinary() string {
	wd, _ := os.Getwd()

	return filepath.Join(wd, "mock-tool")
}

// setupMockTool places the mock tool binary at the expected cache location
// for a specific tool and version.
func setupMockTool(t *testing.T, homeDir, toolName, version string) {
	t.Helper()

	// Create cache directory structure
	cacheDir := filepath.Join(homeDir, ".kdev", "kdev", toolName, version)
	err := os.MkdirAll(cacheDir, 0o755)
	require.NoError(t, err)

	// Copy mock tool binary to cache location with correct name
	mockSrc := mockToolBinary()
	mockDst := filepath.Join(cacheDir, toolName)

	srcData, err := os.ReadFile(mockSrc)
	require.NoError(t, err)

	err = os.WriteFile(mockDst, srcData, 0o755)
	require.NoError(t, err)
}

// fetchKubectlVersion fetches the current kubectl stable version.
func fetchKubectlVersion(t *testing.T) string {
	t.Helper()

	resp, err := http.Get("https://dl.k8s.io/release/stable.txt")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return strings.TrimSpace(string(body))
}

// fetchKindVersion fetches the current kind latest version.
func fetchKindVersion(t *testing.T) string {
	t.Helper()

	resp, err := http.Get("https://api.github.com/repos/kubernetes-sigs/kind/releases/latest")
	require.NoError(t, err)
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}

	err = json.NewDecoder(resp.Body).Decode(&release)
	require.NoError(t, err)

	return release.TagName
}

func TestKubectlExec(t *testing.T) {
	// Create temporary home directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Fetch current kubectl version to place mock at correct location
	kubectlVersion := fetchKubectlVersion(t)

	// Setup mock kubectl binary at expected cache location
	setupMockTool(t, homeDir, "kubectl", kubectlVersion)

	// Execute kdev kubectl with arguments
	cmd := exec.Command(kdevBinary(), "kubectl", "get", "pods")
	output, err := cmd.CombinedOutput()

	// Verify execution
	require.NoError(t, err, "kdev kubectl should execute successfully")

	outputStr := string(output)
	// Verify mock kubectl was executed
	assert.Contains(t, outputStr, "MOCK-KUBECTL-EXECUTED", "should show mock kubectl was executed")
	assert.Contains(t, outputStr, "ARGS: get pods", "should pass through arguments")
}

func TestKindExec(t *testing.T) {
	// Create temporary home directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Fetch current kind version to place mock at correct location
	kindVersion := fetchKindVersion(t)

	// Setup mock kind binary at expected cache location
	setupMockTool(t, homeDir, "kind", kindVersion)

	// Execute kdev kind with arguments
	cmd := exec.Command(kdevBinary(), "kind", "create", "cluster")
	output, err := cmd.CombinedOutput()

	// Verify execution
	require.NoError(t, err, "kdev kind should execute successfully")

	outputStr := string(output)
	// Verify mock kind was executed
	assert.Contains(t, outputStr, "MOCK-KIND-EXECUTED", "should show mock kind was executed")
	assert.Contains(t, outputStr, "ARGS: create cluster", "should pass through arguments")
}

func TestKubectlExecNoArgs(t *testing.T) {
	// Create temporary home directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Fetch current kubectl version to place mock at correct location
	kubectlVersion := fetchKubectlVersion(t)

	// Setup mock kubectl binary
	setupMockTool(t, homeDir, "kubectl", kubectlVersion)

	// Execute kdev kubectl with no arguments
	cmd := exec.Command(kdevBinary(), "kubectl")
	output, err := cmd.CombinedOutput()

	// Verify execution
	require.NoError(t, err, "kdev kubectl should execute successfully")

	outputStr := string(output)
	assert.Contains(t, outputStr, "MOCK-KUBECTL-EXECUTED", "should show mock kubectl was executed")
	// Should not contain ARGS line when no arguments
	assert.NotContains(t, outputStr, "ARGS:", "should not show args line with no arguments")
}

func TestKubectlExecWithFlags(t *testing.T) {
	// Create temporary home directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Fetch current kubectl version to place mock at correct location
	kubectlVersion := fetchKubectlVersion(t)

	// Setup mock kubectl binary
	setupMockTool(t, homeDir, "kubectl", kubectlVersion)

	// Execute kdev kubectl with flags
	cmd := exec.Command(kdevBinary(), "kubectl", "get", "pods", "-n", "default", "--watch")
	output, err := cmd.CombinedOutput()

	// Verify execution
	require.NoError(t, err, "kdev kubectl should execute successfully")

	outputStr := string(output)
	assert.Contains(t, outputStr, "MOCK-KUBECTL-EXECUTED", "should show mock kubectl was executed")
	assert.Contains(t, outputStr, "ARGS: get pods -n default --watch", "should pass through all flags")
}
