/*
Ephemeral schema for stateless entities like views and functions. Unlike the
main persistent schema, this one is intended to be dropped and re-created on
each DB update.

Note that views and functions used by triggers should be defined in the main
schema file (not here) and migrated alongside stateful definitions.
*/

drop schema if exists eph cascade;
create schema eph;
set search_path to eph, starter;

/* Functions */

-- ...

/* Views */

-- ...
