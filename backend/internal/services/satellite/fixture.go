package satellite

import (
	"math/rand"
	"time"
)

type FixtureProvider struct{}

func (FixtureProvider) Fetch(_, _, _ float64, from, to time.Time) ([]Observation, error) {
	r := rand.New(rand.NewSource(42))
	out := []Observation{}
	base := 1901.4
	for d := from.UTC(); d.Before(to.UTC()); d = d.Add(24 * time.Hour) {
		if r.Float64() < .13 {
			continue
		}
		v := base + float64(r.Intn(58)-29)
		if d.Day() == 12 && d.Month() == 9 {
			v = 1985.2
		}
		if d.Day() == 24 && d.Month() == 8 {
			v = 1943.0
		}
		out = append(out, Observation{ObservedAt: d, Value: v, Unit: "ppb", Source: "DEVELOPMENT FIXTURE (replace with Earth Engine)"})
	}
	return out, nil
}
