package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: These tests require the Conduktor Console server to support the
// template-server API endpoints (/public/v1/resources/template).

func Test_TemplateServer_ListKinds(t *testing.T) {
	fmt.Println("Test CLI template-server list kinds")
	stdout, stderr, err := runConsoleCommand("template-server")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should list available kinds
	assert.Containsf(t, stdout, "Available Kinds", "Expected stdout to contain 'Available Kinds', got: %s", stdout)
}

func Test_TemplateServer_GetTopicTemplate(t *testing.T) {
	fmt.Println("Test CLI template-server get Topic template")
	stdout, stderr, err := runConsoleCommand("template-server", "Topic")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template with Topic kind
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
	// Template should start with ---
	assert.True(t, strings.HasPrefix(strings.TrimSpace(stdout), "---"), "Expected stdout to start with '---', got: %s", stdout)
}

func Test_TemplateServer_GetGroupTemplate(t *testing.T) {
	fmt.Println("Test CLI template-server get Group template")
	stdout, stderr, err := runConsoleCommand("template-server", "Group")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
}

func Test_TemplateServer_GetUserTemplate(t *testing.T) {
	fmt.Println("Test CLI template-server get User template")
	stdout, stderr, err := runConsoleCommand("template-server", "User")

	assert.NoErrorf(t, err, "Command failed: %v\nStderr: %s", err, stderr)

	// Should contain YAML template
	assert.Containsf(t, stdout, "kind:", "Expected stdout to contain 'kind:', got: %s", stdout)
}

func Test_TemplateServer_UnknownKind(t *testing.T) {
	fmt.Println("Test CLI template-server with unknown kind")
	_, stderr, err := runConsoleCommand("template-server", "UnknownKindThatDoesNotExist")

	// Should fail for unknown kind
	assert.Error(t, err, "Expected command to fail for unknown kind")
	assert.NotEmptyf(t, stderr, "Expected stderr to have error message, got empty")
}

func Test_TemplateServer_Help(t *testing.T) {
	fmt.Println("Test CLI template-server help")
	stdout, stderr, err := runConsoleCommand("template-server", "--help")

	// Help should always work regardless of server support
	assert.NoErrorf(t, err, "Help command failed: %v\nStderr: %s", err, stderr)

	// Should show usage information
	combinedOutput := stdout + stderr
	assert.Containsf(t, combinedOutput, "template-server", "Expected output to contain 'template-server', got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--output", "Expected output to contain '--output' flag, got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--edit", "Expected output to contain '--edit' flag, got: %s", combinedOutput)
	assert.Containsf(t, combinedOutput, "--apply", "Expected output to contain '--apply' flag, got: %s", combinedOutput)
}

func Test_TemplateServer_TemplateIsValidYAML(t *testing.T) {
	fmt.Println("Test CLI template-server returns valid YAML")
	stdout, stderr, err := runConsoleCommand("template-server", "Group")

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

func Test_TemplateServer_CompareWithLocalTemplate(t *testing.T) {
	fmt.Println("Test CLI template-server vs local template command")

	// Get template from server
	serverStdout, serverStderr, serverErr := runConsoleCommand("template-server", "Group")

	// Get template from local catalog
	localStdout, localStderr, localErr := runConsoleCommand("template", "Group")

	// Both should succeed
	assert.NoErrorf(t, serverErr, "Server template command failed: %v\nStderr: %s", serverErr, serverStderr)
	assert.NoErrorf(t, localErr, "Local template command failed: %v\nStderr: %s", localErr, localStderr)

	// Both should return valid YAML with 'kind' field
	assert.Contains(t, serverStdout, "kind:", "Server template should contain 'kind:'")
	assert.Contains(t, localStdout, "kind:", "Local template should contain 'kind:'")

	// Note: The actual content may differ as server templates may be more up-to-date
	// We don't assert equality, just that both are valid
}
