#!/bin/sh

set -exuo pipefail

printf 'Migating %s\n' $TINYR_DB
case $TINYR_DB in
  "cql")
    migratecql -schema_dir /bin/schemas/cql -cqlHosts $TINYR_CQLHOSTS
    ;;
  "sql")
    migratesql -schema_dir /bin/schemas/sql -connStr $TINYR_CONNSTR
    ;;
  *)
    printf 'No DB migration for %s\n' $TINYR_DB
    ;;
esac


printf 'Starting...\n'

tinyr
