/*
Main persistent schema.

This reflects the PRESENT state of the schema, while migrations reflect the PAST
changes of the schema. When changing this file, you MUST reflect the changes in
`migrations.pgsql`.

Stateless definitions such as views and non-trigger functions must be defined in
`schema_eph.pgsql` instead.

# Usage

The entirety of schema and migrations can be applied via:

    ./misc/db_update

    # With immediate rollback
    ./misc/db_update -r

For migrations, see `migrations.pgsql`.

This file should be viewed with horizontal scroll, without line wrapping.

# Schema Guidelines

  * Everything is lowercase.

  * When creating a new table, take examples from existing tables. Deviations
    need to ne justified.

  * Table and view names are plural: `table persons`, not `table person`.

  * ID is called `id` and is a bigserial.

  * Columns explicitly specify `null` or `not null`.

  * Prefer `not null` over `null`. For primitives such as strings, numbers,
    booleans, prefer `not null default X` where X is the zero value.

  * Approach to polymorphic relations is described below, please refer to the
    [Polymorphic Relations Guideline](#polymorphic-relations-guideline).

  * ALL tables must have `created_at`, `updated_at`, and a `touch_updated_at`
    trigger.

  * Time must be stored as a `timestamptz` rather than `timestamp` in order to
    avoid a massive Postgres gotcha: when parsing datetime from a string,
    `timestamp` TRUNCATES the input and completely ignores the timezone offset,
    parsing it AS IF the datetime part was specified in UTC time rather than
    local time. `timestamptz` avoids this problem. Note that `timestamptz`
    doesn't actually store a timezone, it only parses it, then stores UTC time.

  * Time fields must end with `_at`. Example: `created_at`, `updated_at`. The
    only exceptions are "external" fields of objects from 3rd party services
    such as Stripe. See `stripe_refunds` for examples.

  * Boolean fields must start with `is_`. Example: `is_hidden`, `is_deleted`.
    The only exceptions are "external" fields of objects from 3rd party services
    such as Stripe. See `stripe_refunds` for examples. Where possible, booleans
    should instead be timestamps, see below.

  * Where possible, use timestamps instead of booleans. For example, instead of
    `is_deleted`, store `deleted_at`. This principle applies to other columns.

  * Fields referring to entities must include the entity type in the field name.
    For example, an article author could be either `person_id` or
    `author_person_id`, but not `author_id`. The extended article view would
    contain `person` or `author_person` rather than `author`.

  * All constraints, including unique indexes, must be named like this:
    "db.constraint.<constraint_name>". This may allow the server to detect these
     errors and make them "public", and possibly convert them to a
     human-readable format. Also, named constraints and indexes can be reliably
     dropped in migrations.

  * Names of all non-unique indexes must be UUIDs without dashes. This makes
    them much easier to define while avoiding collisions.

  * Foreign keys referencing a primary key do not specify the column name.
    Foreign keys referencing a non-primary key do specify the column name.
    Examples:

        not null references timezones       on update cascade on delete cascade
        not null references timezones(name) on update cascade on delete cascade
            null references images          on update cascade on delete set null

  * Foreign key columns with `not null` must specify:

      on update cascade on delete cascade

  * Foreign key columns with `null` must specify one of:

      on update cascade on delete cascade
      on update cascade on delete set null

  * Junction/edge tables have their own `id` and a separate unique index for the
    references they contain. In the future we might consider removing their
    `ids` and simply using tuples of references as primary keys. Whichever
    approach we use, it must always be consistent between all edge tables.

  * Always use `as` for aliases, for better readability and clarity.

  * Avoid `create if not exists`, `create or replace`, etc. The schema is
    applied to an empty database. Migrations must precisely know the schema
    they're being applied to. Neither should need the "optional" clauses.

  * When creating new columns, prefer `bigint` (int8) over `int` (int4) to
    minimize surprises.

# Polymorphic Relations Guideline

To represent polymorphic relations, we use a set of foreign key fields
referencing different tables, alongside a polymorphic check constraint.

    create table entities (
      id                     bigserial     primary key,
      person_id              bigint            null references persons on update cascade on delete cascade,
      post_id                bigint            null references posts   on update cascade on delete cascade,
      org_id                 bigint            null references orgs    on update cascade on delete cascade,
      created_at             timestamptz   not null default current_timestamp,
      updated_at             timestamptz   not null default current_timestamp

      constraint "db.constraint.entities_exactly_one_subject"
      check ((
        (post_id    is not null)::int +
        (org_id     is not null)::int +
        (person_id  is not null)::int +
        0
      ) = 1)
    );

This approach has several advantages over others:

  * This preserves the correct relational data structure; we have foreign key
    constaints ensuring the integrity of the data; there's no need for a field
    that would specify the subject's table. Even if all IDs in the system were
    UUIDs, and there were never any collisions, it's not correct even from a
    conceptual point of view to mix IDs from different tables in one field.

  * We retain specialized tables for different entity types, with their own
    structure, constraints, and indexes.
*/

drop schema if exists tbl cascade;
create schema tbl;
set search_path to tbl;

/*
For cross-row constraints via `exclude using`. Implements scalar `=` comparisons
for Postgres' GIST indexes.
*/
create extension btree_gist;

/* Types */

create domain text_short as text
constraint "db.constraint.text_short_length"
check (length(value) <= 256);

create domain text_long as text
constraint "db.constraint.text_long_length"
check (length(value) <= 1 << (1 << (1 << (1 << 1))));

-- TODO more restrictive?
create domain file_name as text
constraint "db.constraint.file_name"
check (length(value) <= 256 and (value = '' or value ~ '^[^/\\]+$'));

create type pub_status as enum ('draft', 'final');

create type log_entry_type as enum ('info', 'error');

/* Functions */

/*
Requires the following fields:
  * updated_at timestamptz

TODO: consider dropping and re-creating automatically.
*/
create function touch_updated_at() returns trigger
language plpgsql as $$
begin
  new.updated_at := current_timestamp;
  return new;
end $$;

/* Tables */

/*
See the schema guidelines at the top of the file. Example:

    create table entities (
      id                           bigserial                primary key,
      created_at                   timestamptz              not null default current_timestamp,
      updated_at                   timestamptz              not null default current_timestamp
    );

    create trigger touch_updated_at
      before update on entities
      for each row execute procedure touch_updated_at();
*/