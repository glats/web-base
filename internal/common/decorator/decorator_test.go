package decorator

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"app.ddd/internal/common/metric"
)

var _ MetricsClient = metric.NoOp{}

func TestMain(m *testing.M) {
	log.Logger = zerolog.Nop()
	os.Exit(m.Run())
}

type recordingMetrics struct {
	keys []string
}

func (r *recordingMetrics) Inc(key string, _ int) {
	r.keys = append(r.keys, key)
}

type doThing struct{ ID string }

type askThing struct{ ID string }

type commandStub struct{ err error }

func (c commandStub) Handle(_ context.Context, _ doThing) error { return c.err }

type queryStub struct {
	result string
	err    error
}

func (q queryStub) Handle(_ context.Context, _ askThing) (string, error) { return q.result, q.err }

func TestApplyCommandDecorators(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name     string
		err      error
		wantKeys []string
	}{
		{name: "success", err: nil, wantKeys: []string{"commands.dothing.duration", "commands.dothing.success"}},
		{name: "failure", err: boom, wantKeys: []string{"commands.dothing.duration", "commands.dothing.failure"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &recordingMetrics{}
			handler := ApplyCommandDecorators[doThing](commandStub{err: tt.err}, metrics)

			err := handler.Handle(context.Background(), doThing{ID: "1"})

			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.wantKeys, metrics.keys)
		})
	}
}

func TestApplyQueryDecorators(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name       string
		stub       queryStub
		wantResult string
		wantKeys   []string
	}{
		{name: "success", stub: queryStub{result: "ok"}, wantResult: "ok", wantKeys: []string{"querys.askthing.duration", "querys.askthing.success"}},
		{name: "failure", stub: queryStub{err: boom}, wantResult: "", wantKeys: []string{"querys.askthing.duration", "querys.askthing.failure"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &recordingMetrics{}
			handler := ApplyQueryDecorators[askThing, string](tt.stub, metrics)

			result, err := handler.Handle(context.Background(), askThing{ID: "1"})

			assert.Equal(t, tt.stub.err, err)
			assert.Equal(t, tt.wantResult, result)
			assert.Equal(t, tt.wantKeys, metrics.keys)
		})
	}
}

func TestGenerateActionName(t *testing.T) {
	assert.Equal(t, "doThing", generateActionName(doThing{}))
}
