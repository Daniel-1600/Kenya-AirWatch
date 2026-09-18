package anomaly

import "math"

const MinBaselineObservations = 10

type Result struct {
	Baseline, DeviationPercent, ZScore float64
	Severity                           string
}

func Classify(z float64) string {
	if z >= 3 {
		return "severe"
	}
	if z >= 2 {
		return "high"
	}
	if z >= 1 {
		return "elevated"
	}
	return "normal"
}

func Score(current float64, previous []float64) Result {
	if len(previous) == 0 {
		return Result{Baseline: current, Severity: "normal"}
	}
	mean := 0.0
	for _, v := range previous {
		mean += v
	}
	mean /= float64(len(previous))
	deviationPercent := 0.0
	if mean != 0 {
		deviationPercent = (current - mean) / mean * 100
	}
	if len(previous) < MinBaselineObservations {
		return Result{Baseline: mean, DeviationPercent: deviationPercent, Severity: "normal"}
	}
	variance := 0.0
	for _, v := range previous {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(previous))
	sd := math.Sqrt(variance)
	z := 0.0
	if sd > 0 {
		z = (current - mean) / sd
	}
	return Result{Baseline: mean, DeviationPercent: deviationPercent, ZScore: z, Severity: Classify(z)}
}
