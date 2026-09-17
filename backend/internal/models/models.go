package models

import "time"

type Site struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	RadiusKM  float64   `json:"radius_km"`
	CreatedAt time.Time `json:"created_at"`
}
type Observation struct {
	ID         int       `json:"id"`
	SiteID     int       `json:"site_id"`
	Pollutant  string    `json:"pollutant"`
	ObservedAt time.Time `json:"observed_at"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Source     string    `json:"source"`
}
type Anomaly struct {
	ID               int       `json:"id"`
	ObservationID    int       `json:"observation_id"`
	ObservedAt       time.Time `json:"observed_at"`
	Value            float64   `json:"value"`
	Baseline         float64   `json:"baseline"`
	DeviationPercent float64   `json:"deviation_percent"`
	ZScore           float64   `json:"z_score"`
	Severity         string    `json:"severity"`
}
