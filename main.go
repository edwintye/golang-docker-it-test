package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var rc *redis.Client
var ctx context.Context

func hello(name string) string {
	return fmt.Sprintf("Hello %s", name)
}

func helloCache(name string) string {
	if rc == nil {
		return hello(name)
	}
	n, err := rc.Get(ctx, name).Result()
	if n != "" && err != nil {
		panic(err)
	} else if n == "" {
		n = "0"
	}

	s := fmt.Sprintf("%s, you have been here %s times before", hello(name), n)
	_, err = rc.IncrBy(ctx, name, 1).Result()
	if err != nil {
		panic(err)
	}
	return s
}

func NewRedisClient(hostname string, port string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     hostname + ":" + port,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	return client
}
func main() {
	ctx = context.Background()
	rc = NewRedisClient("localhost", "6379")
	var s string
	for {
		fmt.Print("> ")
		_, _ = fmt.Scanln(&s)
		fmt.Println(helloCache(s))
	}
}
