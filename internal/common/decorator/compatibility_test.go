package decorator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"app.ddd/internal/common/trace"
)

type commandProbe func(context.Context, doThing) error

func (f commandProbe) Handle(ctx context.Context, value doThing) error { return f(ctx, value) }

type queryProbe func(context.Context, askThing) (string, error)

func (f queryProbe) Handle(ctx context.Context, value askThing) (string, error) {
	return f(ctx, value)
}

type metricProbe struct {
	keys   []string
	values []int
}

func (m *metricProbe) Inc(key string, value int) {
	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
}

func TestDecoratorObservableContract(t *testing.T) {
	previous, level := log.Logger, zerolog.GlobalLevel()
	t.Cleanup(func() { log.Logger = previous; zerolog.SetGlobalLevel(level) })
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	boom := errors.New("handler failure")
	ctx := context.WithValue(context.Background(), trace.TraceIDKey, "trace-contract")
	for _, kind := range []string{"command", "query"} {
		for _, handlerErr := range []error{nil, boom} {
			t.Run(kind+"/"+map[bool]string{true: "success", false: "failure"}[handlerErr == nil], func(t *testing.T) {
				var output bytes.Buffer
				log.Logger = zerolog.New(&output)
				metrics := &metricProbe{}
				var got error
				namespace, action, body := "commands", "doThing", `decorator.doThing{ID:"sample"}`
				if kind == "command" {
					got = ApplyCommandDecorators[doThing](commandProbe(func(received context.Context, value doThing) error {
						assert.Same(t, ctx, received)
						assert.Equal(t, "sample", value.ID)
						return handlerErr
					}), metrics).Handle(ctx, doThing{ID: "sample"})
				} else {
					namespace, action, body = "querys", "askThing", `decorator.askThing{ID:"sample"}`
					var result string
					result, got = ApplyQueryDecorators[askThing, string](queryProbe(func(received context.Context, value askThing) (string, error) {
						assert.Same(t, ctx, received)
						assert.Equal(t, "sample", value.ID)
						return "partial result", handlerErr
					}), metrics).Handle(ctx, askThing{ID: "sample"})
					assert.Equal(t, "partial result", result)
				}
				assert.True(t, got == handlerErr)
				outcome := map[bool]string{true: "success", false: "failure"}[handlerErr == nil]
				assert.Equal(t, []string{namespace + "." + strings.ToLower(action) + ".duration", namespace + "." + strings.ToLower(action) + "." + outcome}, metrics.keys)
				require.Len(t, metrics.values, 2)
				assert.GreaterOrEqual(t, metrics.values[0], 0)
				assert.Equal(t, 1, metrics.values[1])
				lines := strings.Split(strings.TrimSpace(output.String()), "\n")
				require.Len(t, lines, map[bool]int{true: 2, false: 1}[handlerErr == nil])
				for index, line := range lines {
					var event map[string]any
					require.NoError(t, json.Unmarshal([]byte(line), &event))
					message := "Executing " + kind
					if index == 1 {
						message = strings.ToUpper(kind[:1]) + kind[1:] + " executed successfully"
					}
					assert.Equal(t, map[string]any{"level": "debug", "traceId": "trace-contract", kind: action, kind + "_body": body, "message": message}, event)
				}
			})
		}
	}
}
