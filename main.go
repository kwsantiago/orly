// Package main is a nostr relay with a simple follow/mute list authentication
// scheme and the new HTTP REST based protocol. Configuration is via environment
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
	if cfg.Pprof {
		defer profile.Start(profile.MemProfile).Stop()
		go func() {
			chk.E(http.ListenAndServe("127.0.0.1:6060", nil))
		}()
	}
	c, cancel := context.Cancel(context.Bg())
	storage, err := database.New(c, cancel, cfg.DataDir, cfg.DbLogLevel)
	if chk.E(err) {
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
	if server, err = relay.NewServer(serverParams, opts...); chk.E(err) {
		os.Exit(1)
	}
	if err != nil {
		log.F.F("failed to create server: %v", err)
	}
	interrupt.AddHandler(func() { server.Shutdown() })
	if err = server.Start(cfg.Listen, cfg.Port); chk.E(err) {
		log.F.F("server terminated: %v", err)
	}
}
