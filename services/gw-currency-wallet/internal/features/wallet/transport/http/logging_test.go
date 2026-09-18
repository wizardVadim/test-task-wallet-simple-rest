package wallet_http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging(t *testing.T) {
	for _, tt := range []struct {
		name  string
		code  int
		level string
	}{{"success", 201, "INFO"}, {"client error", 400, "WARN"}, {"server error", 500, "ERROR"}, {"implicit success", 0, "INFO"}} {
		t.Run(tt.name, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, nil))
			mux := http.NewServeMux()
			mux.HandleFunc("POST /test", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "preserved")
				if tt.code != 0 {
					w.WriteHeader(tt.code)
				}
				w.Write([]byte("response"))
				w.WriteHeader(503) // Repeated final status must not affect response or log.
			})
			req := httptest.NewRequest("POST", "/test?token=secret", strings.NewReader("private-body"))
			req.Header.Set("Authorization", "Bearer secret")
			w := httptest.NewRecorder()
			Logging(logger, mux).ServeHTTP(w, req)
			expected := tt.code
			if expected == 0 {
				expected = 200
			}
			if w.Code != expected || w.Body.String() != "response" || w.Header().Get("X-Test") != "preserved" {
				t.Fatalf("response changed: %v", w)
			}
			var record map[string]any
			if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["status"] != float64(expected) || record["level"] != tt.level || record["route"] != "POST /test" {
				t.Fatalf("unexpected log: %s", buffer.String())
			}
			if strings.Contains(buffer.String(), "secret") || strings.Contains(buffer.String(), "private-body") {
				t.Fatal("request data leaked")
			}
		})
	}
}

func TestLoggingPreservesPanic(t *testing.T) {
	var buffer bytes.Buffer
	handler := Logging(slog.New(slog.NewJSONHandler(&buffer, nil)), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("failure") }))
	defer func() {
		if recover() != "failure" {
			t.Error("panic was swallowed or changed")
		}
		var record map[string]any
		if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		if record["level"] != "ERROR" || record["status"] != float64(500) {
			t.Errorf("unexpected panic log: %s", buffer.String())
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}
