package version

import _ "embed"

//go:embed version
var V string

var Name = "not.realy.lol"

var Description = "fast, simple nostr relay"
