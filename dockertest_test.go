//go:build integration

package main

import (
	"testing"

	"github.com/ory/dockertest/v4"
	"github.com/redis/go-redis/v9"
)

func TestRedisDockertest(t *testing.T) {
	// NewPoolT registers cleanup via t.Cleanup, so the container is removed
	// automatically; WithoutReuse keeps the v3 semantics of a fresh container.
	pool := dockertest.NewPoolT(t, "")
	resource := pool.RunT(t, "redis",
		// This is actually activating the CI matrix which was never activated before until
		// now that I had a better look at the code while doing the v3 -> v4 upgrade.
		dockertest.WithTag(redisVersion()),
		dockertest.WithoutReuse(),
	)
	defer func() {
		if rc != nil {
			rc.Close()
			rc = nil
		}
	}()

	if err := pool.Retry(t.Context(), 0, func() error {
		rc = redis.NewClient(&redis.Options{
			// v4's GetHostPort already returns host:port (v3's GetPort was port-only)
			Addr: resource.GetHostPort("6379/tcp"),
		})
		return rc.Ping(t.Context()).Err()
	}); err != nil {
		t.Fatalf("Could not connect to redis: %s", err)
	}
}
