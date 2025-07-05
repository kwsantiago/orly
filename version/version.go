package version

import _ "embed"

//go:embed version
var V string

var Name = "orly"

var Description = "fast, simple nostr relay"

var URL = "https://orly.dev"
