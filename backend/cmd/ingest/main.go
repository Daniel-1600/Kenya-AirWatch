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
	sites, err := repo.Sites(ctx)
	if err != nil {
		panic(err)
	}
	total := 0
	for _, site := range sites {
		obs, err := provider.Fetch(site.Latitude, site.Longitude, site.RadiusKM, from, to)
		if err != nil {
			panic(fmt.Errorf("site %d (%s): %w", site.ID, site.Name, err))
		}
		var prev []float64
		for _, o := range obs {
			m := models.Observation{SiteID: site.ID, Pollutant: "CH4", ObservedAt: o.ObservedAt, Value: o.Value, Unit: o.Unit, Source: o.Source}
			inserted, err := repo.InsertObservation(ctx, site.ID, m)
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
			fmt.Printf("[%s] %s %.1f %s baseline=%.1f z=%.2f\n", site.Name, o.ObservedAt.Format("2006-01-02"), o.Value, score.Severity, score.Baseline, score.ZScore)
		}
		fmt.Printf("ingested %d observations for %s in %s mode\n", len(obs), site.Name, mode)
		total += len(obs)
	}
	fmt.Printf("ingested %d total observations across %d sites in %s mode\n", total, len(sites), mode)
}
