package satellite

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type EarthEngineProvider struct{ Project string }

func (p EarthEngineProvider) Fetch(lat, lon, radiusKM float64, from, to time.Time) ([]Observation, error) {
	if p.Project == "" {
		return nil, errors.New("EE_PROJECT is required for SATELLITE_MODE=earthengine")
	}
	script := os.Getenv("EE_SCRIPT")
	if script == "" {
		script = filepath.Join("..", "scripts", "earthengine_ingest.py")
	}
	cmd := exec.Command("python3", script)
	cmd.Env = append(os.Environ(), "EE_PROJECT="+p.Project, "EE_LAT="+format(lat), "EE_LON="+format(lon), "EE_RADIUS_KM="+format(radiusKM), "EE_FROM="+from.UTC().Format("2006-01-02"), "EE_TO="+to.UTC().Format("2006-01-02"))
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.New("Earth Engine retrieval failed: " + err.Error())
	}
	var values []Observation
	if err := json.Unmarshal(out, &values); err != nil {
		return nil, errors.New("invalid Earth Engine output: " + err.Error())
	}
	return values, nil
}

func format(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }
