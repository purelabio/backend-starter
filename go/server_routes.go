package main

import "github.com/mitranim/rout"

func handleRequest(rew Rew, req *Req) {
	req = reqWithReqCtx(req)
	preventCaching(rew.Header())
	allowCors(rew.Header())

	if req.Method == OPTIONS {
		return
	}

	err := errNorm(rout.Route(rew, req, routes))
	writeErr(rew, req, false, err)
}

func routes(r rout.R) {
	r.Sub(`^/api/v1(?:/|$)`, routesApi)
	r.Get(``, env.fileServer.ServeHTTP)
}

func routesApi(r rout.R) {
	r.Get(`^/api/v1$`, apiHealthCheck)
}
