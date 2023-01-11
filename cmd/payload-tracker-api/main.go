package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/logging"
)

func lubdub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("lubdub"))
}

func main() {

	logging.InitLogger()

	cfg := config.Get()

	db.DbConnect(cfg)

	healthHandler := endpoints.HealthCheckHandler(
		db.DB,
		*cfg,
	)

	payloadArchiveLinkHandler := endpoints.CreatePayloadArchiveLinkHandler(
		*cfg,
	)

	r := chi.NewRouter()
	mr := chi.NewRouter()
	sub := chi.NewRouter()

	r.Use(httprate.LimitByIP(cfg.RequestConfig.MaxRequestsPerMinute, 1*time.Minute))

	// Mount the root of the api router on /api/v1 unless ENVIRONMENT is DEV
	if cfg.Environment == "DEV" {
		r.Mount("/app/payload-tracker/api/v1/", sub)
	} else {
		r.Mount("/api/v1/", sub)
	}

	if cfg.RequestConfig.RequestorImpl == "mock" {
		sub.Get("/archive/{id}", endpoints.ArchiveHandler)
	}

	r.Get("/", lubdub)
	r.Get("/health", healthHandler)

	// Mount the metrics handler on /metrics
	mr.Get("/", lubdub)
	mr.Handle("/metrics", promhttp.Handler())

	sub.With(endpoints.ResponseMetricsMiddleware).Get("/", lubdub)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads", endpoints.Payloads)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads/{request_id}", endpoints.RequestIdPayloads)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads/{request_id}/archiveLink", payloadArchiveLinkHandler)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/payloads/{request_id}/kibanaLink", endpoints.PayloadKibanaLink)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/roles/archiveLink", endpoints.RolesArchiveLink)
	sub.With(endpoints.ResponseMetricsMiddleware).Get("/statuses", endpoints.Statuses)

	srv := http.Server{
		Addr:    ":" + cfg.PublicPort,
		Handler: r,
	}

	msrv := http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: mr,
	}

	go func() {

		if err := msrv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
