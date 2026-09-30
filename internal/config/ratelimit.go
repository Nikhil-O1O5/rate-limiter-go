package config

import (
	"fmt"
	"os"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/limiter"
	"gopkg.in/yaml.v3"
)

type endpointConfig struct {
	Capacity   float64 `yaml:"capacity"`
	RefillRate float64 `yaml:"refill_rate"`
}

type rateLimitConfig struct {
	Endpoints map[string]endpointConfig `yaml:"endpoints"`
}

type RateLimitConfig struct {
	rules map[string]limiter.BucketConfig
}

func LoadRateLimitConfig(path string) (*RateLimitConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var raw rateLimitConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	rules := make(map[string]limiter.BucketConfig, len(raw.Endpoints))
	for endpoint, cfg := range raw.Endpoints {
		rules[endpoint] = limiter.BucketConfig{
			Capacity:   cfg.Capacity,
			RefillRate: cfg.RefillRate,
		}
	}

	return &RateLimitConfig{rules: rules}, nil
}

func (c *RateLimitConfig) For(endpoint string) limiter.BucketConfig {
	if cfg, ok := c.rules[endpoint]; ok {
		return cfg
	}
	return c.rules["default"]
}
