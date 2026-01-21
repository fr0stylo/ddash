package ddash

import "embed"

// PublicFS embeds compiled static assets.
//
//go:embed public
var PublicFS embed.FS
