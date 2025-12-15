package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: These tests require the Conduktor Console server to support the
// template API endpoints (/public/v1/resources/template).
// These tests use admin token authentication.

func Test_TemplateFromServer_ListKinds(t *testing.T) {
	fmt.Println("Test CLI template --from-server list kinds")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-list-token")
	defer deleteTokenByName(tokenName)

	stdout, stderr, err := runCommandWithToken(token, "template", "--from-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should list available kinds
	assert.Containsf(t, stdout, "Available Kinds", "Expected stdout to contain 'Available Kinds', got: %s", stdout)
}

func Test_TemplateFromServer_GetTopicTemplate(t *testing.T) {
	fmt.Println("Test CLI template --from-server get Topic template")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-topic-token")
	defer deleteTokenByName(tokenName)

	stdout, stderr, err := runCommandWithToken(token, "template", "Topic", "--from-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template with Topic kind
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
	// Template should start with ---
	assert.True(t, strings.HasPrefix(strings.TrimSpace(stdout), "---"), "Expected stdout to start with '---', got: %s", stdout)
}

func Test_TemplateFromServer_GetGroupTemplate(t *testing.T) {
	fmt.Println("Test CLI template --from-server get Group template")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-group-token")
	defer deleteTokenByName(tokenName)

	stdout, stderr, err := runCommandWithToken(token, "template", "Group", "--from-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
}

func Test_TemplateFromServer_GetUserTemplate(t *testing.T) {
	fmt.Println("Test CLI template --from-server get User template")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-user-token")
	defer deleteTokenByName(tokenName)

	stdout, stderr, err := runCommandWithToken(token, "template", "User", "--from-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
}

func Test_TemplateFromServer_UnknownKind(t *testing.T) {
	fmt.Println("Test CLI template --from-server with unknown kind")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-unknown-token")
	defer deleteTokenByName(tokenName)

	_, stderr, err := runCommandWithToken(token, "template", "UnknownKindThatDoesNotExist", "--from-server")

	// Should fail for unknown kind
	assert.Error(t, err, "Expected command to fail for unknown kind")
	assert.NotEmptyf(t, stderr, "Expected stderr to have error message, got empty")
}

func Test_TemplateFromServer_Help(t *testing.T) {
	fmt.Println("Test CLI template --from-server help")
	// Help command doesn't require authentication
	stdout, stderr, err := runConsoleCommand("template", "--help")

	// Help should always work regardless of server support
	assert.NoErrorf(t, err, "Help command failed: %v\nStderr: %s", err, stderr)

	// Should show usage information
	combinedOutput := stdout + stderr
	assert.Containsf(t, combinedOutput, "template", "Expected output to contain 'template', got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--from-server", "Expected output to contain '--from-server' flag, got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--output", "Expected output to contain '--output' flag, got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--edit", "Expected output to contain '--edit' flag, got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--apply", "Expected output to contain '--apply' flag, got: %s", combinedOutput)
}

func Test_TemplateFromServer_TemplateIsValidYAML(t *testing.T) {
	fmt.Println("Test CLI template --from-server returns valid YAML")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-yaml-token")
	defer deleteTokenByName(tokenName)

	stdout, stderr, err := runCommandWithToken(token, "template", "Group", "--from-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Parse the YAML to verify it's valid
	docs := parseStdoutAsYAMLDocuments(t, stdout)
	assert.NotEmpty(t, docs, "Expected at least one YAML document")

	// First document should have required fields
	if len(docs) > 0 {
		doc := docs[0]
		assert.Contains(t, doc, "kind", "Expected YAML to contain 'kind' field")
		assert.Contains(t, doc, "metadata", "Expected YAML to contain 'metadata' field")
	}
}

func Test_TemplateFromServer_CompareWithLocalTemplate(t *testing.T) {
	fmt.Println("Test CLI template --from-server vs local template command")

	// Create admin token for this test
	token, tokenName := createAdminToken(t, "template-fromserver-compare-token")
	defer deleteTokenByName(tokenName)

	// Get template from server
	serverStdout, serverStderr, serverErr := runCommandWithToken(token, "template", "Group", "--from-server")

	// Get template from local catalog (also needs token for API access)
	localStdout, localStderr, localErr := runCommandWithToken(token, "template", "Group")

	// Both should succeed
	assert.NoErrorf(t, serverErr, "Server template command failed: %v\nStderr: %s", serverErr, serverStderr)
	assert.NoErrorf(t, localErr, "Local template command failed: %v\nStderr: %s", localErr, localStderr)

	// Both should return valid YAML with 'kind' field
	assert.Contains(t, serverStdout, "kind:", "Server template should contain 'kind:'")
	assert.Contains(t, localStdout, "kind:", "Local template should contain 'kind:'")

	// Note: The actual content may differ as server templates may be more up-to-date
	// We don't assert equality, just that both are valid
}
