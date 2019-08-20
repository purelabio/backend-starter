package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/mitranim/try"
	"github.com/pkg/errors"
)

func startServer() error {
	err := initServer()
	if err != nil {
		return err
	}
	return runServer()
}

/*
This should prepare the server for an immediate start, but not actually run the
server or any side effectful operations.
*/
func initServer() (err error) {
	defer try.Rec(&err)

	try.To(initDb())

	/**
	Manually creating a listener allows us to find the auto-assigned port if
	`SERVER_PORT == 0`.
	*/
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", env.conf.ServerPort))
	try.To(err)

	env.serverListener = listener
	env.server = &http.Server{Handler: http.HandlerFunc(handleRequest)}
	env.fileServer = http.FileServer(http.Dir(env.conf.PublicDir))

	return nil
}

/*
Since the TCP listener is already initialized, this should be instant, which
means tests don't need to poll the server for readiness.
*/
func runServer() error {
	listener := env.serverListener
	port := listener.Addr().(*net.TCPAddr).Port
	env.log.Info("listening on http://localhost:", port)
	err := env.server.Serve(listener)
	return errors.WithStack(err)
}
