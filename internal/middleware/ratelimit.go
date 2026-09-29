package middleware

import (
	"net/http"
	"strconv"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/handler"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	"github.com/sirupsen/logrus"
)

func RateLimit(tb *limiter.TokenBucket, rlCfg *config.RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				handler.WriteError(w, http.StatusUnauthorized, "X-User-ID header is required")
				return
			}

			endpoint := r.URL.Path
			cfg := rlCfg.For(endpoint)

			result, err := tb.Allow(r.Context(), userID, endpoint, cfg)
			if err != nil {
				logrus.WithError(err).Error("rate limiter error")
				handler.WriteError(w, http.StatusInternalServerError, "rate limiter unavailable")
				return
			}

			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(cfg.Capacity, 'f', 0, 64))

			if !result.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(result.RetryAfterMS/1000, 10))
				handler.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
