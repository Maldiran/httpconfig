// This module handles basic configuration. It provides a function for reading environment variables and can set up a logger.
package httpconfig

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
)

// A configuration struct for the logger, populated from environmental variables.
type CfgLog struct {
	level  slog.Level
	source bool
}

// A function used for reading environment variables. It accepts a variable name, a default value, and a parsing function.
// This parsing function accepts a string (the raw representation of the environment variable) and returns a parsed value and an error.
func EnvGet[T any](name string, def T, parse func(string) (T, error)) (T, error) {
	s, e := os.LookupEnv(name)
	if !e {
		return def, nil
	}
	v, err := parse(s)
	if err != nil {
		slog.Error("config error",
			slog.String("name", name),
			slog.String("value", s),
			slog.Any("err", err),
		)
		return def, fmt.Errorf("envGet: %w", err)
	}
	return v, nil
}

// This method configures logging by reading values from the following environmental variables:
// - LOG_LEVEL: Minimum log level for emitted messages. Can be any value of slog.Level.
// - LOG_SOURCE - boolean value - defines if log should contain source code position of the log statement.
func (cfg *CfgLog) Config() error {
	// Default logging in case below code fails
	logSetup(os.Stdout, slog.HandlerOptions{})
	var err error
	cfg.level, err = EnvGet("LOG_LEVEL", slog.LevelInfo, func(s string) (slog.Level, error) {
		var l slog.Level
		if err := l.UnmarshalText([]byte(s)); err != nil {
			return l, err
		}
		return l, nil
	})
	if err != nil {
		return fmt.Errorf("logConfig: %w", err)
	}

	cfg.source, err = EnvGet("LOG_SOURCE", false, strconv.ParseBool)
	if err != nil {
		return fmt.Errorf("cfgLog.config: %w", err)
	}

	logSetup(os.Stdout, cfg.handler())
	return nil
}

func logSetup(w io.Writer, h slog.HandlerOptions) {
	handler := slog.NewJSONHandler(w, &h)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func (cfg *CfgLog) handler() slog.HandlerOptions {
	return slog.HandlerOptions{
		AddSource: cfg.source,
		Level:     cfg.level,
	}
}
