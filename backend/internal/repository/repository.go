package repository

import (
	"context"
	"github.com/dandora-airwatch/backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repo struct{ DB *pgxpool.Pool }

func (r Repo) Sites(ctx context.Context) ([]models.Site, error) {
	rows, err := r.DB.Query(ctx, `SELECT id,name,latitude,longitude,radius_km,created_at FROM sites ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Site
	for rows.Next() {
		var x models.Site
		if err := rows.Scan(&x.ID, &x.Name, &x.Latitude, &x.Longitude, &x.RadiusKM, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r Repo) Site(ctx context.Context, id int) (models.Site, error) {
	var x models.Site
	err := r.DB.QueryRow(ctx, `SELECT id,name,latitude,longitude,radius_km,created_at FROM sites WHERE id=$1`, id).Scan(&x.ID, &x.Name, &x.Latitude, &x.Longitude, &x.RadiusKM, &x.CreatedAt)
	return x, err
}
func (r Repo) Observations(ctx context.Context, siteID int, pollutant string, from, to time.Time) ([]models.Observation, error) {
	rows, err := r.DB.Query(ctx, `SELECT id,site_id,pollutant,observed_at,value,unit,source FROM observations WHERE site_id=$1 AND pollutant=$2 AND observed_at >= $3 AND observed_at < $4 ORDER BY observed_at`, siteID, pollutant, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Observation
	for rows.Next() {
		var x models.Observation
		if err := rows.Scan(&x.ID, &x.SiteID, &x.Pollutant, &x.ObservedAt, &x.Value, &x.Unit, &x.Source); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r Repo) InsertObservation(ctx context.Context, siteID int, o models.Observation) (models.Observation, error) {
	err := r.DB.QueryRow(ctx, `INSERT INTO observations(site_id,pollutant,observed_at,value,unit,source) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(site_id,pollutant,observed_at) DO UPDATE SET value=EXCLUDED.value,unit=EXCLUDED.unit,source=EXCLUDED.source RETURNING id`, siteID, o.Pollutant, o.ObservedAt, o.Value, o.Unit, o.Source).Scan(&o.ID)
	return o, err
}
func (r Repo) UpsertAnomaly(ctx context.Context, o models.Observation, b, d, z float64, severity string) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO anomalies(observation_id,baseline,deviation_percent,z_score,severity) VALUES($1,$2,$3,$4,$5) ON CONFLICT(observation_id) DO UPDATE SET baseline=$2,deviation_percent=$3,z_score=$4,severity=$5`, o.ID, b, d, z, severity)
	return err
}
func (r Repo) DeleteAnomaly(ctx context.Context, observationID int) error {
	_, err := r.DB.Exec(ctx, `DELETE FROM anomalies WHERE observation_id=$1`, observationID)
	return err
}
func (r Repo) Anomalies(ctx context.Context, siteID int) ([]models.Anomaly, error) {
	rows, err := r.DB.Query(ctx, `SELECT a.id,a.observation_id,o.observed_at,o.value,a.baseline,a.deviation_percent,a.z_score,a.severity FROM anomalies a JOIN observations o ON o.id=a.observation_id WHERE o.site_id=$1 ORDER BY o.observed_at DESC`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Anomaly
	for rows.Next() {
		var x models.Anomaly
		if err := rows.Scan(&x.ID, &x.ObservationID, &x.ObservedAt, &x.Value, &x.Baseline, &x.DeviationPercent, &x.ZScore, &x.Severity); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
