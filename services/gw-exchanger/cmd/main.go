package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"exchanger-app/internal/app"
	"exchanger-app/internal/core/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("exchanger stopped", "error", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("exchanger", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("c", "", "path to config.env (environment overrides file)")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Println("Usage: exchanger [-c config.env]")
			return nil
		}
		return fmt.Errorf("invalid arguments; usage: exchanger [-c config.env]")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments; usage: exchanger [-c config.env]")
	}
	var cfg config.Config
	var err error
	if *path == "" {
		cfg, err = config.Load()
	} else {
		cfg, err = config.LoadFile(*path)
	}
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "gw-exchanger")
	return app.RunWithConfig(cfg, logger)
}
