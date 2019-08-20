package main

/*
Currently this file is for test UTILS, not util TESTS.
*/

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitranim/try"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// Note: this can't be aliased to `*testing.T` and `*testing.B` because the test
// runner would erroneously fail the checks of test function signatures. We have
// to write `*` everywhere.
type T = testing.T
type B = testing.B
type TB = testing.TB

type TestSess struct{}

func (self TestSess) Header() http.Header { return nil }

/*
Must be called at the start of each test. Initializes the context and DB
transaction for this test; the context is canceled at the end, rolling back the
transaction. Sets the context in `env` to make it available to request handlers
and background routines.

Known deficiency: using global state to share context between tests and other
parts of the app makes it impossible to run tests concurrenly with each other.
*/
func testInit(t TB) (Ctx, DbTx) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	conn, err := beginTx(ctx, env.db)
	require.NoError(t, err)

	return ctx, conn
}

func selfHttpFetch(ctx Ctx, sess TestSess, method string, path string, body []byte) ([]byte, error) {
	rew := httptest.NewRecorder()
	req := selfReq(ctx, sess, HttpReqParams{Method: method, Url: path, Body: body})
	handleRequest(rew, req)
	return rew.Body.Bytes(), selfReqErr(rew)
}

func selfJsonFetch(ctx Ctx, sess TestSess, method string, path string, body interface{}, output interface{}) error {
	params := JsonReqParams{Method: method, Url: path, Body: body}
	httpParams, err := params.HttpReqParams()
	if err != nil {
		return err
	}

	rew := httptest.NewRecorder()
	req := selfReq(ctx, sess, httpParams)
	handleRequest(rew, req)

	err = selfReqErr(rew)
	if err != nil {
		return err
	}

	resBody := rew.Body.Bytes()
	if output != nil {
		err = jsonUnmarshal(resBody, output)
		return errors.WithMessagef(err, "failed to JSON-decode response into value of type %T; response:\n%s", output, resBody)
	}
	return nil
}

func tSelfJsonFetch(t TB, ctx Ctx, sess TestSess, method string, path string, body interface{}, out interface{}) {
	err := selfJsonFetch(ctx, sess, method, path, body, out)
	if err != nil {
		t.Fatalf("error: %v %q: %+v", method, path, err)
	}
}

func selfReq(ctx Ctx, sess TestSess, params HttpReqParams) *Req {
	req, err := http.NewRequestWithContext(ctx, params.Method, params.Url, bytes.NewReader(params.Body))
	try.To(err)
	patchHttpHeaderMut(req.Header, params.Header)
	patchHttpHeaderMut(req.Header, sess.Header())
	return req
}

func selfReqErr(rew *httptest.ResponseRecorder) error {
	if rew.Code != 0 && !isHttpStatusOk(rew.Code) {
		return errors.Errorf(`self-request returned a non-OK status code; code: %v; body: %s`,
			rew.Code, rew.Body.Bytes())
	}
	return nil
}

func readFixture(path string) []byte {
	return try.ByteSlice(os.ReadFile(filepath.Join(`fixtures`, path)))
}
