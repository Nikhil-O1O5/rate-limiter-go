# rate-limiter-go

A distributed rate limiter service built in Go. Uses the token bucket algorithm backed by Redis, with per-user and per-endpoint isolation, Prometheus metrics, and a Grafana dashboard.

## How it works

Every incoming request passes through the rate limit middleware which:

1. Reads the `X-User-ID` header — returns 401 if missing
2. Looks up the per-endpoint config from `config.yaml`
3. Calls `TokenBucket.Allow(userID, endpoint)` which runs an atomic Lua script in Redis
4. Allows the request or returns 429 with a `Retry-After` header

Each user gets an independent bucket per endpoint, keyed as `rate_limit:{userID}:{endpoint}`. A user exhausting `/hash` has no effect on their `/feed` bucket.

## Token bucket algorithm

The Lua script runs atomically inside Redis — no locks, no race conditions:

```
1. HMGET key tokens last_refill
2. elapsed = (now - last_refill) / 1000
3. tokens  = min(capacity, tokens + elapsed × refill_rate)
4. if tokens >= 1 → consume one, allow
   else           → deny, return ms until next token
5. HSET key tokens <new> last_refill <now>
6. EXPIRE key <ttl>
```

Because Redis executes Lua scripts as a single atomic operation, concurrent requests from the same user can never double-spend tokens, even under heavy load.

## Project structure

```
cmd/
  main.go                     entry point, wires everything together

internal/
  config/
    config.go                 app config loaded from env vars
    ratelimit.go              YAML config loader + For(endpoint) lookup
    ratelimit_test.go         config tests

  limiter/
    token_bucket.go           TokenBucket struct, Allow(), clock injection
    token_bucket_test.go      unit, concurrency, and benchmark tests
    lua/
      token_bucket.lua        atomic Redis script

  middleware/
    ratelimit.go              HTTP middleware, sets response headers
    ratelimit_test.go         middleware HTTP tests

  metrics/
    metrics.go                Prometheus counters and histograms

  handler/                    HTTP handlers (users, search, hash, resize, health)
  service/                    business logic
  repo/                       Postgres queries
  model/                      domain types
  db/                         Postgres connection
  redis/                      Redis connection

grafana/
  provisioning/
    datasources/prometheus.yml    auto-registers Prometheus on Grafana start
    dashboards/dashboards.yml     tells Grafana where to load dashboard JSON from
  dashboards/
    rate-limiter.json             pre-built dashboard (request rate, denial %, tokens)

prometheus.yml                Prometheus scrape config
config.yaml                   per-endpoint rate limit rules
docker-compose.yml            Postgres, Redis, Prometheus, Grafana
```

## Endpoints

| Method | Path | Description | Rate limited |
|--------|------|-------------|--------------|
| GET | /health | Postgres + Redis health check | No |
| GET | /metrics | Prometheus metrics | No |
| POST | /users | Create a user | Yes |
| GET | /users/{id} | Get user by ID | Yes |
| POST | /search | Full-text search over users | Yes |
| GET | /feed | List all users | Yes |
| POST | /hash | bcrypt hash a password (CPU-heavy) | Yes |
| POST | /resize | Resize an uploaded image | Yes |

`/health` and `/metrics` bypass the rate limit middleware entirely via a two-mux pattern — Prometheus scrapes `/metrics` every 5s without needing a user ID or burning tokens.

## Rate limit config

Edit `config.yaml` to change limits — no recompile needed:

```yaml
endpoints:
  /hash:    { capacity: 3,  refill_rate: 0.5 }   # 3 bursts, refills 1 token every 2s
  /resize:  { capacity: 5,  refill_rate: 1   }
  /search:  { capacity: 10, refill_rate: 3   }
  /feed:    { capacity: 20, refill_rate: 5   }
  /users:   { capacity: 10, refill_rate: 2   }
  default:  { capacity: 10, refill_rate: 2   }
```

Unknown endpoints fall back to `default`.

## Response headers

Every rate-limited response includes:

```
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 2
```

Denied requests (HTTP 429) also include:

```
Retry-After: 2
```

## Running locally

**Prerequisites:** Docker, Go 1.22+

```bash
# copy and fill in your credentials
cp .env.example .env

# start Postgres, Redis, Prometheus, Grafana
make up

# build and run the app
make run
```

| Service | URL | Credentials |
|---------|-----|-------------|
| App | http://localhost:8080 | — |
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | — |

The Rate Limiter dashboard in Grafana loads automatically — no manual setup needed.

## Testing

```bash
make test        # all tests
make test-race   # all tests with Go race detector
```

### Token bucket tests (`internal/limiter/token_bucket_test.go`)

