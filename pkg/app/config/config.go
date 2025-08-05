// Package config provides a go-simpler.org/env configuration table and helpers
// for working with the list of key/value lists stored in .env files.
package config

import (
	"fmt"
	"io"
	"orly.dev/pkg/utils/apputil"
	"orly.dev/pkg/utils/chk"
	env2 "orly.dev/pkg/utils/env"
	"orly.dev/pkg/utils/log"
	"orly.dev/pkg/utils/lol"
	"orly.dev/pkg/version"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"go-simpler.org/env"
)

// C holds application configuration settings loaded from environment variables
// and default values. It defines parameters for app behaviour, storage
// locations, logging, and network settings used across the relay service.
type C struct {
	AppName            string        `env:"ORLY_APP_NAME" default:"orly"`
	Config             string        `env:"ORLY_CONFIG_DIR" usage:"location for configuration file, which has the name '.env' to make it harder to delete, and is a standard environment KEY=value<newline>... style" default:"~/.config/orly"`
	State              string        `env:"ORLY_STATE_DATA_DIR" usage:"storage location for state data affected by dynamic interactive interfaces" default:"~/.local/state/orly"`
	DataDir            string        `env:"ORLY_DATA_DIR" usage:"storage location for the event store" default:"~/.local/cache/orly"`
	Listen             string        `env:"ORLY_LISTEN" default:"0.0.0.0" usage:"network listen address"`
	Port               int           `env:"ORLY_PORT" default:"3334" usage:"port to listen on"`
	LogLevel           string        `env:"ORLY_LOG_LEVEL" default:"info" usage:"debug level: fatal error warn info debug trace"`
	DbLogLevel         string        `env:"ORLY_DB_LOG_LEVEL" default:"info" usage:"debug level: fatal error warn info debug trace"`
	Pprof              string        `env:"ORLY_PPROF" usage:"enable pprof on 127.0.0.1:6060" enum:"cpu,memory,allocation"`
	AuthRequired       bool          `env:"ORLY_AUTH_REQUIRED" default:"false" usage:"require authentication for all requests"`
	PublicReadable     bool          `env:"ORLY_PUBLIC_READABLE" default:"true" usage:"allow public read access to regardless of whether the client is authed"`
	SpiderSeeds        []string      `env:"ORLY_SPIDER_SEEDS" usage:"seeds to use for the spider (relays that are looked up initially to find owner relay lists) (comma separated)" default:"wss://profiles.nostr1.com/,wss://relay.nostr.band/,wss://relay.damus.io/,wss://nostr.wine/,wss://nostr.land/,wss://theforest.nostr1.com/,wss://profiles.nostr1.com/"`
	SpiderType         string        `env:"ORLY_SPIDER_TYPE" usage:"whether to spider, and what degree of spidering: none, directory, follows (follows means to the second degree of the follow graph)" default:"directory"`
	SpiderTime         time.Duration `env:"ORLY_SPIDER_FREQUENCY" usage:"how often to run the spider, uses notation 0h0m0s" default:"1h"`
	SpiderSecondDegree bool          `env:"ORLY_SPIDER_SECOND_DEGREE" default:"true" usage:"whether to enable spidering the second degree of follows for non-directory events if ORLY_SPIDER_TYPE is set to 'follows'"`
	Owners             []string      `env:"ORLY_OWNERS" usage:"list of users whose follow lists designate whitelisted users who can publish events, and who can read if public readable is false (comma separated)"`
	Private            bool          `env:"ORLY_PRIVATE" usage:"do not spider for user metadata because the relay is private and this would leak relay memberships" default:"false"`
	Whitelist          []string      `env:"ORLY_WHITELIST" usage:"only allow connections from this list of IP addresses"`
	RelaySecret        string        `env:"ORLY_SECRET_KEY" usage:"secret key for relay cluster replication authentication"`
	PeerRelays         []string      `env:"ORLY_PEER_RELAYS" usage:"list of peer relays URLs that new events are pushed to in format <pubkey>|<url>"`
}

