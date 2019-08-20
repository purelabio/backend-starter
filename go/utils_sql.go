package main

import (
	"github.com/mitranim/sqlb"
	"github.com/mitranim/sqlp"
	"github.com/mitranim/try"
)

func Cols(val interface{}) string {
	defer try.Trace()
	return sqlb.Cols(val)
}

// Alias for embedding. Disambiguates embedded field from method.
type SqlbQuery = sqlb.Query

func SqlQueryOrd(code string, args ...interface{}) SqlQuery {
	defer try.Trace()
	return SqlQuery{sqlb.QueryOrd(code, args...)}
}

func SqlQueryNamed(code string, args Dict) SqlQuery {
	defer try.Trace()
	return SqlQuery{sqlb.QueryNamed(code, args)}
}

/*
Wraps `sqlb.Query`, adding stacktraces for errors/panics and some additional
methods.
*/
type SqlQuery struct{ SqlbQuery }

func (self *SqlQuery) Append(code string, args ...interface{}) {
	defer try.Trace()
	self.SqlbQuery.Append(code, args...)
}

func (self *SqlQuery) AppendNamed(code string, args Dict) {
	defer try.Trace()
	self.SqlbQuery.AppendNamed(code, args)
}

func (self SqlQuery) QueryCols(ctx Ctx, conn DbConn, out interface{}) error {
	self.WrapSelectCols(out)
	return self.Query(ctx, conn, out)
}

func (self SqlQuery) Query(ctx Ctx, conn DbConn, out interface{}) error {
	return dbQuery(ctx, conn, out, self.String(), self.Args)
}

func (self SqlQuery) Exec(ctx Ctx, conn DbConn) error {
	return dbExec(ctx, conn, self.String(), self.Args)
}

func (self SqlQuery) ExecTx(ctx Ctx, conn DbTx) error {
	return self.Exec(ctx, conn)
}

func (self SqlQuery) ExecSingle(ctx Ctx, conn DbConn) error {
	return dbExecSingle(ctx, conn, self.String(), self.Args)
}

func (self SqlQuery) ExecTxSingle(ctx Ctx, conn DbTx) error {
	return self.ExecSingle(ctx, conn)
}

func (self *SqlQuery) WrapSelectCols(out interface{}) {
	self.WrapSelect(Cols(out))
}

func sqlToSingleLine(input string) string {
	var buf []byte
	tokenizer := sqlp.Tokenizer{Source: input}

	for {
		node := tokenizer.Next()
		if node == nil {
			break
		}

		switch node := node.(type) {
		case sqlp.NodeCommentLine, sqlp.NodeCommentBlock:
			continue
		case sqlp.NodeWhitespace:
			buf = append(buf, ' ')
		default:
			node.Append(&buf)
		}
	}

	return bytesToMutableString(buf)
}
