// Package foam holds the defaults of the anotherhadi components: load Defaults as
// the lowest layer of the app's config (config.Options.Base).
package foam

import _ "embed"

//go:embed defaults.yaml
var Defaults []byte
