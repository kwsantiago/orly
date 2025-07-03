package main

import (
	"fmt"
	"github.com/pkg/profile"
	"net/http"
	"not.realy.lol/chk"
	"not.realy.lol/config"
	"not.realy.lol/log"
	"not.realy.lol/lol"
	"not.realy.lol/version"
	"os"
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
	if config.GetEnv() {
		config.PrintEnv(cfg, os.Stdout)
		os.Exit(0)
	}
	if config.HelpRequested() {
		config.PrintHelp(cfg, os.Stderr)
		os.Exit(0)
	}
	if cfg.Pprof {
		defer profile.Start(profile.MemProfile).Stop()
		go func() {
			chk.E(http.ListenAndServe("127.0.0.1:6060", nil))
		}()
	}
	log.I.F(
		"starting %s %s; log level: %s", version.Name, version.V, lol.GetLevel(),
	)

}
