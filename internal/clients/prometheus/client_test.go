package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientQuery(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/query", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "query=")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"service": "test"},
						"value": [1234567890, "0.5"]
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 10*time.Second)
	result, err := client.Query(context.Background(), "up")
	require.NoError(t, err)
	assert.Equal(t, 0.5, result)
}

func TestClientQueryNoResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 10*time.Second)
	result, err := client.Query(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0.0, result)
}

func TestClientQueryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, 10*time.Second)
	_, err := client.Query(context.Background(), "up")
	assert.Error(t, err)
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:9090", 30*time.Second)
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:9090", client.baseURL)
}
