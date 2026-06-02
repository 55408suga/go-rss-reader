package applog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestRequestIDRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requestID string
		want      string
	}{
		{name: "stores and reads a request id", requestID: "req-123", want: "req-123"},
		{name: "empty request id is not stored", requestID: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := WithRequestID(context.Background(), tc.requestID)
			if got := RequestID(ctx); got != tc.want {
				t.Errorf("RequestID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequestIDFromEmptyContexts(t *testing.T) {
	t.Parallel()

	if got := RequestID(context.Background()); got != "" {
		t.Errorf("RequestID(Background) = %q, want empty", got)
	}

	var nilCtx context.Context
	if got := RequestID(nilCtx); got != "" {
		t.Errorf("RequestID(nil) = %q, want empty", got)
	}
}

func TestWithContextAttachesRequestID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	ctx := WithRequestID(context.Background(), "req-1")
	WithContext(ctx, base).Info("hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log line: %v", err)
	}
	if entry["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1", entry["request_id"])
	}
}

func TestWithContextWithoutRequestIDOmitsAttr(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	WithContext(context.Background(), base).Info("hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log line: %v", err)
	}
	if _, ok := entry["request_id"]; ok {
		t.Error("request_id attribute should be absent when no request id is set")
	}
}

func TestWithContextNilBaseReturnsUsableLogger(t *testing.T) {
	t.Parallel()

	if WithContext(context.Background(), nil) == nil {
		t.Error("WithContext(nil base) = nil, want a usable logger")
	}
}
