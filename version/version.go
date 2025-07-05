package version

import _ "embed"

//go:embed version
var V string

var Name = "orly.dev"

var Description = "fast, simple nostr relay"
