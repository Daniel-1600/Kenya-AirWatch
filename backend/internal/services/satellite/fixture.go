package satellite

import (
	"math/rand"
	"time"
)

type FixtureProvider struct{}

func (FixtureProvider) Fetch(lat, lon, _ float64, from, to time.Time) ([]Observation, error) {
	// Seed and baseline derive from the requested coordinates so each demo
	// site gets a distinct, deterministic signal instead of identical noise.
	seed := int64(lat*1000) ^ int64(lon*1000)
	r := rand.New(rand.NewSource(seed))
	base := 1901.4 + (lat+lon)*2
	out := []Observation{}
	for d := from.UTC(); d.Before(to.UTC()); d = d.Add(24 * time.Hour) {
		if r.Float64() < .13 {
			continue
		}
		v := base + float64(r.Intn(58)-29)
		if d.Day() == 12 && d.Month() == 9 {
			v = base + 70
		}
		if d.Day() == 24 && d.Month() == 8 {
			v = base + 30
		}
		out = append(out, Observation{ObservedAt: d, Value: v, Unit: "ppb", Source: "DEVELOPMENT FIXTURE (replace with Earth Engine)"})
	}
	return out, nil
}
