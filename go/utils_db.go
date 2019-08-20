package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/mitranim/gos"
	"github.com/mitranim/try"
	"github.com/pkg/errors"
)

func dbQuery(ctx Ctx, conn DbConn, out interface{}, queryStr string, args []interface{}) error {
	queryStr = formatSql(queryStr)
	err := gos.Query(ctx, conn, out, queryStr, args)
	return decodeDbErr(err, queryStr)
}

/*
Similar to `dbExec` but verifies that the output has exactly one row. Useful for
operations such as updating or deleting a single record, when you want to ensure
that the record exists and no other record is affected. Use inside a transaction
block such as `withDbTx` to automatically abort the update in such cases.

The query must allow to append a `returning` clause.
*/
func dbExecSingle(ctx Ctx, conn DbConn, queryStr string, args []interface{}) error {
	query := SqlQueryOrd(queryStr, args...)
	query.Append(`returning true`)

	var ok bool
	return query.Query(ctx, conn, &ok)
}

func dbExec(ctx Ctx, conn DbConn, queryStr string, args []interface{}) error {
	queryStr = formatSql(queryStr)
	_, err := conn.ExecContext(ctx, queryStr, args...)
	return decodeDbErr(err, queryStr)
}

func dbExecFile(ctx Ctx, conn DbConn, path string) error {
	realPath, err := filepath.Abs(path)
	if err != nil {
		return errors.WithStack(err)
	}

	content, err := os.ReadFile(realPath)
	if err != nil {
		return errors.WithStack(err)
	}

	queryStr := bytesToMutableString(content)
	queryStr = formatSql(queryStr)

	_, err = conn.ExecContext(ctx, queryStr)
	if err != nil {
		return errors.Wrapf(decodeDbErr(err, queryStr), `failed to execute %q`, path)
	}
	return nil
}

func dbExecFiles(ctx Ctx, conn DbTx, paths []string) error {
	for _, path := range paths {
		err := dbExecFile(ctx, conn, path)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
By convention, we use the following format:

	* "db.error.error_name_or_description"
	* "db.constraint.constraint_name_or_description"

The error/constraint name may be followed by an optional message, which we also
want to capture.
*/
var dbCodeRegexp = regexp.MustCompile(`"(db\.(?:constraint|error)(?:\.\w+)+)"\s*:?\s*(.*)`)

/*
TODO: consider supporting Postgres's common errors, including but not limited
to:

	* "null value in column A violates not-null constraint"
	* "A violates check constraint B"
	* etc.

Also, anonymous check constraints specified directly on columns aren't actually
anonymous; Postgres automatically gives them names using the following format:
"<table_name>_<column_name>_<check>". We might consider supporting those.
*/
func decodeDbErr(err error, queryStr string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrPubNotFound(errors.WithStack(err))
	}

	/**
	For certain Postgres errors, find the code surrounding the error and add a
	code snippet to the error message.

	Reference: https://www.postgresql.org/docs/current/protocol-error-fields.html
	*/
	var pgErr PgErr
	if errors.As(err, &pgErr) {
		err := Error{
			Cause:     errors.WithStack(err),
			IsPublic:  pgErr.Constraint != "" || pgErr.Code == POSTGRES_ERROR_CODE_CHECK_VIOLATION,
			DbQuery:   pgErr.InternalQuery,
			DbContext: strOr(pgErr.Where, pgErr.Hint),
		}

		if pgErr.Constraint != "" {
			err.DbCode = DbCode(pgErr.Constraint)
		}

		if pgErr.Code == POSTGRES_ERROR_CODE_SYNTAX_ERROR && err.DbContext == "" {
			pos, _ := strconv.Atoi(pgErr.Position)
			// TODO: find out whether it's set to 0 or -1 when Postgres doesn't report it.
			if pos > 0 {
				const hintOffset = 128
				err.DbContext = sliceStringAsChars(queryStr, pos-hintOffset, pos+hintOffset)
			}
		}

		return err
	}

	/**
	We no longer need this for constraints, but this allows to expose errors
	raised by PLPGSQL code, where `db.error.___` is embedded in the error string.
	*/
	match := dbCodeRegexp.FindStringSubmatch(err.Error())
	if len(match) > 0 {
		return Error{Cause: errors.WithStack(err), DbCode: DbCode(match[1]), IsPublic: true}
	}

	return errors.WithStack(err)
}

/*
Reformats an SQL query from multi-line to single-line for compatibility with
logging systems such as Stackdriver which expect strictly single-line messages.

`PRETTY_SQL` suppresses this behavior, and should be used only for local
development.
*/
func formatSql(input string) string {
	if env.conf.PrettySql {
		return input
	}
	return sqlToSingleLine(input)
}

/*
Runs a function within a transaction, which is either retrieved from the context
or created. If the transaction comes from the context, it's neither committed
or aborted here. If the transaction is created here, it's automatically
committed or aborted here, depending on whether the function returns an error.

Storing a transaction in the context is used mainly for testing. Tests run in a
transaction that is rolled back at the end. Supporting this here makes it
automatic for practically all our code.
*/
func withDbTx(ctx Ctx, fun func(Ctx, DbTx) error) (err error) {
	defer try.Rec(&err)

	if fun == nil {
		return errors.New("missing callback argument")
	}

	tx := ctxDbTx(ctx)
	if tx != nil {
		return fun(ctx, tx)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, tx, err = beginTxCtx(ctx, env.db)
	if err != nil {
		return err
	}

	err = fun(ctx, tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	return errors.Wrap(err, `failed to commit transaction`)
}

func withReqDbTx(req *Req, fun func(Ctx, DbTx) error) error {
	return withDbTx(req.Context(), fun)
}

func ctxDbTx(ctx Ctx) DbTx {
	tx, _ := ctx.Value(CTX_DB_TX_KEY).(DbTx)
	return tx
}

func ctxWithDbTx(ctx Ctx, conn DbTx) Ctx {
	return context.WithValue(ctx, CTX_DB_TX_KEY, conn)
}

func beginTxCtx(ctx Ctx, db DbTxer) (Ctx, DbTx, error) {
	tx, err := beginTx(ctx, db)
	return ctxWithDbTx(ctx, tx), tx, err
}

func beginTx(ctx Ctx, db DbTxer) (DbTx, error) {
	conn, err := db.BeginTx(ctx, DEFAULT_DB_TX_OPTIONS)
	return conn, errors.Wrap(err, `failed to start DB transaction`)
}
