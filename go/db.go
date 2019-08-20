package main

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/mitranim/try"
)

func initDb() (err error) {
	defer try.Rec(&err)

	conn, err := sql.Open("postgres", env.conf.PostgresConnString())
	try.To(err)
	try.To(conn.Ping())
	env.db = conn

	return nil
}
