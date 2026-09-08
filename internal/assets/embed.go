// Package assets stores real, independently testable runtime source as embedded
// generation assets. Runtime files remain CLI-owned in generated services.
package assets

import "embed"

//go:embed runtime
var Runtime embed.FS
