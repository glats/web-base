package logger

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Configure(levelName string, pretty bool, output io.Writer) error {
	level, err := zerolog.ParseLevel(levelName)
	if err != nil {
		return fmt.Errorf("parse LOG_LEVEL: %w", err)
	}

	zerolog.SetGlobalLevel(level)
	options := &slog.HandlerOptions{Level: slog.Level((int(level) - 1) * 4)}
	if level == zerolog.Disabled {
		options.Level = slog.Level(100)
	}
	if pretty {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: output, NoColor: true}).With().Timestamp().Logger()
		slog.SetDefault(slog.New(slog.NewTextHandler(output, options)))
	} else {
		log.Logger = zerolog.New(output).With().Timestamp().Logger()
		slog.SetDefault(slog.New(slog.NewJSONHandler(output, options)))
	}

	return nil
}
