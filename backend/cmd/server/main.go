package main

import (
	"context"
	"encoding/json"
	"github.com/dandora-airwatch/backend/internal/config"
	"github.com/dandora-airwatch/backend/internal/database"
	"github.com/dandora-airwatch/backend/internal/repository"
	"github.com/dandora-airwatch/backend/internal/services/anomaly"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()
	db, err := database.Open(ctx, config.Get("DATABASE_URL", "postgres://airwatch:airwatch@localhost:5432/airwatch?sslmode=disable"))
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		panic(err)
	}
	r := repository.Repo{DB: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/api/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		jsonOut := func(v any) { w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(v) }
		if path == "sites" {
			v, e := r.Sites(req.Context())
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			jsonOut(v)
			return
		}
		if len(parts) < 2 {
			http.NotFound(w, req)
			return
		}
		id, e := strconv.Atoi(parts[1])
		if e != nil {
			http.NotFound(w, req)
			return
		}
		if parts[0] != "sites" {
			http.NotFound(w, req)
			return
		}
		if len(parts) == 2 {
			v, e := r.Site(req.Context(), id)
			if e != nil {
				http.Error(w, e.Error(), 404)
				return
			}
			jsonOut(v)
			return
		}
		site, e := r.Site(req.Context(), id)
		if e != nil {
			http.Error(w, e.Error(), 404)
			return
		}
		pollutant := req.URL.Query().Get("pollutant")
		if pollutant == "" {
			pollutant = "CH4"
		}
		to := time.Now().UTC().Add(24 * time.Hour)
		from := to.AddDate(0, 0, -90)
		if s := req.URL.Query().Get("from"); s != "" {
			from, _ = time.Parse("2006-01-02", s)
		}
		if s := req.URL.Query().Get("to"); s != "" {
			to, _ = time.Parse("2006-01-02", s)
			to = to.Add(24 * time.Hour)
		}
		switch parts[2] {
		case "observations":
			v, e := r.Observations(req.Context(), id, pollutant, from, to)
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			jsonOut(v)
		case "latest":
			v, e := r.Observations(req.Context(), id, pollutant, time.Unix(0, 0), to)
			if e != nil || len(v) == 0 {
				http.Error(w, "no observations", 404)
				return
			}
			jsonOut(v[len(v)-1])
		case "anomalies":
			v, e := r.Anomalies(req.Context(), id)
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			jsonOut(v)
		case "summary":
			v, e := r.Observations(req.Context(), id, pollutant, time.Unix(0, 0), to)
			if e != nil || len(v) == 0 {
				http.Error(w, "no observations", 404)
				return
			}
			latest := v[len(v)-1]
			prev := make([]float64, 0, len(v)-1)
			for _, o := range v[:len(v)-1] {
				prev = append(prev, o.Value)
			}
			s := anomaly.Score(latest.Value, prev)
			status := s.Severity
			if status == "normal" {
				status = "stable"
			}
			jsonOut(map[string]any{"site": site.Name, "pollutant": pollutant, "latest": map[string]any{"date": latest.ObservedAt.Format("2006-01-02"), "value": latest.Value, "unit": latest.Unit}, "baseline": s.Baseline, "difference_percent": s.DeviationPercent, "status": status})
		default:
			http.NotFound(w, req)
		}
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		mux.ServeHTTP(w, req)
	})
	http.ListenAndServe(":"+config.Get("PORT", "8080"), handler)
}
