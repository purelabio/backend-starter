package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

type PgErr = *pq.Error

/*
Error type used throughout the application. Includes details such as HTTP code,
public or private, and more.

Errors with `.IsPublic = true` may be exposed to clients (without a stacktrace).
All other errors should be hidden from clients.

The innermost `.Cause` must have a stack trace; it can be `errors.New()`,
`errors.Errorf()`, `errors.WithStack()`, etc.

TODO: when formatting the error with `%+v`, detect inner `PgErr` and
print its details, such as the failing statement.
*/
type Error struct {
	Cause      error  `json:"cause"`
	IsPublic   bool   `json:"isPublic"`
	HttpStatus int    `json:"httpStatus"`
	DbCode     DbCode `json:"dbCode"`
	DbQuery    string `json:"dbQuery"`
	DbContext  string `json:"dbContext"`
}

func (self Error) Error() string {
	var buf strings.Builder
	self.writeError(&buf)
	return buf.String()
}

// Implement a hidden interface in "errors".
func (self Error) Unwrap() error {
	return self.Cause
}

// Implement a hidden interface in "github.com/pkg/errors".
func (self Error) StackTrace() errors.StackTrace {
	impl, ok := self.Cause.(interface{ StackTrace() errors.StackTrace })
	if ok {
		return impl.StackTrace()
	}
	return nil
}

func (self Error) Format(fms fmt.State, verb rune) {
	if verb == 'v' && fms.Flag('+') {
		self.writeErrorVerbose(fms)
	} else {
		self.writeError(fms)
	}
}

func (self Error) HttpStatusCode() int {
	if self.HttpStatus > 0 {
		return self.HttpStatus
	}
	if self.DbCode != "" {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func (self Error) writeError(out io.Writer) {
	self.writeErrorShallow(out)
	if self.Cause != nil {
		fmt.Fprintf(out, `: %v`, self.Cause)
	}
}

func (self Error) writeErrorShallow(out io.Writer) {
	_, _ = io.WriteString(out, `error`)
	if self.HttpStatus != 0 {
		fmt.Fprintf(out, ` (HTTP status %v)`, self.HttpStatus)
	}
	if self.DbCode != "" {
		fmt.Fprintf(out, ` (DB code %v)`, self.DbCode)
	}
}

func (self Error) writeErrorVerbose(out io.Writer) {
	self.writeErrorShallow(out)
	if self.DbQuery != "" {
		fmt.Fprintf(out, "DB query or statement:\n%v\n", self.DbQuery)
	}
	if self.DbContext != "" {
		fmt.Fprintf(out, "DB context:\n%v\n", self.DbContext)
	}
	if self.Cause != nil {
		fmt.Fprintf(out, `cause: %+v`, self.Cause)
	}
}
