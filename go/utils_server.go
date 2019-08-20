package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/mitranim/goh"
	"github.com/mitranim/try"
)

// For inclusion in URL patterns.
const (
	intIdPattern      = `(\d+)`
	urlSegmentPattern = `([^?/#\s]+)`
)

// Should be used after `reqWithReqCtx`.
func ctxReq(ctx Ctx) *Req {
	if ctx != nil {
		req, _ := ctx.Value(CTX_REQ_KEY).(*Req)
		return req
	}
	return nil
}

func isCtxHttp(ctx Ctx) bool {
	return ctxReq(ctx) != nil
}

// Makes sure the request context contains the request. It can be retrieved
// with `ctxReq`.
func reqWithReqCtx(req *Req) *Req {
	if req == nil {
		return nil
	}
	ctx := context.WithValue(req.Context(), CTX_REQ_KEY, req)
	return req.WithContext(ctx)
}

func reqDownloadDecode(req *Req, out interface{}) error {
	// Automatically uses `ErrPubBadRequest` when appropriate.
	dec, err := DownloadReqdec(req)
	if err != nil {
		return err
	}
	// Automatically uses `ErrPubBadRequest` when appropriate.
	return dec.DecodeStruct(out)
}

func reqDownloadDecodeValidate(req *Req, out Validator) error {
	err := reqDownloadDecode(req, out)
	if err != nil {
		return err
	}
	return ErrPubBadRequest(out.Validate())
}

func reqRemovePrefix(req *Req, prefix string) {
	req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
}

/*
Short for "respond with". Runs a response handler and sends back its response or
error.
*/
func resWith(rew Rew, req *Req, fun func(Ctx) (Res, error)) {
	res, err := unpanicResWith(req.Context(), fun)
	writeResOrErr(rew, req, res, err)
}

func unpanicResWith(ctx Ctx, fun func(Ctx) (Res, error)) (_ Res, err error) {
	defer try.Rec(&err)
	return fun(ctx)
}

/*
Short for "respond with DB transaction". Runs a response handler in the context
of a DB transaction, commits the transaction, THEN sends back the handler's
response or an error. If the handler returns an error, we abort the DB
transaction. If the DB transaction fails to commit, we ignore the handler's
response and send the commit error.
*/
func resWithDbTx(rew Rew, req *Req, fun func(Ctx, DbTx) (Res, error)) {
	var res Res
	var err error

	/**
	Note: we must use the error returned by `withReqDbTx` rather than just the
	error returned by `fun` because `withReqDbTx` may fail to commit the DB
	transaction.
	*/
	err = withReqDbTx(req, func(ctx Ctx, conn DbTx) error {
		res, err = fun(ctx, conn)
		return err
	})

	writeResOrErr(rew, req, res, err)
}

func writeRes(rew Rew, req *Req, res Res) {
	if res != nil {
		res.ServeHTTP(rew, req)
	}
}

/*
TODO: consider choosing error encoding based on the requested content type. For
plain text, we continue returning the error's text. For JSON, we might encode
the error as JSON for easy programmatic inspection; an `Error` may be encoded
as-is; for other errors we should define something similarly-structured but
opaque. For HTML, we might render a special HTML error page.
*/
func writeErr(rew Rew, req *Req, wrote bool, err error) {
	if err == nil {
		return
	}

	// Log the error, unless it's meaningless or being reported to the client.
	if errPub(err) != err {
		maybeLogError(req.Context(), err)
	}

	if wrote {
		return
	}

	pub, ok := errPub(err).(Error)
	if ok {
		writeRes(rew, req, goh.StringWith(pub.HttpStatusCode(), pub.Error()))
		return
	}

	/**
	TODO: consider auto-detecting the HTTP status code here, despite hiding the
	error message. For example, we could use `isErrNotFound` to detect 404.
	*/
	writeRes(rew, req, goh.StringWith(http.StatusInternalServerError, `Unexpected Error`))
}

func writeResOrErr(rew Rew, req *Req, res Res, err error) {
	if err != nil {
		writeErr(rew, req, false, err)
	} else {
		writeRes(rew, req, res)
	}
}
