package main

import (
	"context"
	"database/sql"
	"math"
	"net/http"
	"reflect"

	"github.com/mitranim/sqlb"
	"github.com/pkg/errors"
)

// Type aliases for brevity.
type (
	Rew  = http.ResponseWriter
	Req  = http.Request
	Ctx  = context.Context
	Dict = map[string]interface{}
	Args = sqlb.NamedArgs
	Res  = http.Handler
)

// Function aliases for brevity. TODO avoid `var` for functions; it prevents
// many optimizations.
var (
	Arg = sqlb.Named
)

// `Limit` is nullable in order to differentiate missing from 0. We fall back on
// the default limit only if no value was provided. Limit 0 can be useful for
// requesting a feed just for its totals.
type FeedParams struct {
	Limit      *uint64   `json:"limit"`
	Offset     uint64    `json:"offset"`
	NoPageInfo bool      `json:"noPageInfo"`
	Orderings  sqlb.Ords `json:"orderings"`
}

func (self FeedParams) ValidLimit() uint64 {
	if self.Limit == nil {
		return FEED_SIZE_DEFAULT
	}

	if *self.Limit > FEED_SIZE_MAX {
		return FEED_SIZE_MAX
	}

	return *self.Limit
}

func (self FeedParams) ValidOffset() uint64 {
	return self.Offset
}

type Feed struct {
	Items    interface{} `json:"items"`
	PageInfo PageInfo    `json:"pageInfo"`
}

type PageInfo struct {
	Total      uint64  `json:"total"`
	TotalPages uint64  `json:"totalPages"`
	Page       uint64  `json:"page"`
	PerPage    uint64  `json:"perPage"`
	NextPage   *uint64 `json:"nextPage"`
	PrevPage   *uint64 `json:"prevPage"`
	HasNext    bool    `json:"hasNext"`
	HasPrev    bool    `json:"hasPrev"`
}

func PageInfoFrom(limit uint64, offset uint64, total uint64) PageInfo {
	var totalPages uint64 = 1
	if limit > 0 {
		totalPages = uint64(math.Ceil(float64(total) / float64(limit)))
	}

	// 0-indestarter for now. Should pages be 1-indestarter? (At this point probably not.)
	var page uint64
	if limit > 0 {
		page = offset / limit
	}

	var nextPage *uint64
	if total > 0 && totalPages-1 > page {
		val := page + 1
		nextPage = &val
	}

	var prevPage *uint64
	if total > 0 && page > 0 {
		val := page - 1
		prevPage = &val
	}

	return PageInfo{
		Total:      total,
		TotalPages: totalPages,
		Page:       page,
		PerPage:    limit,
		NextPage:   nextPage,
		PrevPage:   prevPage,
		HasNext:    nextPage != nil,
		HasPrev:    prevPage != nil,
	}
}

/*
Integer-based ID. Should be used for Postgres integer keys.
*/
type IntId int64

func (self *IntId) IsValid() bool {
	return self != nil && *self > 0
}

func (self *IntId) Validate() error {
	if self == nil {
		return errors.Errorf("missing integer ID")
	}
	if !self.IsValid() {
		return errors.Errorf("integer ID %v doesn't appear to be valid", *self)
	}
	return nil
}

func (self IntId) Maybe() *IntId {
	if self.IsValid() {
		return &self
	}
	return nil
}

func (self IntId) String() string {
	return intToString(int64(self))
}

type IntIds []IntId

func (self *IntIds) IsEmpty() bool {
	return self == nil || len(*self) == 0
}

func (self IntIds) Strings() []string {
	var out []string
	for _, val := range self {
		out = append(out, val.String())
	}
	return out
}

type Db interface {
	DbConn
	DbTxer
}

type DbTxer interface {
	BeginTx(Ctx, *sql.TxOptions) (*sql.Tx, error)
}

type DbConn interface {
	QueryContext(Ctx, string, ...interface{}) (*sql.Rows, error)
	ExecContext(Ctx, string, ...interface{}) (sql.Result, error)
}

type DbTx interface {
	DbConn
	Commit() error
	Rollback() error
}

type DbCode string

// Must be kept in sync with the conventions used by out schema, and with
// `dbCodeRegexp`.
const DbCodeErrorPrefix DbCode = "db.error."

// For app subcommands.
type cmd struct{ fun func() error }

func (self cmd) Execute([]string) error { return self.fun() }

/*
Implemented by types that define "inputs" to HTTP endpoints. The returned error
doesn't need to be "public" (see `Error.IsPublic`). The caller takes care of
making the error public and assigning the appropriate HTTP status code.

When appropriate, this may modify the value before validating it. Types with
string fields may trim those strings before validating their length. Because of
this, many types will define this as a pointer method, not a value method.
*/
type Validator interface {
	Validate() error
}

// Interface for data types that can refetch themselves from the DB.
type DbFiller interface {
	DbFill(Ctx, DbConn) error
}

var extendableRtype = reflect.TypeOf((*DbFiller)(nil)).Elem()
