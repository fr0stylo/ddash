package main

import "github.com/fr0stylo/ddash/apps/gitlabingestor/internal/gitlabbridge"

type ingestorRuntime struct {
	ingestorToken     string
	webhookPath       string
	webhookToken      string
	ddashEndpoint     string
	defaultConvertCfg gitlabbridge.ConvertConfig
}
