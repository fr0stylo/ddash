package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/fr0stylo/ddash/internal/config"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func main() {
	var (
		dbPath         string
		organizationID int64
		windowDays     int
		maxConcurrent  int
	)

	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	cfg, err := config.LoadForTool()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	flag.StringVar(&dbPath, "db", cfg.Database.Path, "database path without .sqlite suffix")
	flag.Int64Var(&organizationID, "org", 0, "organization id (0 resolves default organization)")
	flag.IntVar(&windowDays, "window-days", 30, "dashboard/event window in days")
	flag.IntVar(&maxConcurrent, "max-concurrent-users", 0, "observed maximum concurrent users")
	flag.Parse()

	ctx := context.Background()
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	if organizationID <= 0 {
		org, orgErr := database.GetDefaultOrganization(ctx)
		if orgErr != nil {
			if strings.Contains(strings.ToLower(orgErr.Error()), "no rows") {
				organizationID = 0
			} else {
				log.Fatalf("resolve default organization: %v", orgErr)
			}
		} else {
			organizationID = org.ID
		}
	}

	sinceMs := time.Now().UTC().AddDate(0, 0, -windowDays).UnixMilli()
	totalRows, err := database.CountEventStore(ctx)
	if err != nil {
		log.Fatalf("count event_store rows: %v", err)
	}

	windowRows := int64(0)
	dailyRows := make([]queries.ListEventStoreDailyVolumeRow, 0)
	if organizationID > 0 {
		windowRows, err = database.CountEventStoreByOrganizationSinceMs(ctx, queries.CountEventStoreByOrganizationSinceMsParams{
			OrganizationID: organizationID,
			SinceMs:        sinceMs,
		})
		if err != nil {
			log.Fatalf("count window rows: %v", err)
		}
		dailyRows, err = database.ListEventStoreDailyVolume(ctx, queries.ListEventStoreDailyVolumeParams{
			OrganizationID: organizationID,
			SinceMs:        sinceMs,
			Limit:          int64(windowDays),
		})
		if err != nil {
			log.Fatalf("list daily volume: %v", err)
		}
	}

	avgPerDay := float64(0)
	if windowDays > 0 {
		avgPerDay = float64(windowRows) / float64(windowDays)
	}

	if organizationID > 0 {
		fmt.Printf("Organization: %d\n", organizationID)
	} else {
		fmt.Printf("Organization: none configured (showing global totals only)\n")
	}
	fmt.Printf("event_store rows: %d\n", totalRows)
	if organizationID > 0 {
		fmt.Printf("events in last %dd: %d (avg %.2f/day)\n", windowDays, windowRows, avgPerDay)
	} else {
		fmt.Printf("events in last %dd: n/a (select org with -org)\n", windowDays)
	}
	if maxConcurrent > 0 {
		fmt.Printf("max concurrent users: %d\n", maxConcurrent)
	} else {
		fmt.Printf("max concurrent users: unknown (pass -max-concurrent-users)\n")
	}

	fmt.Printf("\nTop query candidates to monitor (UI-heavy):\n")
	for _, name := range []string{
		"ListDeploymentsFromEvents",
		"ListServiceInstancesFromEvents",
		"ListServiceInstancesByEnvFromEvents",
		"GetServiceLatestFromEvents",
		"ListDeploymentHistoryByServiceFromEvents",
	} {
		fmt.Printf("- %s\n", name)
	}

	if organizationID > 0 {
		fmt.Printf("\nDaily events (%s to now):\n", time.UnixMilli(sinceMs).UTC().Format("2006-01-02"))
		for _, row := range dailyRows {
			fmt.Printf("- %s: %d\n", toString(row.Day), row.Total)
		}
	}
}

func toString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
