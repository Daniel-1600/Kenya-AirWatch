package anomaly

import "testing"

func TestScoreHigh(t *testing.T) {
	r := Score(120, []float64{100, 101, 99, 100, 100, 101, 99, 100, 100, 101})
	if r.Severity != "severe" {
		t.Fatalf("got %s", r.Severity)
	}
}

func TestScoreDoesNotAlertOnLowObservation(t *testing.T) {
	r := Score(80, []float64{100, 101, 99, 100, 100, 101, 99, 100, 100, 101})
	if r.Severity != "normal" {
		t.Fatalf("got %s", r.Severity)
	}
}

func TestScoreBuildsBaselineBeforeAlerting(t *testing.T) {
	r := Score(120, []float64{100, 101, 99, 100})
	if r.Severity != "normal" || r.ZScore != 0 {
		t.Fatalf("got severity=%s z=%f", r.Severity, r.ZScore)
	}
}
