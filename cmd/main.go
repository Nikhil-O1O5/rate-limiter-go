package main

import (
	"net/http"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/db"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/handler"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/middleware"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/redis"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/repo"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/service"
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

	tb := limiter.NewTokenBucket(rdb)
	defaultCfg := limiter.BucketConfig{
		Capacity:   10,
		RefillRate: 2,
	}
	rl := middleware.RateLimit(tb, defaultCfg)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health(database, rdb))
	userHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	hashHandler.RegisterRoutes(mux)
	resizeHandler.RegisterRoutes(mux)

	logrus.WithField("port", cfg.AppPort).Info("server starting")
	if err := http.ListenAndServe(":"+cfg.AppPort, rl(mux)); err != nil {
		logrus.WithError(err).Fatal("server error")
	}
}
