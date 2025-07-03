package version

import _ "embed"

//go:embed version
var V string

var Name = "manifold"

var Description = "Reference implementation of the Manifold protocol"
