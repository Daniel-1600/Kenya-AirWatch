package main

import (
	"context"
	"fmt"
	"github.com/dandora-airwatch/backend/internal/config"
	"github.com/dandora-airwatch/backend/internal/database"
	"github.com/dandora-airwatch/backend/internal/models"
	"github.com/dandora-airwatch/backend/internal/repository"
	"github.com/dandora-airwatch/backend/internal/services/anomaly"
	"github.com/dandora-airwatch/backend/internal/services/satellite"
	"os"
	"strconv"
	"time"
)

func main() {
	ctx := context.Background()
	url := config.Get("DATABASE_URL", "postgres://airwatch:airwatch@localhost:5432/airwatch?sslmode=disable")
	days, _ := strconv.Atoi(config.Get("INGEST_DAYS", "90"))
	to := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	from := to.AddDate(0, 0, -days)
	mode := config.Get("SATELLITE_MODE", "fixture")
	var provider satellite.Provider
	if mode == "earthengine" {
		provider = satellite.EarthEngineProvider{Project: os.Getenv("EE_PROJECT")}
	} else {
		provider = satellite.FixtureProvider{}
	}
	db, err := database.Open(ctx, url)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db); err != nil {
		panic(err)
	}
	repo := repository.Repo{DB: db}
	obs, err := provider.Fetch(-1.2467, 36.9068, 4.5, from, to)
	if err != nil {
		panic(err)
	}
	var prev []float64
	for _, o := range obs {
		m := models.Observation{SiteID: 1, Pollutant: "CH4", ObservedAt: o.ObservedAt, Value: o.Value, Unit: o.Unit, Source: o.Source}
		inserted, err := repo.InsertObservation(ctx, 1, m)
		if err != nil {
			panic(err)
		}
		score := anomaly.Score(o.Value, prev)
		prev = append(prev, o.Value)
		if score.Severity != "normal" {
			if err := repo.UpsertAnomaly(ctx, inserted, score.Baseline, score.DeviationPercent, score.ZScore, score.Severity); err != nil {
				panic(err)
			}
		}
		fmt.Printf("%s %.1f %s baseline=%.1f z=%.2f\n", o.ObservedAt.Format("2006-01-02"), o.Value, score.Severity, score.Baseline, score.ZScore)
	}
	fmt.Printf("ingested %d observations in %s mode\n", len(obs), mode)
}
