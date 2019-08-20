## Overview

Backend starter.

## Dependencies

* Go: https://golang.org, `brew install go`.
* Postgres: https://www.postgresql.org, `brew install postgres`.

See [Tips and Tricks](#tips-and-tricks) below.

## Development Setup

### Env Variables

    cp .env.properties.example .env.properties

Change `POSTGRES_USER` and `POSTGRES_PASSWORD` to match your local installation.

### Go

To watch source files and automatically restart, test, or lint the code, get https://github.com/mitranim/gow:

    go get -u github.com/mitranim/gow

Example commands:

    gow run ./go
    gow -c -v run ./go
    gow -c -v vet ./go
    gow -c test ./go

### Postgres

Launch in the foreground and leave running:

    /usr/local/opt/postgres/bin/postgres -D /usr/local/var/postgres

Create the database:

    sh misc/pg_reset_db

Apply schema and migrations:

    ./misc/db_update

Open an interactive REPL:

    ./misc/pg_repl

### Run Server

After the database setup, run the server:

    go run ./go
    gow -c -v run ./go

To automatically restart, test, or lint the code on changes, use the `gow` tool. See [Tips and Tricks for Go](#tips-and-tricks-for-go).

## Tips and Tricks

### Tips and Tricks for Go

To auto-format code and automatically update import declarations, get an editor plugin for `goimports`. For example, for Sublime Text, get the Fmt package and configure it for `goimports`.

To watch source files and automatically restart, test, or lint the code, get https://github.com/mitranim/gow. Install it globally:

    go get -u github.com/mitranim/gow

Run and auto-restart the server:

    gow -c -v run ./go

Run tests and rerun on changes. Use the `-count` flag to disable test caching:

    gow -c test ./go -count=1
    gow -c test ./go -count=1 -run <TestNameRegexpPattern>

Make tests verbose:

    gow -c test ./go -v

Analyze the code using `go vet`. It notices more errors than the compiler, although it reports only the first error it finds:

    gow -c -v vet ./go

### Tips and Tricks for Postgres

The `./misc` directory has a few scripts that automatically read Postgres connection variables from `.env.properties`. Use them!

#### Postgres Script Examples

Connect via REPL:

    ./misc/pg_repl

Backup the database:

    mkdir -p dumps
    ./misc/pg_dump > dump.pgsql

Restore the database:

    ./misc/pg_script -f dump.pgsql

When developing schema changes and migrations, verify them and immediately roll back to the previous version:

    ./misc/pg_verify_schema
    ./misc/pg_verify_migrations

Alternatively, backup, apply and restore manually.

#### Postgres Script Credentials

The scripts in `misc` automatically take credentials from `.env.properties` either in the current directory, or in the directory specified by the `CONF` environment variable. This allows you to store different sets of credentials in different `.env.properties` files.

For example, after you've received staging DB credentials, store them in `.env.properties` in a separate folder not committed into git:

    mkdir -p ./conf/develop
    cp .env.properties conf/develop/
    # Don't forget to update `conf/develop/.env.properties`.

Then run scripts in the staging database like this:

    CONF=conf/develop ./misc/pg_dump > dump.pgsql
    CONF=conf/develop ./misc/pg_repl
