//go:build unit

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestHello(t *testing.T) {
	s := "World"
	t.Run("Testing hello", func(t *testing.T) {
		if hello(s) != fmt.Sprintf("Hello %s", s) {
			t.Error("Didn't manage to say hello")
		}
	})

	t.Run("Testing helloCache without a redis client", func(t *testing.T) {
		if !strings.HasPrefix(helloCache(s), fmt.Sprintf("Hello %s", s)) {
			t.Error("Didn't manage to say helloCache to the world")
		}
	})
}
