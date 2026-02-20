package tempo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildQueries(t *testing.T) {
	assert.Equal(t, "{ resource.service.name = \"cart\" }", BuildServiceQuery("cart"))
	assert.Equal(t, "{ resource.service.name = \"login\" && duration > 500ms }", BuildSlowSpansQuery("login", 500))
	assert.Equal(t, "{ resource.service.name = \"checkout\" && status = \"error\" }", BuildErrorSpansQuery("checkout"))
}

func TestGetTracesByService(t *testing.T) {
	// Mock Tempo server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/search", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "q=%7B+resource.service.name+%3D+%22test-service%22+%7D")
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"traces": [
				{"traceID": "trace-123"},
				{"traceID": "trace-456"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, nil)
	traces, err := client.GetTracesByService(context.Background(), "test-service", time.Now().Add(-1*time.Hour), time.Now())
	
	require.NoError(t, err)
	assert.Len(t, traces, 2)
	assert.Equal(t, "trace-123", traces[0].TraceID)
	assert.Equal(t, "trace-456", traces[1].TraceID)
}

func TestGetTraceByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/traces/abc-123", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"traceID": "abc-123"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, nil)
	trace, err := client.GetTraceByID(context.Background(), "abc-123")
	
	require.NoError(t, err)
	assert.NotNil(t, trace)
}
