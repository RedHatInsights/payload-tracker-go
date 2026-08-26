package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redhatinsights/payload-tracker-go/internal/config"
	"github.com/redhatinsights/payload-tracker-go/internal/db"
	"github.com/redhatinsights/payload-tracker-go/internal/endpoints"
	"github.com/redhatinsights/payload-tracker-go/internal/kafka"
	"github.com/redhatinsights/payload-tracker-go/internal/logging"
	"github.com/redhatinsights/payload-tracker-go/internal/securitylog"
)

func lubdub(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("lubdub"))
}

func main() {
	logging.InitLogger()

	cfg := config.Get()
	ctx := context.Background()

	logging.Log.Info("Setting up DB")
	db.DbConnect(cfg)

	healthHandler := endpoints.HealthCheckHandler(
		db.DB,
		*cfg,
	)

	logging.Log.Info("Starting a new kafka consumer...")

	// Webserver is created only for metrics collection
	r := chi.NewRouter()

	// Mount the metrics handler on /metrics
	r.Get("/", lubdub)
	r.Get("/live", healthHandler)
	r.Get("/ready", healthHandler)
	r.Handle("/metrics", promhttp.Handler())

	msrv := http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: r,
	}

	consumer, err := kafka.NewConsumer(ctx, cfg, cfg.KafkaConfig.KafkaTopic)

	if err != nil {
		securitylog.LogLifecycle("STARTUP", "failure", "pt-consumer")
		logging.Log.Fatal("ERROR! ", err)
	}

	securitylog.LogLifecycle("STARTUP", "success", "pt-consumer")

	go func() {

		if err := msrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Log.Error("Metrics server error: ", err)
		}
	}()

	kafka.NewConsumerEventLoop(ctx, cfg, consumer, db.DB)
	securitylog.LogLifecycle("SHUTDOWN", "success", "pt-consumer")
}
