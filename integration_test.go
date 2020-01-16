// +build integration

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var docker *simpleDockerContainer

func setUp() {
	ver := os.Getenv("REDIS_VERSION")
	if ver == "" {
		ver = "latest"
	}
	imageName := fmt.Sprintf("redis:%s", ver)
	fmt.Println("Starting test with Redis image " + imageName)

	docker = &simpleDockerContainer{}
	err := docker.initialize(imageName, "6379")
	if err != nil {
		panic(err)
	}
	hostname, port := docker.getContainerNetworkInfo()
	rc = NewRedisClient(hostname, port)
}

func tearDown() {
	if err := docker.stopContainer(); err != nil {
		panic(err)
	}

	docker.client.Close()
	rc.Close()
	rc = nil
}


func TestRedis(t *testing.T) {
	setUp()

	t.Run("Testing Redis is connectable", func(t *testing.T) {
		_, err := rc.Ping().Result()
		if err != nil {
			t.Error("Failed to validate Redis connection")
		}
	})

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

	tearDown()
}
