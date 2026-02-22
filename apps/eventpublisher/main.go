package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/fr0stylo/ddash/pkg/eventpublisher"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintln(os.Stderr, "no .env file loaded:", err)
	}
	v := viper.New()
	v.AutomaticEnv()

	endpoint := flag.String("endpoint", strings.TrimSpace(v.GetString("DDASH_ENDPOINT")), "DDash base URL (or DDASH_ENDPOINT)")
	token := flag.String("token", strings.TrimSpace(v.GetString("DDASH_AUTH_TOKEN")), "Org auth token (or DDASH_AUTH_TOKEN)")
	secret := flag.String("secret", strings.TrimSpace(v.GetString("DDASH_WEBHOOK_SECRET")), "Org webhook secret (or DDASH_WEBHOOK_SECRET)")
	eventType := flag.String("type", "service.deployed", "Event type")
	service := flag.String("service", "", "Service name")
	environment := flag.String("environment", "", "Environment")
	artifact := flag.String("artifact", "", "Artifact id (optional)")
	subjectID := flag.String("subject-id", "", "Override subject id (optional)")
	subjectType := flag.String("subject-type", "", "Override subject type (optional)")
	chainID := flag.String("chain-id", "", "Chain identifier (optional)")
	actorName := flag.String("actor", "", "Actor name (optional)")
	pipelineRun := flag.String("pipeline-run", "", "Pipeline run id (optional)")
	pipelineURL := flag.String("pipeline-url", "", "Pipeline run URL (optional)")
	source := flag.String("source", strings.TrimSpace(v.GetString("DDASH_EVENT_SOURCE")), "Event source")
	timeout := flag.Duration("timeout", 10*time.Second, "Request timeout")
	flag.Parse()
	if strings.TrimSpace(*source) == "" {
		*source = "ci/pipeline"
	}

	if strings.TrimSpace(*endpoint) == "" || strings.TrimSpace(*token) == "" || strings.TrimSpace(*secret) == "" {
		exitErr("endpoint/token/secret are required (or set DDASH_ENDPOINT, DDASH_AUTH_TOKEN, DDASH_WEBHOOK_SECRET)")
	}

	client := eventpublisher.Client{
		Endpoint: strings.TrimSpace(*endpoint),
		Token:    strings.TrimSpace(*token),
		Secret:   strings.TrimSpace(*secret),
		Timeout:  *timeout,
	}
	resolvedType, err := client.Publish(context.Background(), eventpublisher.Event{
		Type:        strings.TrimSpace(*eventType),
		Source:      strings.TrimSpace(*source),
		Service:     strings.TrimSpace(*service),
		Environment: strings.TrimSpace(*environment),
		Artifact:    strings.TrimSpace(*artifact),
		SubjectID:   strings.TrimSpace(*subjectID),
		SubjectType: strings.TrimSpace(*subjectType),
		ChainID:     strings.TrimSpace(*chainID),
		ActorName:   strings.TrimSpace(*actorName),
		PipelineRun: strings.TrimSpace(*pipelineRun),
		PipelineURL: strings.TrimSpace(*pipelineURL),
	})
	if err != nil {
		exitErr(err.Error())
	}

	fmt.Printf("Published %s for service=%s env=%s\n", resolvedType, strings.TrimSpace(*service), strings.TrimSpace(*environment))
}

func exitErr(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
