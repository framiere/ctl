package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conduktor/ctl/internal/cli"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateServer_ListKinds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/public/v1/resources/template", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, cli.ApiVersion, r.Header.Get(cli.ApiVersionHeader))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"Topic", "User", "Group", "Application"})
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	var kinds []string
	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		SetResult(&kinds).
		Get("/public/v1/resources/template")

	require.NoError(t, err)
	assert.False(t, resp.IsError())
	assert.Len(t, kinds, 4)
	assert.Contains(t, kinds, "Topic")
	assert.Contains(t, kinds, "User")
	assert.Contains(t, kinds, "Group")
	assert.Contains(t, kinds, "Application")
}

func TestTemplateServer_GetTemplate(t *testing.T) {
	expectedTemplate := `apiVersion: v2
kind: Topic
metadata:
  name: my-topic
  cluster: my-cluster
spec:
  partitions: 3
  replicationFactor: 1`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/public/v1/resources/template/Topic", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, cli.ApiVersion, r.Header.Get(cli.ApiVersionHeader))

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(expectedTemplate))
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template/Topic")

	require.NoError(t, err)
	assert.False(t, resp.IsError())
	assert.Equal(t, expectedTemplate, resp.String())
}

func TestTemplateServer_GetTemplate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/public/v1/resources/template/UnknownKind", r.URL.Path)

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Kind 'UnknownKind' not found"))
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template/UnknownKind")

	require.NoError(t, err)
	assert.True(t, resp.IsError())
	assert.Equal(t, http.StatusNotFound, resp.StatusCode())
}

func TestTemplateServer_ApiVersionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unsupported API version: 1.0"))
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template")

	require.NoError(t, err)
	assert.True(t, resp.IsError())
	assert.Contains(t, resp.String(), "Unsupported API version")
}

func TestTemplateServer_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template/Topic")

	require.NoError(t, err)
	assert.True(t, resp.IsError())
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
}

func TestTemplateServer_EmptyKindsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	var kinds []string
	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		SetResult(&kinds).
		Get("/public/v1/resources/template")

	require.NoError(t, err)
	assert.False(t, resp.IsError())
	assert.Empty(t, kinds)
}

func TestTemplateServer_GetTemplateWithSpecialCharacters(t *testing.T) {
	// Template with special YAML characters
	expectedTemplate := `apiVersion: v2
kind: Topic
metadata:
  name: "topic-with-special:chars"
  labels:
    description: "This has 'quotes' and \"double quotes\""
spec:
  config:
    retention.ms: "86400000"
    cleanup.policy: "compact,delete"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(expectedTemplate))
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template/Topic")

	require.NoError(t, err)
	assert.False(t, resp.IsError())
	assert.Equal(t, expectedTemplate, resp.String())
}

func TestTemplateServer_MultipleKindsReturned(t *testing.T) {
	allKinds := []string{
		"Topic",
		"User",
		"Group",
		"Application",
		"ApplicationInstance",
		"ApplicationInstancePermission",
		"Cluster",
		"KafkaCluster",
		"Subject",
		"Connector",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allKinds)
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	var kinds []string
	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		SetResult(&kinds).
		Get("/public/v1/resources/template")

	require.NoError(t, err)
	assert.False(t, resp.IsError())
	assert.Len(t, kinds, 10)

	// Verify all expected kinds are present
	for _, expected := range allKinds {
		assert.Contains(t, kinds, expected)
	}
}

func TestTemplateServer_RequestHeaders(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"Topic"})
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	_, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		SetHeader("Authorization", "Bearer test-token").
		Get("/public/v1/resources/template")

	require.NoError(t, err)

	// Verify headers were sent correctly
	assert.Equal(t, cli.ApiVersion, capturedHeaders.Get(cli.ApiVersionHeader))
	assert.Equal(t, "Bearer test-token", capturedHeaders.Get("Authorization"))
}

func TestTemplateServer_ResponseWithVersionHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set(cli.ApiVersionHeader, "1.1") // Server returns different version
		json.NewEncoder(w).Encode([]string{"Topic"})
	}))
	defer server.Close()

	restyClient := resty.New().SetBaseURL(server.URL)

	resp, err := restyClient.R().
		SetHeader(cli.ApiVersionHeader, cli.ApiVersion).
		Get("/public/v1/resources/template")

	require.NoError(t, err)
	assert.False(t, resp.IsError())

	// Check that we can detect version mismatch
	serverVersion := resp.Header().Get(cli.ApiVersionHeader)
	assert.Equal(t, "1.1", serverVersion)
	assert.NotEqual(t, cli.ApiVersion, serverVersion)
}
