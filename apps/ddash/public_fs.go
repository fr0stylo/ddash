package main

import "embed"

// PublicFS embeds compiled static assets for ddash.
//
//go:embed public
var publicFS embed.FS
