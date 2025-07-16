// Package main is a nostr relay with a simple follow/mute list authentication
// scheme and the new HTTP REST based protocol. Configuration is via environment
// variables or an optional .env file.
package main

import (
	"fmt"
	"github.com/pkg/profile"
	"net/http"
	_ "net/http/pprof"
	"orly.dev/app/realy"
	"orly.dev/app/realy/options"
	"orly.dev/utils/chk"
	"orly.dev/utils/interrupt"
	"orly.dev/utils/log"
	"orly.dev/version"
	"os"

	"orly.dev/app"
	"orly.dev/app/config"
	"orly.dev/database"
	"orly.dev/utils/context"
	"orly.dev/utils/lol"
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
	log.I.Ln("log level", cfg.LogLevel)
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
	r := &app.Relay{C: cfg, Store: storage}
	go app.MonitorResources(c)
	var server *realy.Server
	serverParams := &realy.ServerParams{
		Ctx:            c,
		Cancel:         cancel,
		Rl:             r,
		DbPath:         cfg.DataDir,
		MaxLimit:       512, // Default max limit for events
		AuthRequired:   cfg.AuthRequired,
		PublicReadable: cfg.PublicReadable,
	}
	var opts []options.O
	if server, err = realy.NewServer(serverParams, opts...); chk.E(err) {
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
