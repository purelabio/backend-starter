/*
# Overview

This file contains migrations for persistent entities, such as types, tables,
triggers, and indexes. Ephemeral entities such as views and non-trigger
functions are defined in `schema_eph.pgsql` and refreshed on each DB update
without the need for migrations.

# Usage

    ./misc/db_update

    # With immediate rollback
    ./misc/pg_verify_migrations

# Adding a Migration

  * Copy-paste the template at the bottom.

  * Write migration statements.

When developing migrations, test them in "immediate rollback" mode:

    ./misc/pg_verify_migrations

Dump-and-restore may also be useful:

    ./misc/pg_dump > dump.pgsql
    ./misc/pg_script -f dump.pgsql

# Removing Migrations

The source of truth is the schema, not the migrations. A migration may be
removed after it's been run on every deployed database and every developer's
machine.

Migrations must have unique names to avoid collisions with migrations that were
removed in the past. Prefix a migration name with the current date or a UUID.

# Writing Migration Statements

With VERY RARE exceptions, statements must not contain "cascade". Explicitly
drop the dependencies.

Prefer `drop` + `create` over `create or replace` to avoid surprises.

Avoid `if not exists` for anything other than schemas and extensions:

  * Migrations must exactly know the schema they're being applied to.
  * Migrations must be applied only once.
  * Migrations are not idempotent.
  * Migrations are not reversible.
*/

create schema if not exists starter;
set search_path to starter;

create table if not exists migrations (
  id         bigserial   primary key,
  name       text        not null unique check (name <> ''),
  created_at timestamptz not null default current_timestamp
);

create or replace function should_run_new_migration(_migrations_exist bool, _name text)
returns bool
language plpgsql as $$
declare
  _is_registered bool := exists(select from migrations where name = _name);
  _should_run bool := _migrations_exist and not _is_registered;
begin
  /*
  This relies on transactional execution. If this wasn't transactional, failed
  transactions would still be registered, and never retried.
  */
  if not _is_registered then
    insert into migrations (name) values (_name);
  end if;

  return _should_run;
end $$;

/*
This runs all migrations in a transactional block, just in case the script is
not executed transactionally.

After applying the schema for the first time, the application also immediately
runs this file. On the first run, all migrations are registered as completed
without actually running them. On subsequent runs, only new migrations apply.
*/
do $file$
declare
  migrations_exist bool := exists(select from migrations);
begin



/*
TEMPLATE: DO NOT REMOVE OR EDIT



if should_run_new_migration(migrations_exist, '') then
  -- ...
end if;



*/

end $file$;
