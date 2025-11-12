package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/option"
)

// MockAPIHandler represents a handler function for mock API responses
type MockAPIHandler func(w http.ResponseWriter, r *http.Request)

// MockAPIServer provides a test HTTP server that mocks the Cloudflare API
type MockAPIServer struct {
	Server   *httptest.Server
	Client   *cloudflare.Client
	Handlers map[string]MockAPIHandler
}

// NewMockAPIServer creates a new mock API server for testing
func NewMockAPIServer() *MockAPIServer {
	m := &MockAPIServer{
		Handlers: make(map[string]MockAPIHandler),
	}

	// Create HTTP test server
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Look up handler by path (check if query params are included)
		path := r.URL.Path
		handler, ok := m.Handlers[path]
		if !ok {
			// Default response for unmocked endpoints
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"errors":  []map[string]interface{}{{"message": "endpoint not mocked: " + path}},
			})
			return
		}

		handler(w, r)
	}))

	// Create Cloudflare client pointing to our mock server
	m.Client = cloudflare.NewClient(
		option.WithBaseURL(m.Server.URL),
		option.WithAPIToken("test-token"),
	)

	return m
}

// Close shuts down the mock server
func (m *MockAPIServer) Close() {
	m.Server.Close()
}

// AddTunnelRoutesListHandler adds a mock handler for the tunnel routes list endpoint
// Example usage:
//   server.AddTunnelRoutesListHandler(accountID, []map[string]interface{}{
//       {"id": "uuid-123", "network": "10.0.0.0/16", ...},
//   })
func (m *MockAPIServer) AddTunnelRoutesListHandler(accountID string, routes []map[string]interface{}) {
	path := "/accounts/" + accountID + "/teamnet/routes"
	m.Handlers[path] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  routes,
		})
	}
}
