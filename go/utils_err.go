package main

import (
	"context"
	"database/sql"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/mitranim/rout"
	"github.com/pkg/errors"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
// https://www.postgresql.org/docs/12/errcodes-appendix.html
const (
	POSTGRES_ERROR_CODE_QUERY_CANCELED  = "57014"
	POSTGRES_ERROR_CODE_SYNTAX_ERROR    = "42601"
	POSTGRES_ERROR_CODE_CHECK_VIOLATION = "23514"
)

/*
Wraps the input in `Error`, if necessary. If the input is already an `Error`,
it's returned as-is.
*/
func Err(err error) Error {
	if err == nil {
		panic("unexpected nil error in Err")
	}
	unwrapped, ok := err.(Error)
	if ok {
		return unwrapped
	}
	return Error{Cause: err}
}

/*
Wraps the input in `Error`, if necessary, and modifies the resulting `Error` by
calling the provided function. Note that the function modifies a local copy of
the error struct; the original input is not mutated.

Returns `error` rather than `Error` to avoid some edge-case gotchas related to
interface conversions.
*/
func ErrWith(err error, fun func(*Error)) error {
	if err == nil {
		return nil
	}
	out := Err(err)
	if fun != nil {
		fun(&out)
	}
	return out
}

func ErrPub(err error) error {
	return ErrWith(err, func(err *Error) { err.IsPublic = true })
}

func ErrHttp(err error, httpStatus int) error {
	return ErrWith(err, func(err *Error) { err.HttpStatus = httpStatus })
}

func ErrPubHttp(err error, httpStatus int) error { return ErrPub(ErrHttp(err, httpStatus)) }

func ErrPubNotFound(err error) error { return ErrPubHttp(err, http.StatusNotFound) }

func ErrPubBadRequest(err error) error { return ErrPubHttp(err, http.StatusBadRequest) }

/*
Intentionally renamed to reduce the chance of someone confusing "authorization",
which in THIS case actually means "authentication", with the USUAL meaning of
"authorization" as permission checking. Use `ErrPubForbidden` for the latter.
*/
func ErrPubUnauthenticated(err error) error { return ErrPubHttp(err, http.StatusUnauthorized) }

func ErrPubForbidden(err error) error { return ErrPubHttp(err, http.StatusForbidden) }

// Can be used to expose the error message to the client.
func ErrPubInternal(err error) error { return ErrPubHttp(err, http.StatusInternalServerError) }

/*
Returns the public portion of this error, which may be either the error itself,
or an instance of `Error` wrapped by it. When an error chain contains an
`Error` that is public, only THAT particular error and its causes are
considered public. Any other errors that wrap the public error LATER are
considered non-public. For example:

	root := errors.New("root cause")
	public := error(Error{Cause: root, IsPublic: true})
	derived := errors.WithMessage(public, "some message")

	errPub(derived) == public // true

	pub, ok := errPub(derived).(Error)
	if ok { ... }

Known limitation: stops at the first `Error`, without considering any other
instances of `Error` possibly wrapped by it (we currently don't intentionally
wrap Error in Error).
*/
func errPub(err error) error {
	var unwrapped Error
	if errors.As(err, &unwrapped) && unwrapped.IsPublic {
		return unwrapped
	}
	return nil
}

func isErrWithHttpStatus(err error, httpStatus int) bool {
	var unwrapped Error
	return errors.As(err, &unwrapped) && unwrapped.HttpStatus == httpStatus
}

func isErrWithDbCode(err error, dbCode string) bool {
	var unwrapped Error
	return errors.As(err, &unwrapped) && string(unwrapped.DbCode) == dbCode
}

func isErrWithDbCodePrefix(err error, prefix string) bool {
	var unwrapped Error
	return errors.As(err, &unwrapped) && strings.HasPrefix(string(unwrapped.DbCode), prefix)
}

/*
Minor note: our DB functions auto-wrap `sql.ErrNoRows` into `Error`, but
there's no strong reason to not include an additional check for it.
*/
func isErrNotFound(err error) bool {
	if isErrWithHttpStatus(err, http.StatusNotFound) || errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return false
}

/*
Some of these errors don't LOOK like cancellation errors, but tend to be
predominantly caused by cancellation, and incorrectly obfuscated by libraries
in our stack.
*/
func isErrPossiblyCancelRelated(err error) bool {
	return isErrCancel(err) ||
		isErrDisconnect(err) ||
		isErrNetDeadlineExceeded(err) ||
		errors.Is(err, io.EOF)
}

func isErrCancel(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, sql.ErrTxDone) ||
		isPgErrWithCode(err, POSTGRES_ERROR_CODE_QUERY_CANCELED) ||
		isErrNetCancel(err)
}

/*
This is the fault of the "net" package. It artificially converts the
`context.Canceled` error into its own cancellation error, which is PRIVATE,
and doesn't wrap the original error, making it impossible to detect via
`errors.Is`.
*/
func isErrNetCancel(err error) bool {
	return err != nil && strings.Contains(err.Error(), "operation was canceled")
}

func isErrDisconnect(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) &&
		(errno == syscall.ECONNRESET || errno == syscall.EPIPE)
}

// Frequent result of context cancelation during DB reads.
func isErrNetDeadlineExceeded(err error) bool {
	var unwrapped *net.OpError
	return errors.As(err, &unwrapped) &&
		unwrapped.Op == "read" &&
		errors.Is(err, os.ErrDeadlineExceeded)
}

// Helps avoid confusion by using consistent terms.
func isErrUnauthenticated(err error) bool {
	return isErrWithHttpStatus(err, http.StatusUnauthorized)
}

func isPgErrWithCode(err error, code string) bool {
	var pgErr PgErr
	return errors.As(err, &pgErr) && string(pgErr.Code) == code
}

func isPgErrWithConstraint(err error, name string) bool {
	var pgErr PgErr
	return errors.As(err, &pgErr) && pgErr.Constraint == name
}

func isErrDbConstraint(err error) bool {
	var pgErr PgErr
	return errors.As(err, &pgErr) && pgErr.Constraint != ""
}

// TODO: use `PgErr` instead.
func isErrDbError(err error) bool {
	return isErrWithDbCodePrefix(err, string(DbCodeErrorPrefix))
}

func shouldLogError(ctx Ctx, err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation happens all the time due to client disconnect. For
	// some errors, we KNOW they're usually caused by context cancellation, and
	// logging is unnecessary. But we do want to log anything unexpected.
	if isCtxCanceled(ctx) && (isCtxHttp(ctx) || isErrPossiblyCancelRelated(err)) {
		return false
	}

	return true
}

/*
"Normalizes" errors by converting known types to `Error`, when possible.
Database errors are handled separately by `decodeDbErr`.
*/
func errNorm(err error) error {
	routErr, ok := err.(rout.Err)
	if ok {
		return Error{Cause: routErr.Cause, HttpStatus: routErr.HttpStatus, IsPublic: true}
	}
	return err
}
