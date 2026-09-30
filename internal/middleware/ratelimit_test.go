package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	"github.com/Nikhil-O1O5/rate-limiter-go/internal/middleware"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// okHandler is a trivial next handler that writes 200 so we can confirm the
// middleware passed the request through.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func setupMiddleware(t *testing.T) (http.Handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	tb := limiter.NewTokenBucket(rdb)

	// write a minimal config to a temp file
	f, err := os.CreateTemp("", "rl-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.WriteString(`
endpoints:
  /feed:
    capacity: 3
    refill_rate: 1
  default:
    capacity: 10
    refill_rate: 2
`)
	f.Close()

	rlCfg, err := config.LoadRateLimitConfig(f.Name())
	require.NoError(t, err)

	handler := middleware.RateLimit(tb, rlCfg)(okHandler)
	return handler, mr
}

func get(path, userID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if userID != "" {
		r.Header.Set("X-User-ID", userID)
	}
	return r
}

func TestRateLimit_MissingUserID(t *testing.T) {
	h, _ := setupMiddleware(t)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", ""))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRateLimit_AllowedRequest(t *testing.T) {
	h, _ := setupMiddleware(t)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", "user1"))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_RemainingDecrementsAcrossRequests(t *testing.T) {
	h, _ := setupMiddleware(t)

	for _, want := range []string{"2", "1", "0"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, get("/feed", "user1"))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, want, w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestRateLimit_DeniedRequest(t *testing.T) {
	h, _ := setupMiddleware(t)

	// drain the bucket (capacity 3)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, get("/feed", "user1"))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", "user1"))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimit_RetryAfterIsPositive(t *testing.T) {
	h, _ := setupMiddleware(t)

	for i := 0; i < 3; i++ {
		httptest.NewRecorder() // drain without checking
		w := httptest.NewRecorder()
		h.ServeHTTP(w, get("/feed", "user1"))
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", "user1"))

	retryAfter := w.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter)
	assert.NotEqual(t, "0", retryAfter)
}

func TestRateLimit_UserIsolation(t *testing.T) {
	h, _ := setupMiddleware(t)

	// exhaust user1
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, get("/feed", "user1"))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", "user1"))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// user2 should be unaffected
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, get("/feed", "user2"))
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimit_UnknownEndpointUsesDefault(t *testing.T) {
	h, _ := setupMiddleware(t)

	// default capacity is 10 — first request should be allowed with remaining 9
	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/some/unknown/path", "user1"))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_RedisUnavailable(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	tb := limiter.NewTokenBucket(rdb)

	f, err := os.CreateTemp("", "rl-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.WriteString("endpoints:\n  default:\n    capacity: 10\n    refill_rate: 2\n")
	f.Close()

	rlCfg, err := config.LoadRateLimitConfig(f.Name())
	require.NoError(t, err)

	mr.Close() // kill Redis before the request

	h := middleware.RateLimit(tb, rlCfg)(okHandler)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, get("/feed", "user1"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
