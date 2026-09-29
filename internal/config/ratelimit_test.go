package config_test

import (
	"os"
	"testing"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfig = `
endpoints:
  /hash:
    capacity: 3
    refill_rate: 0.5
  /feed:
    capacity: 20
    refill_rate: 5
  default:
    capacity: 10
    refill_rate: 2
`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "ratelimit-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	f.Close()
	return f.Name()
}

func TestLoadRateLimitConfig_ValidFile(t *testing.T) {
	path := writeTempConfig(t, testConfig)
	cfg, err := config.LoadRateLimitConfig(path)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestLoadRateLimitConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadRateLimitConfig("nonexistent.yaml")
	assert.Error(t, err)
}

func TestFor_KnownEndpoint(t *testing.T) {
	path := writeTempConfig(t, testConfig)
	cfg, err := config.LoadRateLimitConfig(path)
	require.NoError(t, err)

	hashCfg := cfg.For("/hash")
	assert.Equal(t, float64(3), hashCfg.Capacity)
	assert.Equal(t, 0.5, hashCfg.RefillRate)

	feedCfg := cfg.For("/feed")
	assert.Equal(t, float64(20), feedCfg.Capacity)
	assert.Equal(t, float64(5), feedCfg.RefillRate)
}

func TestFor_UnknownEndpointFallsBackToDefault(t *testing.T) {
	path := writeTempConfig(t, testConfig)
	cfg, err := config.LoadRateLimitConfig(path)
	require.NoError(t, err)

	unknown := cfg.For("/some/unknown/path")
	assert.Equal(t, float64(10), unknown.Capacity)
	assert.Equal(t, float64(2), unknown.RefillRate)
}
