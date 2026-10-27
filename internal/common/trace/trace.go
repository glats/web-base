package trace

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync/atomic"
)

type ctxKeyTraceID int

// TraceIDKey is the key that holds the unique request ID in a request context.
const TraceIDKey ctxKeyTraceID = 0

var prefix = makePrefix()

var sequence atomic.Uint64

func makePrefix() string {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	return hostname + "/" + rand.Text()[:10]
}

// GetTraceId returns a request ID from the given context if one is present.
// Returns the empty string if a request ID cannot be found.
func GetTraceId(ctx context.Context) string {
	if ctx != nil {
		value, _ := ctx.Value(TraceIDKey).(string)
		return value
	}
	return ""
}

// NextRequestId generates the next request ID in the sequence.
func NextRequestId() uint64 {
	return sequence.Add(1)
}

func GenerateTraceId() string {
	return fmt.Sprintf("%s-%06d", prefix, NextRequestId())
}

// AttachTraceIdToContext attach listener ID to the current context
func AttachTraceIdToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, TraceIDKey, GenerateTraceId())
}
