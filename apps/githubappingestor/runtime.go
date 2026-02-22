package main

import (
	"github.com/fr0stylo/ddash/apps/githubappingestor/internal/githubbridge"
)

type ingestorRuntime struct {
	ingestorToken     string
	webhookPath       string
	githubSecret      string
	ddashEndpoint     string
	defaultConvertCfg githubbridge.ConvertConfig
}
