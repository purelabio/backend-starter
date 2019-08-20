package main

import (
	"reflect"

	"github.com/mitranim/refut"
	"github.com/mitranim/sqlb"
	"github.com/mitranim/try"
)

/*
Fetches a feed's total and items.
*/
func dbGetFeedUnpaged(
	ctx Ctx, conn DbTx, queryTotal, queryItems SqlQuery, feed *Feed, params FeedParams,
) error {
	err := dbMaybeGetFeedTotal(ctx, conn, queryTotal, feed, params)
	if err != nil {
		return err
	}
	if params.ValidLimit() == 0 {
		return nil
	}
	return queryItems.QueryCols(ctx, conn, feed.Items)
}

/*
Fetches a feed by fetching primary keys and calling `.DbFill` on every item.
The items query is expected to produce only primary keys. Every item in
`feed.Items` is expected to implement `DbFiller`.
*/
func dbGetFeedWithFill(
	ctx Ctx, conn DbTx, queryTotal, queryItems SqlQuery, feed *Feed, params FeedParams,
) error {
	err := dbMaybeGetFeedTotal(ctx, conn, queryTotal, feed, params)
	if err != nil {
		return err
	}
	if params.ValidLimit() == 0 {
		return nil
	}
	return dbQueryWithFill(ctx, conn, queryItems, feed.Items)
}

/*
Populates the feed's total. Example:

	feed := Feed{Items: new([]SomeType)}
	try.To(dbGetFeedTotal(ctx, conn, query, &feed))

Accepts `&feed` rather than `&feed.PageInfo.Total` to keep call sites simpler.
*/
func dbGetFeedTotal(ctx Ctx, conn DbTx, query SqlQuery, feed *Feed) error {
	query = qCount(query)
	return query.Query(ctx, conn, &feed.PageInfo.Total)
}

func dbMaybeGetFeedTotal(ctx Ctx, conn DbTx, query SqlQuery, feed *Feed, params FeedParams) error {
	if params.NoPageInfo {
		return nil
	}

	err := dbGetFeedTotal(ctx, conn, query, feed)
	if err != nil {
		return err
	}
	feed.PageInfo = PageInfoFrom(params.ValidLimit(), params.ValidOffset(), feed.PageInfo.Total)
	return nil
}

func qCount(query SqlQuery) SqlQuery {
	query.WrapSelect(`count(*)`)
	return query
}

func qPaging(params FeedParams) SqlQuery {
	var query SqlQuery

	limit := params.ValidLimit()
	if limit > 0 {
		query.Append(`limit $1`, limit)
	}

	offset := params.ValidOffset()
	if offset > 0 {
		query.Append(`offset $1`, offset)
	}

	return query
}

var qEmpty SqlQuery

func maybeEmptyQuery(query sqlb.IQuery) sqlb.IQuery {
	if query == nil {
		return qEmpty
	}
	return query
}

/*
Take a slice of values that may implement `DbFiller`, and calls `.DbFill` on
each of them. Uses reflection because Go doesn't have an easy way to convert a
slice of `[]X` where `X: SomeInterface` to `[]SomeInterface`, nor does it allow
to write a type-safe function generic over X, yet.

The performance cost of doing this via reflection, rather than hardcoded, is not
even measurable. The main issue is the lack of static validation.
*/
func maybeDbFillSlice(ctx Ctx, conn DbTx, items interface{}) (err error) {
	defer try.Rec(&err)

	rval := refut.RvalDeref(reflect.ValueOf(items))
	try.To(validateRkind(rval.Kind(), reflect.Slice))

	for i := 0; i < rval.Len(); i++ {
		try.To(maybeDbFillRval(ctx, conn, rval.Index(i)))
	}
	return nil
}

func maybeDbFillRval(ctx Ctx, conn DbTx, rval reflect.Value) error {
	if !rval.Type().Implements(extendableRtype) && rval.CanAddr() {
		rval = rval.Addr()
	}
	if rval.Type().Implements(extendableRtype) {
		return rval.Interface().(DbFiller).DbFill(ctx, conn)
	}
	return nil
}

func maybeDbFill(ctx Ctx, conn DbTx, val interface{}) error {
	extendable, _ := val.(DbFiller)
	if extendable != nil {
		return extendable.DbFill(ctx, conn)
	}
	return nil
}

/*
Queries a slice of rows in two steps:

	* Query keys. (The provided query should produce only primary keys or
	  equivalent.)

	* Call `.DbFill` on each element (see `DbFiller`).

Caution: this abstraction is poorly chosen. It magically "knows" to use `.Query`
rather than `.QueryCols` for the first query. It's likely to lead to bugs.
Should be revised and replaced.
*/
func dbQueryWithFill(ctx Ctx, conn DbTx, query SqlQuery, items interface{}) error {
	err := query.Query(ctx, conn, items)
	if err != nil {
		return err
	}
	return maybeDbFillSlice(ctx, conn, items)
}
