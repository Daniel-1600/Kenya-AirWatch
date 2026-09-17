package anomaly

import "math"

type Result struct {
	Baseline, DeviationPercent, ZScore float64
	Severity                           string
}

func Classify(z float64) string {
	a := math.Abs(z)
	if a >= 3 {
		return "severe"
	}
	if a >= 2 {
		return "high"
	}
	if a >= 1 {
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
	return Result{Baseline: mean, DeviationPercent: (current - mean) / mean * 100, ZScore: z, Severity: Classify(z)}
}
