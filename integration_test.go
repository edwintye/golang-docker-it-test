//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

var docker *simpleDockerContainer

func setUp() {
	fmt.Printf("Begin setUp() on the test environment")
	ctx = context.Background()
	ver := os.Getenv("REDIS_VERSION")
	if ver == "" {
		ver = "latest"
	}
	imageName := fmt.Sprintf("redis:%s", ver)
	fmt.Println("Starting test with Redis image " + imageName)

	docker = &simpleDockerContainer{}
	err := docker.initialize(imageName, "6379")
	if err != nil {
		fmt.Println("We are in a real panic")
		panic(err)
	}
	hostname, port := docker.getContainerNetworkInfo()
	fmt.Printf("The hostname = %s and port number = %s\n", hostname, port)
	fmt.Printf("Initialized the container and pause to ensure container is ready and setup client\n")
	time.Sleep(time.Second)
	rc = NewRedisClient(hostname, port)
	fmt.Printf("Finish setUp() and ready to start testing\n")
}

func tearDown() {
	fmt.Printf("Begin tearDown() on the test environment")
	if err := docker.stopContainer(); err != nil {
		panic(err)
	}

	docker.client.Close()
	rc.Close()
	rc = nil
	fmt.Printf("Complete tearDown()")
}

func TestRedis(t *testing.T) {
	setUp()

	t.Run("Testing Redis is connectable", func(t *testing.T) {
		_, err := rc.Ping(ctx).Result()
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
		res, err := rc.Get(ctx, s).Result()
		if err != nil {
			t.Error("Failed to connect GET key from Redis", err.Error())
		}
		if res != "1" {
			t.Error(fmt.Sprintf("Redis failed to update %s", res))
		}
	})

	tearDown()
}
