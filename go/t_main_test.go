package main

import (
	"os"
	"testing"

	_ "starter/go/testhack"

	"github.com/mitranim/try"
)

func TestMain(m *testing.M) {
	// Auto-select a free port to avoid conflicts with the main server process.
	env.conf.ServerPort = 0

	// Note: this doesn't actually start the server. We "fake" our HTTP requests,
	// which allows us to pass the test DB transaction to the request handlers.
	// See `testInit` and `selfJsonFetch`.
	try.To(initServer())

	os.Exit(m.Run())
}