// New creates and initializes a new configuration object for the relay
// application
//
// # Return Values
//
//   - cfg: A pointer to the initialized configuration struct containing default
//     or environment-provided values
//
//   - err: An error object that is non-nil if any operation during
//     initialization fails
//
// # Expected Behaviour:
//
// Initializes a new configuration instance by loading environment variables and
// checking for a .env file in the default configuration directory. Sets logging
// levels based on configuration values and returns the populated configuration
// or an error if any step fails
func New() (cfg *C, err error) {
	cfg = &C{}
	if err = env.Load(cfg, &env.Options{SliceSep: ","}); chk.T(err) {
		return
	}
	if cfg.Config == "" || strings.Contains(cfg.State, "~") {
		cfg.Config = filepath.Join(xdg.ConfigHome, cfg.AppName)
	}
	if cfg.DataDir == "" || strings.Contains(cfg.State, "~") {
		cfg.DataDir = filepath.Join(xdg.DataHome, cfg.AppName)
	}
	if cfg.State == "" || strings.Contains(cfg.State, "~") {
		cfg.State = filepath.Join(xdg.StateHome, cfg.AppName)
	}
	if len(cfg.Owners) > 0 {
		cfg.AuthRequired = true
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
	}
	// if spider seeds has no elements, there still is a single entry with an
	// empty string; and also if any of the fields are empty strings, they need
	// to be removed.
	var seeds []string
	for _, u := range cfg.SpiderSeeds {
		if u == "" {
			continue
		}
		seeds = append(seeds, u)
	}
	cfg.SpiderSeeds = seeds
	return
}

// HelpRequested determines if the command line arguments indicate a request for help
//
// # Return Values
//
//   - help: A boolean value indicating true if a help flag was detected in the
//     command line arguments, false otherwise
//
// # Expected Behaviour
//
// The function checks the first command line argument for common help flags and
// returns true if any of them are present. Returns false if no help flag is found
func HelpRequested() (help bool) {
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "help", "-h", "--h", "-help", "--help", "?":
			help = true
		}
	}
	return
}

// GetEnv checks if the first command line argument is "env" and returns
// whether the environment configuration should be printed.
//
// # Return Values
//
//   - requested: A boolean indicating true if the 'env' argument was
//     provided, false otherwise.
//
// # Expected Behaviour
//
// The function returns true when the first command line argument is "env"
// (case-insensitive), signalling that the environment configuration should be
// printed. Otherwise, it returns false.
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

// KVSlice is a sortable slice of key/value pairs, designed for managing
// configuration data and enabling operations like merging and sorting based on
// keys.
type KVSlice []KV

func (kv KVSlice) Len() int           { return len(kv) }
func (kv KVSlice) Less(i, j int) bool { return kv[i].Key < kv[j].Key }
func (kv KVSlice) Swap(i, j int)      { kv[i], kv[j] = kv[j], kv[i] }

// Compose merges two KVSlice instances into a new slice where key-value pairs
// from the second slice override any duplicate keys from the first slice.
//
// # Parameters
//
//   - kv2: The second KVSlice whose entries will be merged with the receiver.
//
// # Return Values
//
//   - out: A new KVSlice containing all entries from both slices, with keys
//     from kv2 taking precedence over keys from the receiver.
//
// # Expected Behaviour
//
// The method returns a new KVSlice that combines the contents of the receiver
// and kv2. If any key exists in both slices, the value from kv2 is used. The
// resulting slice remains sorted by keys as per the KVSlice implementation.
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

// EnvKV generates key/value pairs from a configuration object's struct tags
//
// # Parameters
//
//   - cfg: A configuration object whose struct fields are processed for env tags
//
// # Return Values
//
//   - m: A KVSlice containing key/value pairs derived from the config's env tags
//
// # Expected Behaviour
//
// Processes each field of the config object, extracting values tagged with
// "env" and converting them to strings. Skips fields without an "env" tag.
// Handles various value types including strings, integers, booleans, durations,
// and string slices by joining elements with commas.
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

// PrintEnv outputs sorted environment key/value pairs from a configuration object
// to the provided writer
//
// # Parameters
//
//   - cfg: Pointer to the configuration object containing env tags
//
//   - printer: Destination for the output, typically an io.Writer implementation
//
// # Expected Behaviour
//
// Outputs each environment variable derived from the config's struct tags in
// sorted order, formatted as "key=value\n" to the specified writer
func PrintEnv(cfg *C, printer io.Writer) {
	kvs := EnvKV(*cfg)
	sort.Sort(kvs)
	for _, v := range kvs {
		_, _ = fmt.Fprintf(printer, "%s=%s\n", v.Key, v.Value)
	}
}

// PrintHelp prints help information including application version, environment
// variable configuration, and details about .env file handling to the provided
// writer
//
// # Parameters
//
//   - cfg: Configuration object containing app name and config directory path
//
//   - printer: Output destination for the help text
//
// # Expected Behaviour
//
// Prints application name and version followed by environment variable
// configuration details, explains .env file behaviour including automatic
// loading and custom path options, and displays current configuration values
// using PrintEnv. Outputs all information to the specified writer
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
			"set the environment using\n\n\t%s env > %s/.env\n",
		cfg.Config,
		os.Args[0],
		cfg.Config,
	)
	fmt.Fprintf(printer, "\ncurrent configuration:\n\n")
	PrintEnv(cfg, printer)
	fmt.Fprintln(printer)
	return
}
