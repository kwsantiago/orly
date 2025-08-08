// Package main is a nostr relay with a simple follow/mute list authentication
// scheme and the new HTTP REST-based protocol. Configuration is via environment
// variables or an optional .env file.
package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/pkg/profile"
	app2 "orly.dev/pkg/app"
	"orly.dev/pkg/app/config"
	"orly.dev/pkg/app/relay"
	"orly.dev/pkg/app/relay/options"
	"orly.dev/pkg/database"
	"orly.dev/pkg/protocol/openapi"
	"orly.dev/pkg/protocol/servemux"
	"orly.dev/pkg/utils/chk"
	"orly.dev/pkg/utils/context"
	"orly.dev/pkg/utils/interrupt"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
	"orly.dev/pkg/version"
)

func main() {
	var err error
	var cfg *config.C
	if cfg, err = config.New(); chk.T(err) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
		}
		config.PrintHelp(cfg, os.Stderr)
		os.Exit(0)
	}
	log.I.F("starting %s %s", cfg.AppName, version.V)
	if config.GetEnv() {
		config.PrintEnv(cfg, os.Stdout)
		os.Exit(0)
	}
	if config.HelpRequested() {
		config.PrintHelp(cfg, os.Stderr)
		os.Exit(0)
	}
	lol.SetLogLevel(cfg.LogLevel)
	if cfg.Pprof != "" {
		switch cfg.Pprof {
		case "cpu":
			prof := profile.Start(profile.CPUProfile)
			defer prof.Stop()
		case "memory":
			prof := profile.Start(profile.MemProfile)
			defer prof.Stop()
		case "allocation":
			prof := profile.Start(profile.MemProfileAllocs)
			defer prof.Stop()
		}
	}
	c, cancel := context.Cancel(context.Bg())
	var storage *database.D
	if storage, err = database.New(
		c, cancel, cfg.DataDir, cfg.DbLogLevel,
	); chk.E(err) {
		os.Exit(1)
	}
	r := &app2.Relay{C: cfg, Store: storage}
	go app2.MonitorResources(c)
	var server *relay.Server
	serverParams := &relay.ServerParams{
		Ctx:      c,
		Cancel:   cancel,
		Rl:       r,
		DbPath:   cfg.DataDir,
		MaxLimit: 512, // Default max limit for events
		C:        cfg,
	}
	var opts []options.O
	serveMux := servemux.NewServeMux()

	// Add favicon handler
	serveMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
		http.ServeFile(w, r, "static/favicon.ico")
	})

	if server, err = relay.NewServer(
		serverParams, serveMux, opts...,
	); chk.E(err) {
		os.Exit(1)
	}
	openapi.New(
		server,
		cfg.AppName,
		version.V,
		version.Description,
		"/api",
		serveMux,
	)
	if err != nil {
		log.F.F("failed to create server: %v", err)
	}
	interrupt.AddHandler(func() { server.Shutdown() })
	if err = server.Start(cfg.Listen, cfg.Port); chk.E(err) {
		log.F.F("server terminated: %v", err)
	}
}