Uses [miniredis](https://github.com/alicebob/miniredis) — an in-process Redis — so tests run fast with no external dependencies.

| Test | What it verifies |
|------|-----------------|
| `TestAllow_FirstRequestAllowed` | First request is allowed, remaining = capacity - 1 |
| `TestAllow_CapacityExhausted` | Bucket drains correctly, 6th request on a capacity-5 bucket is denied |
| `TestAllow_RetryAfterIsSet` | RetryAfterMS is set and ≥ 1000ms for a 0.5 token/s bucket |
| `TestAllow_TokensRefillOverTime` | Advancing a fake clock by 2s refills 2 tokens and allows a previously-denied request |
| `TestAllow_UserIsolation` | Exhausting user1's bucket has no effect on user2 |
| `TestAllow_EndpointIsolation` | Exhausting `/hash` has no effect on `/feed` for the same user |
| `TestAllow_RemainingDecrementsCorrectly` | Remaining decrements 2→1→0 across sequential requests |
| `TestAllow_ConcurrentRequests` | 50 goroutines released simultaneously via a channel barrier — allowed count never exceeds capacity, every request is accounted for |

The time-refill test uses clock injection: `NewTokenBucketWithClock(rdb, func() int64 { return fakeNow })`. Advancing `fakeNow` controls the bucket's sense of time without sleeping. Production code always uses `time.Now()`.

The concurrency test is also run with `-race` to confirm the Lua script's atomicity prevents data races under real concurrent load.

### Benchmarks (`internal/limiter/token_bucket_test.go`)

```bash
go test -bench=. -benchtime=3s -benchmem ./internal/limiter/...
```

| Benchmark | What it measures |
|-----------|-----------------|
| `BenchmarkAllow_Sequential` | Single-goroutine throughput — baseline Redis + Lua round-trip cost |
| `BenchmarkAllow_Parallel` | Multi-goroutine throughput via `b.RunParallel` |

Both use an effectively-unlimited bucket (capacity 1e9) so the benchmark measures infrastructure overhead, not contention.

> Note: Against miniredis both benchmarks show similar numbers (~82µs/op) because miniredis serialises on a mutex rather than a real network socket. Against a real Redis the parallel benchmark would show significantly higher total throughput as goroutines overlap their network wait time.

### Middleware tests (`internal/middleware/ratelimit_test.go`)

Tests the full HTTP layer using `httptest` and miniredis — no real network or database needed.

| Test | What it verifies |
|------|-----------------|
| `TestRateLimit_MissingUserID` | Request without `X-User-ID` returns 401 |
| `TestRateLimit_AllowedRequest` | Allowed request passes through with correct `X-RateLimit-Remaining` and `X-RateLimit-Limit` headers |
| `TestRateLimit_RemainingDecrementsAcrossRequests` | Remaining decrements correctly across sequential requests |
| `TestRateLimit_DeniedRequest` | Exhausted bucket returns 429 with `Retry-After` header set |
| `TestRateLimit_RetryAfterIsPositive` | `Retry-After` is non-zero on denial |
| `TestRateLimit_UserIsolation` | Exhausting one user's bucket does not affect another user |
| `TestRateLimit_UnknownEndpointUsesDefault` | Unrecognised path falls back to the `default` config |
| `TestRateLimit_RedisUnavailable` | Returns 500 when Redis is down |

### Config tests (`internal/config/ratelimit_test.go`)

| Test | What it verifies |
|------|-----------------|
| `TestLoadRateLimitConfig_ValidFile` | YAML file loads without error |
| `TestLoadRateLimitConfig_FileNotFound` | Returns an error for a missing file |
| `TestFor_KnownEndpoint` | Correct capacity and refill_rate returned for `/hash` and `/feed` |
| `TestFor_UnknownEndpointFallsBackToDefault` | Unknown path returns the `default` config |

## Triggering a 429

`/hash` has the tightest limit (capacity 3, refills 1 token every 2s). Run this to exhaust it:

```bash
for i in 1 2 3 4 5; do curl -s -o /dev/null -w "req $i: %{http_code}\n" -X POST http://localhost:8080/hash -H "X-User-ID: u1" -H "Content-Type: application/json" -d '{"password":"secret"}'; done
```

Expected:
```
req 1: 200
req 2: 200
req 3: 200
req 4: 429
req 5: 429
```

## Observability

Two Prometheus metrics are recorded by the middleware on every request:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `rate_limit_decisions_total` | Counter | `endpoint`, `result` | Total allowed / denied decisions |
| `rate_limit_remaining_tokens` | Histogram | `endpoint` | Token count at decision time |

The Grafana dashboard has three panels:

- **Request Rate** — allowed vs denied req/s per endpoint, stacked and colour-coded
- **Denial Rate %** — fraction of requests being denied per endpoint
- **Remaining Tokens** — p50 and p10 quantiles showing how full buckets are over time
