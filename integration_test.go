// +build integration

package main

import (
	"os"
	"fmt"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	ver := os.Getenv("REDIS_VERSION")
	if ver == "" {
		ver = "latest"
	}
	imageName := fmt.Sprintf("redis:%s", ver)
	fmt.Println("Starting test with Redis image " + imageName)

	docker := &simpleDockerContainer{}
	err := docker.initialize(imageName, "6379")
	if err != nil {
		panic(err)
	}
	hostname, port := docker.getContainerNetworkInfo()
	rc = NewRedisClient(hostname, port)
	exitCode := m.Run()
	rc.Close()
	if err = docker.stopContainer(); err != nil {
		panic(err)
	}

	docker.client.Close()
	os.Exit(exitCode)
}

func TestRedisPing(t *testing.T) {
	t.Run("Testing Redis is connectable", func(t *testing.T) {
		_, err := rc.Ping().Result()
		if err != nil {
			t.Error("Failed to validate Redis connection")
		}
	})
}

func TestRedisCount(t *testing.T) {
	s := "Whatever"
	t.Run(fmt.Sprintf("Testing Hello %s", s), func(t *testing.T) {
		if !strings.HasPrefix(helloCache(s), fmt.Sprintf("Hello %s", s)) {
			t.Error("Didn't manage to say helloCache")
		}
	})

	t.Run("Redis has been updated", func(t *testing.T) {
		res, err := rc.Get(s).Result()
		if err != nil {
			t.Error("Failed to connect GET key from Redis", err.Error())
		}
		if res != "1" {
			t.Error(fmt.Sprintf("Redis failed to update %s", res))
		}
	})
}