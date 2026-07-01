// This module handles basic configuration of HTTP server. It provides a function for reading environment variables and can set up a logger.
package httpconfig

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
)

// A configuration struct for the logger, populated from environmental variables.
type CfgLog struct {
	level  slog.Level
	source bool
}

// A configuration struct for the HTTP server, populated from environmental variables.
type CfgServer struct {
	address string
	port    uint16
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

// This middleware logs http request data at debug level.
func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("request",
			slog.String("host", r.Header.Get("Host")),
			slog.String("ip", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("user-agent", r.Header.Get("User-Agent")),
		)
		next.ServeHTTP(w, r)
	})
}

// This method configures HTTP server by reading values from the following environmental variables:
// - ADDRESS: TCP address for the server to listen on. Defaults to 0.0.0.0
// - PORT - Specifies TCP port for the server to listen on. Defaults to 8080
func (cfg *CfgServer) Config() error {
	var err error
	cfg.address, err = EnvGet("ADDRESS", "0.0.0.0", func(s string) (string, error) {
		if net.ParseIP(s) == nil {
			return "", fmt.Errorf("not valid IP adress: %s", s)
		}
		return s, nil
	})
	if err != nil {
		return fmt.Errorf("cfgServer.config: %w", err)
	}

	cfg.port, err = EnvGet("PORT", 8080, func(s string) (uint16, error) {
		a, err := strconv.ParseUint(s, 10, 16)
		return uint16(a), err
	})
	if err != nil {
		return fmt.Errorf("cfgServer.config: %w", err)
	}

	return nil
}

// This method returns *http.Server based on provided mux and configuration parsed with Config method.
func (cfg *CfgServer) GetServer(mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.address, cfg.port),
		Handler: mux,
	}
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
