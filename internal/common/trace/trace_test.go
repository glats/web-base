package trace

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTraceId(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "empty context", ctx: context.Background(), want: ""},
		{name: "value of another type", ctx: context.WithValue(context.Background(), TraceIDKey, 42), want: ""},
		{name: "string value", ctx: context.WithValue(context.Background(), TraceIDKey, "abc"), want: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetTraceId(tt.ctx))
		})
	}
}

func TestAttachTraceIdToContext(t *testing.T) {
	first := GetTraceId(AttachTraceIdToContext(context.Background()))
	second := GetTraceId(AttachTraceIdToContext(context.Background()))

	assert.True(t, len(first) > len(prefix))
	assert.Contains(t, first, prefix)
	assert.NotEqual(t, first, second)
}

func TestGenerateTraceIdFormat(t *testing.T) {
	next := NextRequestId()

	assert.Equal(t, fmt.Sprintf("%s-%06d", prefix, next+1), GenerateTraceId())
}

func TestNextRequestIdIncrements(t *testing.T) {
	first := NextRequestId()

	assert.Equal(t, first+1, NextRequestId())
}
