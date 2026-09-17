package satellite

import "time"

type Observation struct {
	ObservedAt   time.Time
	Value        float64
	Unit, Source string
}
type Provider interface {
	Fetch(siteLat, siteLon, radiusKM float64, from, to time.Time) ([]Observation, error)
}
