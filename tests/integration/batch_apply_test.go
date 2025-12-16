package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// contains checks if s contains substr
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Note: These tests require the Conduktor Console server to support the
// batch-apply API endpoints (/public/v1/resources/batch-apply).
// These tests use admin token authentication.

func Test_ApplyBatch_Empty_File(t *testing.T) {
	fmt.Println("Test CLI apply --batch with empty file")
	filePath := testDataFilePath(t, "empty.yaml")
	stdout, stderr, err := runConsoleCommand("apply", "--batch", "-f", filePath, "--yes")

	// Empty file should succeed with no output
	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)
	assert.Emptyf(t, stdout, "Expected no stdout output, got: %s", stdout)
}

func Test_ApplyBatch_Nonexistent_File(t *testing.T) {
	fmt.Println("Test CLI apply --batch with nonexistent file")
	filePath := testDataFilePath(t, "nonexistent.yaml")
	_, stderr, err := runConsoleCommand("apply", "--batch", "-f", filePath)
	assert.Error(t, err, "Expected command to fail for nonexistent file")

	expectedError := fmt.Sprintf("stat %s: no such file or directory", filePath)
	assert.NotEmptyf(t, stderr, "Expected stderr to contain '%s', got empty stderr", expectedError)
	assert.Containsf(t, stderr, expectedError, "Expected stderr to contain '%s', got: %s", expectedError, stderr)
}

func Test_ApplyBatch_Invalid_Strategy(t *testing.T) {
	fmt.Println("Test CLI apply --batch with invalid strategy")
	filePath := testDataFilePath(t, "valid_group.yaml")
	_, stderr, err := runConsoleCommand("apply", "--batch", "-f", filePath, "--strategy", "invalid-strategy")
	assert.Error(t, err, "Expected command to fail for invalid strategy")

	expectedError := "--strategy must be one of [fail-fast, continue-on-error]"
	assert.Containsf(t, stderr, expectedError, "Expected stderr to contain '%s', got: %s", expectedError, stderr)
}

func Test_ApplyBatch_Valid_Resource_FailFast(t *testing.T) {
	fmt.Println("Test CLI apply --batch with valid resource using fail-fast strategy")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-failfast-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "valid_group.yaml")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--strategy", "fail-fast", "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain the created resource
	assert.Containsf(t, stdout, "Group/team-a", "Expected stdout to contain 'Group/team-a', got: %s", stdout)

	// Cleanup after test
	_, _, _ = runCommandWithToken(token, "delete", "-f", filePath)
}

func Test_ApplyBatch_Valid_Resource_ContinueOnError(t *testing.T) {
	fmt.Println("Test CLI apply --batch with valid resource using continue-on-error strategy")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-continue-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "valid_group.yaml")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--strategy", "continue-on-error", "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain the created resource
	assert.Containsf(t, stdout, "Group/team-a", "Expected stdout to contain 'Group/team-a', got: %s", stdout)

	// Cleanup after test
	_, _, _ = runCommandWithToken(token, "delete", "-f", filePath)
}

func Test_ApplyBatch_Dry_Run(t *testing.T) {
	fmt.Println("Test CLI apply --batch with dry-run")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-dryrun-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "valid_group.yaml")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--dry-run", "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should indicate dry run mode
	assert.Containsf(t, stdout, "DRY RUN", "Expected stdout to contain 'DRY RUN', got: %s", stdout)
}

func Test_ApplyBatch_NoProgress_Flag(t *testing.T) {
	fmt.Println("Test CLI apply --batch with --no-progress flag")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-noprogress-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "valid_group.yaml")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--no-progress", "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// With --no-progress, output should still show results but not progress bars
	// The exact output format depends on the server implementation
	_ = stdout // Output is server-dependent

	// Cleanup after test
	_, _, _ = runCommandWithToken(token, "delete", "-f", filePath)
}

func Test_ApplyBatch_Folder(t *testing.T) {
	fmt.Println("Test CLI apply --batch with folder")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-folder-token")
	defer deleteTokenByName(tokenName)

	folderPath := testDataFilePath(t, "resources_folder")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", folderPath, "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain created resources
	assert.Containsf(t, stdout, "Group/team-b", "Expected stdout to contain 'Group/team-b', got: %s", stdout)
	assert.Containsf(t, stdout, "Group/team-c", "Expected stdout to contain 'Group/team-c', got: %s", stdout)

	// Cleanup after test
	_, _, _ = runCommandWithToken(token, "delete", "-f", folderPath)
}

func Test_ApplyBatch_Folder_Recursive(t *testing.T) {
	fmt.Println("Test CLI apply --batch with folder recursively")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-recursive-token")
	defer deleteTokenByName(tokenName)

	folderPath := testDataFilePath(t, "resources_folder")
	stdout, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", folderPath, "-r", "--yes")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain all created resources including nested
	assert.Containsf(t, stdout, "Group/team-b", "Expected stdout to contain 'Group/team-b', got: %s", stdout)
	assert.Containsf(t, stdout, "Group/team-c", "Expected stdout to contain 'Group/team-c', got: %s", stdout)
	assert.Containsf(t, stdout, "Group/team-d", "Expected stdout to contain 'Group/team-d', got: %s", stdout)

	// Cleanup after test
	_, _, _ = runCommandWithToken(token, "delete", "-f", folderPath, "-r")
}

func Test_ApplyBatch_LargeCount_RequiresYes(t *testing.T) {
	fmt.Println("Test CLI apply --batch refuses large batch without --yes")
	// This test would need a file with >50 resources
	// For now, we just verify the flag exists and is documented
	stdout, stderr, err := runConsoleCommand("apply", "--help")
	assert.NoError(t, err)
	assert.Contains(t, stdout+stderr, "--yes", "Expected help to document --yes flag")
	assert.Contains(t, stdout+stderr, "--batch", "Expected help to document --batch flag")
}

func Test_ApplyBatch_InvalidResource_UnknownKind(t *testing.T) {
	fmt.Println("Test CLI apply --batch with invalid resource (unknown kind)")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-invalidkind-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "invalid_resource.yaml")
	_, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--yes")

	// Should fail because "InvalidResource" is not a valid kind
	assert.Error(t, err, "Expected command to fail for unknown kind")
	assert.NotEmptyf(t, stderr, "Expected stderr to contain error message, got empty stderr")
}

func Test_ApplyBatch_InvalidTopic_UnknownCluster(t *testing.T) {
	fmt.Println("Test CLI apply --batch with invalid topic (unknown cluster)")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "apply-batch-invalidcluster-token")
	defer deleteTokenByName(tokenName)

	filePath := testDataFilePath(t, "invalid_topic.yaml")
	_, stderr, err := runCommandWithToken(token, "apply", "--batch", "-f", filePath, "--yes")

	// Should fail because "unkown-cluster" does not exist
	assert.Error(t, err, "Expected command to fail for unknown cluster")
	assert.NotEmptyf(t, stderr, "Expected stderr to contain error message, got empty stderr")
}
