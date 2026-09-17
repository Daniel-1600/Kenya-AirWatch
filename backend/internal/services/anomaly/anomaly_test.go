package anomaly

import "testing"

func TestScoreHigh(t *testing.T) {
	r := Score(120, []float64{100, 101, 99, 100})
	if r.Severity != "severe" {
		t.Fatalf("got %s", r.Severity)
	}
}
