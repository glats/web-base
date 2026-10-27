package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingFormatAndLevel(t *testing.T) {
	previous, previousSlog, previousLevel := log.Logger, slog.Default(), zerolog.GlobalLevel()
	t.Cleanup(func() {
		log.Logger = previous
		slog.SetDefault(previousSlog)
		zerolog.SetGlobalLevel(previousLevel)
	})
	for _, pretty := range []bool{false, true} {
		for _, level := range []string{"debug", "info", "error", "disabled"} {
			t.Run(level+map[bool]string{false: "/json", true: "/pretty"}[pretty], func(t *testing.T) {
				var output bytes.Buffer
				require.NoError(t, Configure(level, pretty, &output))
				log.Debug().Msg("zero-debug")
				slog.Debug("slog-debug")
				log.Info().Msg("zero-info")
				slog.Info("slog-info")
				log.Error().Msg("zero-error")
				slog.Error("slog-error")
				for _, stack := range []string{"zero", "slog"} {
					assert.Equal(t, level == "debug", bytes.Contains(output.Bytes(), []byte(stack+"-debug")))
					assert.Equal(t, level == "debug" || level == "info", bytes.Contains(output.Bytes(), []byte(stack+"-info")))
					assert.Equal(t, level != "disabled", bytes.Contains(output.Bytes(), []byte(stack+"-error")))
				}
				for _, line := range bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n")) {
					if len(line) != 0 {
						assert.Equal(t, !pretty, json.Valid(line), string(line))
					}
				}
			})
		}
	}
}

func TestLoggingRejectsInvalidLevelWithoutChangingLoggers(t *testing.T) {
	previous, previousSlog := log.Logger, slog.Default()
	var output bytes.Buffer
	assert.Error(t, Configure("unknown", false, &output))
	assert.Equal(t, previous, log.Logger)
	assert.Same(t, previousSlog, slog.Default())
}
