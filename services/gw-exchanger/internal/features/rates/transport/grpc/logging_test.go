package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogging(t *testing.T) {
	for _, tt := range []struct {
		code  codes.Code
		level string
	}{{codes.OK, "INFO"}, {codes.InvalidArgument, "WARN"}, {codes.Internal, "ERROR"}} {
		t.Run(tt.level, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, nil))
			wantErr := status.Error(tt.code, "secret diagnostic")
			response, err := Logging(logger)(context.Background(), "secret request", &grpcgo.UnaryServerInfo{FullMethod: "/exchange.ExchangeService/GetExchangeRates"}, func(context.Context, any) (any, error) { return "response", wantErr })
			if response != "response" || err != wantErr {
				t.Fatal("interceptor changed result")
			}
			var record map[string]any
			if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["level"] != tt.level || record["code"] != tt.code.String() || record["method"] != "/exchange.ExchangeService/GetExchangeRates" {
				t.Fatalf("unexpected log: %s", buffer.String())
			}
			if _, ok := record["duration_ms"].(float64); !ok {
				t.Fatal("duration missing")
			}
			if strings.Contains(buffer.String(), "secret") {
				t.Fatal("request or diagnostic leaked")
			}
		})
	}
}
