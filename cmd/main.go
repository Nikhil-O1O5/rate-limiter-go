package main

import (
	"net/http"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/db"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/handler"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/redis"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to postgres")
	}

	rdb, err := redis.New(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to redis")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health(database, rdb))

	logrus.WithField("port", cfg.AppPort).Info("server starting")
	if err := http.ListenAndServe(":"+cfg.AppPort, mux); err != nil {
		logrus.WithError(err).Fatal("server error")
	}
}
