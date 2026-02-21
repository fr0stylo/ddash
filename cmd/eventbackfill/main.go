package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
	"github.com/joho/godotenv"

	"github.com/fr0stylo/ddash/internal/config"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	cfg, err := config.LoadForTool()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dbPath := flag.String("db", cfg.Database.Path, "database path without .sqlite suffix")
	dryRun := flag.Bool("dry-run", false, "build and validate events without writing to event_store")
	source := flag.String("source", "ddash/backfill", "CDEvent source used for backfilled events")
	flag.Parse()

	database, err := db.New(strings.TrimSpace(*dbPath))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	org, err := getOrCreateDefaultOrganization(ctx, database)
	if err != nil {
		log.Fatalf("resolve organization: %v", err)
	}

	rows, err := database.ListLegacyDeploymentsForBackfill(ctx)
	if err != nil {
		log.Fatalf("list legacy deployments: %v", err)
	}

	converted := 0
	for _, row := range rows {
		event, err := serviceEventForStatus(strings.TrimSpace(row.Status))
		if err != nil {
			log.Printf("skip deployment id=%d status=%q: %v", row.ID, row.Status, err)
			continue
		}

		eventID := fmt.Sprintf("backfill-deployment-%d", row.ID)
		event.SetId(eventID)
		event.SetSource(*source)
		event.SetTimestamp(parseTimestamp(row.DeployedAt))
		event.SetSubjectId("service/" + row.Service)
		event.SetSubjectEnvironment(&cdeventsapi.Reference{Id: row.Environment})
		event.SetSubjectArtifactId(artifactForBackfill(row.Service, row.ReleaseRef, row.ID))

		if err := cdeventsapi.Validate(event); err != nil {
			log.Printf("skip invalid backfill event id=%s: %v", eventID, err)
			continue
		}

		raw, err := cdeventsapi.AsJsonString(event)
		if err != nil {
			log.Printf("skip serialization failure id=%s: %v", eventID, err)
			continue
		}

		if !*dryRun {
			err = database.AppendEventStore(ctx, queries.AppendEventStoreParams{
				OrganizationID: org.ID,
				EventID:        event.GetId(),
				EventType:      event.GetType().String(),
				EventSource:    event.GetSource(),
				EventTimestamp: event.GetTimestamp().UTC().Format(time.RFC3339Nano),
				EventTsMs:      event.GetTimestamp().UTC().UnixMilli(),
				SubjectID:      event.GetSubjectId(),
				SubjectSource:  sql.NullString{String: event.GetSubjectSource(), Valid: strings.TrimSpace(event.GetSubjectSource()) != ""},
				SubjectType:    event.GetType().Subject,
				ChainID:        sql.NullString{},
				RawEventJson:   raw,
			})
			if err != nil {
				log.Printf("skip append failure id=%s: %v", eventID, err)
				continue
			}
		}

		converted++
	}

	mode := "written"
	if *dryRun {
		mode = "validated"
	}
	fmt.Printf("Backfill complete: %d events %s from %d legacy deployments\n", converted, mode, len(rows))

	if err := database.Close(); err != nil {
		log.Fatalf("close db: %v", err)
	}
}

func serviceEventForStatus(status string) (serviceEventWriter, error) {
	switch strings.ToLower(status) {
	case "success":
		return cdeventsv05.NewServiceDeployedEvent()
	case "error":
		return cdeventsv05.NewServiceRolledbackEvent()
	case "processing", "queued":
		return cdeventsv05.NewServiceDeployedEvent()
	default:
		return cdeventsv05.NewServiceDeployedEvent()
	}
}

type serviceEventWriter interface {
	cdeventsapi.CDEventReader
	SetId(string)
	SetSource(string)
	SetTimestamp(time.Time)
	SetSubjectId(string)
	SetSubjectEnvironment(*cdeventsapi.Reference)
	SetSubjectArtifactId(string)
}

func parseTimestamp(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}

func artifactForBackfill(service string, releaseRef sql.NullString, deploymentID int64) string {
	if releaseRef.Valid && strings.TrimSpace(releaseRef.String) != "" {
		return fmt.Sprintf("pkg:generic/%s@%s", sanitizePurlName(service), strings.TrimSpace(releaseRef.String))
	}
	return fmt.Sprintf("pkg:generic/%s@backfill-%d", sanitizePurlName(service), deploymentID)
}

func sanitizePurlName(value string) string {
	value = strings.TrimSpace(value)
	clean := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			clean.WriteRune(r)
			continue
		}
		clean.WriteRune('-')
	}
	name := strings.Trim(clean.String(), "-.")
	if name == "" {
		name = "service"
	}
	return name
}

func getOrCreateDefaultOrganization(ctx context.Context, database *db.Database) (queries.Organization, error) {
	org, err := database.GetDefaultOrganization(ctx)
	if err == nil {
		return org, nil
	}
	if !errorsIsNoRows(err) {
		return queries.Organization{}, err
	}

	authToken, err := randomHexToken(16)
	if err != nil {
		return queries.Organization{}, err
	}
	secret, err := randomHexToken(24)
	if err != nil {
		return queries.Organization{}, err
	}

	return database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "default",
		AuthToken:     authToken,
		WebhookSecret: secret,
		Enabled:       1,
	})
}

func randomHexToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
