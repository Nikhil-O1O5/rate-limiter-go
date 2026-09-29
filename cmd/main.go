package main

import (
	"net/http"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/db"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/handler"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	_ "github.com/Nikhil-O1O5/rate-limiter-go/internal/metrics"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/middleware"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/redis"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/repo"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	userRepo := repo.NewUserRepo(database)
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userSvc)

	searchRepo := repo.NewSearchRepo(database)
	searchSvc := service.NewSearchService(searchRepo)
	searchHandler := handler.NewSearchHandler(searchSvc)

	hashSvc := service.NewHashService()
	hashHandler := handler.NewHashHandler(hashSvc)

	resizeSvc := service.NewResizeService()
	resizeHandler := handler.NewResizeHandler(resizeSvc)

	rlCfg, err := config.LoadRateLimitConfig("config.yaml")
	if err != nil {
		logrus.WithError(err).Fatal("failed to load rate limit config")
	}

	tb := limiter.NewTokenBucket(rdb)
	rl := middleware.RateLimit(tb, rlCfg)

	// apiMux is rate-limited; internalMux bypasses the middleware.
	apiMux := http.NewServeMux()
	userHandler.RegisterRoutes(apiMux)
	searchHandler.RegisterRoutes(apiMux)
	hashHandler.RegisterRoutes(apiMux)
	resizeHandler.RegisterRoutes(apiMux)

	root := http.NewServeMux()
	root.Handle("GET /metrics", promhttp.Handler())
	root.HandleFunc("GET /health", handler.Health(database, rdb))
	root.Handle("/", rl(apiMux))

	logrus.WithField("port", cfg.AppPort).Info("server starting")
	if err := http.ListenAndServe(":"+cfg.AppPort, root); err != nil {
		logrus.WithError(err).Fatal("server error")
	}
}
