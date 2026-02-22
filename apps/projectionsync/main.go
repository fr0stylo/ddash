package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/fr0stylo/ddash/internal/config"
	"github.com/fr0stylo/ddash/internal/db"
)

func main() {
	var (
		dbPath         string
		organizationID int64
	)

	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	cfg, err := config.LoadForTool()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	flag.StringVar(&dbPath, "db", cfg.Database.Path, "database path without .sqlite suffix")
	flag.Int64Var(&organizationID, "org", 0, "organization id for stats output (rebuild is global)")
	flag.Parse()

	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	stats, err := database.RebuildServiceProjections(context.Background(), organizationID)
	if err != nil {
		log.Fatalf("rebuild projections: %v", err)
	}

	fmt.Printf("service projection rebuild complete\n")
	fmt.Printf("service_current_state rows: %d\n", stats.CurrentStateRows)
	fmt.Printf("service_env_state rows: %d\n", stats.EnvStateRows)
	fmt.Printf("service_delivery_stats_daily rows: %d\n", stats.DailyStatsRows)
	fmt.Printf("service_change_links rows: %d\n", stats.ChangeLinkRows)
}
