package main

import (
	"net/http"
)

const (
	GET     = http.MethodGet
	HEAD    = http.MethodHead
	POST    = http.MethodPost
	PUT     = http.MethodPut
	PATCH   = http.MethodPatch
	DELETE  = http.MethodDelete
	OPTIONS = http.MethodOptions
)

func isHttpStatusOk(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}

func preventCaching(header http.Header) {
	header.Add("cache-control", "must-revalidate")
	header.Add("cache-control", "no-cache")
	header.Add("cache-control", "no-store")
	header.Add("cache-control", "proxy-revalidate")
	header.Add("cache-control", "max-age=0")
	header.Add("expires", "0")
}

/*
Reference: https://www.w3.org/TR/cors/.
*/
func allowCors(header http.Header) {
	header.Add("access-control-allow-credentials", "true")
	header.Add("access-control-allow-headers", "content-type")
	header.Add("access-control-allow-methods", "OPTIONS, GET, HEAD, POST, PUT, PATCH, DELETE")
	header.Add("access-control-allow-origin", "*")
}

var jsonResHeader = httpHead("content-type", "application/json")

func patchHttpHeader(left http.Header, right http.Header) http.Header {
	out := http.Header{}
	patchHttpHeaderMut(out, left)
	patchHttpHeaderMut(out, right)
	return out
}

// Panics if `target` is nil.
func patchHttpHeaderMut(target http.Header, source http.Header) {
	for key, vals := range source {
		target.Del(key)
		for _, val := range vals {
			target.Add(key, val)
		}
	}
}

/*
Allows to construct an `http.Header` in an expression while avoiding the gotcha
with literal constructors where using non-canonical keys makes them inaccessible
through `.Get`.

TODO: replace with `normalizeHttpHeader`.
*/
func httpHead(key string, val string) http.Header {
	out := http.Header{}
	out.Add(key, val)
	return out
}
