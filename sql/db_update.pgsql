/*
Usage:

    # Simplified:
    ./misc/db_update

    # Direct:
    ./misc/pg_script -f sql/db_update.pgsql

    # Validation with rollback:
    ./misc/pg_verify_schema
    ./misc/pg_verify_migrations
*/

select not exists(
  select from information_schema.schemata where schema_name = 'tbl'
) as no_schema
\gset

/*
This is done to avoid accidentally referencing ephemeral entities in persistent
entities. `eph` is dropped and re-created after applying migrations.
Accidentally referencing `eph` in persistent entities, such as tables, would
cause them to be dropped when `eph` is refreshed.
*/
drop schema if exists eph cascade;

\if :no_schema
  \ir schema_tbl.pgsql
\endif

\ir migrations.pgsql
\ir schema_eph.pgsql

set search_path to tbl, eph, public;
