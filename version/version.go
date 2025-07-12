package version

import (
	_ "embed"
)

//go:embed version
var V string

var Description = "relay powered by the orly framework"

var URL = "https://orly"
