package version

import _ "embed"

//go:embed version
var V string

var Name = "realy"

var Description = "fast, simple nostr relay"
