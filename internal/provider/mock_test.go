package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	metronome "github.com/Metronome-Industries/metronome-go/v3"
	"github.com/Metronome-Industries/metronome-go/v3/option"
)

// recordedCall is one request the mock server saw: "METHOD /path" plus the raw body.
type recordedCall struct {
	Route string
	Body  []byte
}

// newMockClient starts an httptest server, points a Metronome client at it with
// retries disabled, and records every request. Resource tests use it to prove
// that Read hits only read paths: assert on Route ("POST /v1/x/get"), not on the
// verb alone, because most Metronome reads are POSTs. The server is torn down
// with the test.
func newMockClient(t *testing.T, handler http.Handler) (*metronome.Client, func() []recordedCall) {
	t.Helper()
	var mu sync.Mutex
	var calls []recordedCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, recordedCall{Route: r.Method + " " + r.URL.Path, Body: body})
		mu.Unlock()
		if handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c := metronome.NewClient(
		option.WithBearerToken("test-token"),
		option.WithBaseURL(srv.URL),
		option.WithMaxRetries(0),
	)
	return &c, func() []recordedCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]recordedCall(nil), calls...)
	}
}

// jsonHandler answers every request with the same status and JSON body.
func jsonHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

// routes returns just the "METHOD /path" strings of the recorded calls.
func routes(calls []recordedCall) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.Route)
	}
	return out
}

// bodyJSON decodes a recorded request body into a generic map for assertions.
func bodyJSON(t *testing.T, c recordedCall) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(c.Body, &m); err != nil {
		t.Fatalf("decode body %q: %v", c.Body, err)
	}
	return m
}
