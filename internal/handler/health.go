package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type healthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

func Health(db *sqlx.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := map[string]string{
			"postgres": "ok",
			"redis":    "ok",
		}
		status := http.StatusOK

		if err := db.PingContext(context.Background()); err != nil {
			services["postgres"] = "unavailable"
			status = http.StatusServiceUnavailable
		}

		if err := rdb.Ping(context.Background()).Err(); err != nil {
			services["redis"] = "unavailable"
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(healthResponse{
			Status:   http.StatusText(status),
			Services: services,
		})
	}
}
