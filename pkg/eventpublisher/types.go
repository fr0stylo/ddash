package eventpublisher

import (
	"net/http"
	"time"
)

type Client struct {
	Endpoint   string
	Token      string
	Secret     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Event struct {
	Type        string
	Source      string
	Service     string
	Environment string
	Artifact    string
	SubjectID   string
	SubjectType string
	ChainID     string
	ActorName   string
	PipelineRun string
	PipelineURL string
}
