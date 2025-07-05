// Package config provides a go-simpler.org/env configuration table and helpers
// for working with the list of key/value lists stored in .env files.
package config

import (
	"fmt"
	"io"
	"orly.dev/chk"
	env2 "orly.dev/env"
	"orly.dev/log"
	"orly.dev/lol"
	"orly.dev/version"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"go-simpler.org/env"

	"orly.dev/apputil"
)

// C is the configuration for realy relay. These are read from the environment if present, or if
// a .env file is found in ~/.config/realy/ that is read instead and overrides anything else.
type C struct {
	AppName    string `env:"REALY_APP_NAME" default:"orly.dev"`
	Config     string `env:"REALY_CONFIG_DIR" usage:"location for configuration file, which has the name '.env' to make it harder to delete, and is a standard environment KEY=value<newline>... style"`
	State      string `env:"REALY_STATE_DATA_DIR" usage:"storage location for state data affected by dynamic interactive interfaces"`
	DataDir    string `env:"REALY_DATA_DIR" usage:"storage location for the ratel event store"`
	Listen     string `env:"REALY_LISTEN" default:"0.0.0.0" usage:"network listen address"`
	DNS        string `env:"REALY_DNS" usage:"external DNS name that points at the relay"`
	Port       int    `env:"REALY_PORT" default:"3334" usage:"port to listen on"`
	LogLevel   string `env:"REALY_LOG_LEVEL" default:"info" usage:"debug level: fatal error warn info debug trace"`
	DbLogLevel string `env:"REALY_DB_LOG_LEVEL" default:"info" usage:"debug level: fatal error warn info debug trace"`
	Pprof      bool   `env:"REALY_PPROF" default:"false" usage:"enable pprof on 127.0.0.1:6060"`
}

// New creates a new config.C.
func New() (cfg *C, err error) {
	cfg = &C{}
	if err = env.Load(cfg, &env.Options{SliceSep: ","}); chk.T(err) {
		return
	}
	if cfg.Config == "" {
		cfg.Config = filepath.Join(xdg.ConfigHome, cfg.AppName)
	}
	// if cfg.State == "" {
	//	cfg.State = filepath.Join(xdg.StateHome, cfg.AppName)
	// }
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(xdg.DataHome, cfg.AppName)
	}
	envPath := filepath.Join(cfg.Config, ".env")
	if apputil.FileExists(envPath) {
		var e env2.Env
		if e, err = env2.GetEnv(envPath); chk.T(err) {
			return
		}
		if err = env.Load(
			cfg, &env.Options{SliceSep: ",", Source: e},
		); chk.E(err) {
			return
		}
		lol.SetLogLevel(cfg.LogLevel)
		log.I.F("loaded configuration from %s", envPath)
		// var owners []string
		// // remove empties if any
		// for _, o := range cfg.Owners {
		//	if len(o) > 0 {
		//		owners = append(owners, o)
		//	}
		// }
		// cfg.Owners = owners
	}
	return
}

// HelpRequested returns true if any of the common types of help invocation are
// found as the first command line parameter/flag.
func HelpRequested() (help bool) {
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "help", "-h", "--h", "-help", "--help", "?":
			help = true
		}
	}
	return
}

// GetEnv processes os.Args to detect a request for printing the current settings as a list of
// environment variable key/values.
func GetEnv() (requested bool) {
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "env":
			requested = true
		}
	}
	return
}

// KV is a key/value pair.
type KV struct{ Key, Value string }

// KVSlice is a collection of key/value pairs.
type KVSlice []KV

func (kv KVSlice) Len() int           { return len(kv) }
func (kv KVSlice) Less(i, j int) bool { return kv[i].Key < kv[j].Key }
func (kv KVSlice) Swap(i, j int)      { kv[i], kv[j] = kv[j], kv[i] }

// Compose merges two KVSlice together, replacing the values of earlier keys with same named
// KV items later in the slice (enabling compositing two together as a .env, as well as them
// being composed as structs.
func (kv KVSlice) Compose(kv2 KVSlice) (out KVSlice) {
	// duplicate the initial KVSlice
	for _, p := range kv {
		out = append(out, p)
	}
out:
	for i, p := range kv2 {
		for j, q := range out {
			// if the key is repeated, replace the value
			if p.Key == q.Key {
				out[j].Value = kv2[i].Value
				continue out
			}
		}
		out = append(out, p)
	}
	return
}

// EnvKV turns a struct with `env` keys (used with go-simpler/env) into a standard formatted
// environment variable key/value pair list, one per line. Note you must dereference a pointer
// type to use this. This allows the composition of the config in this file with an extended
// form with a customized variant of realy to produce correct environment variables both read
// and write.
func EnvKV(cfg any) (m KVSlice) {
	t := reflect.TypeOf(cfg)
	for i := 0; i < t.NumField(); i++ {
		k := t.Field(i).Tag.Get("env")
		v := reflect.ValueOf(cfg).Field(i).Interface()
		var val string
		switch v.(type) {
		case string:
			val = v.(string)
		case int, bool, time.Duration:
			val = fmt.Sprint(v)
		case []string:
			arr := v.([]string)
			if len(arr) > 0 {
				val = strings.Join(arr, ",")
			}
		}
		// this can happen with embedded structs
		if k == "" {
			continue
		}
		m = append(m, KV{k, val})
	}
	return
}

// PrintEnv renders the key/values of a config.C to a provided io.Writer.
func PrintEnv(cfg *C, printer io.Writer) {
	kvs := EnvKV(*cfg)
	sort.Sort(kvs)
	for _, v := range kvs {
		_, _ = fmt.Fprintf(printer, "%s=%s\n", v.Key, v.Value)
	}
}

// PrintHelp outputs a help text listing the configuration options and default
// values to a provided io.Writer (usually os.Stderr or os.Stdout).
func PrintHelp(cfg *C, printer io.Writer) {
	_, _ = fmt.Fprintf(
		printer,
		"%s %s\n\n", cfg.AppName, version.V,
	)

	_, _ = fmt.Fprintf(
		printer,
		"Environment variables that configure %s:\n\n", cfg.AppName,
	)

	env.Usage(cfg, printer, &env.Options{SliceSep: ","})
	_, _ = fmt.Fprintf(
		printer,
		"\nCLI parameter 'help' also prints this information\n"+
			"\n.env file found at the path %s will be automatically "+
			"loaded for configuration.\nset these two variables for a custom load path,"+
			" this file will be created on first startup.\nenvironment overrides it and "+
			"you can also edit the file to set configuration options\n\n"+
			"use the parameter 'env' to print out the current configuration to the terminal\n\n"+
			"set the environment using\n\n\t%s env > %s/.env\n", os.Args[0],
		cfg.Config,
		cfg.Config,
	)

	fmt.Fprintf(printer, "\ncurrent configuration:\n\n")
	PrintEnv(cfg, printer)
	fmt.Fprintln(printer)
	return
}
