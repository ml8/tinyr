#!/bin/sh

# To be run against test db to generate schema structs.
go run github.com/scylladb/gocqlx/v2/cmd/schemagen -cluster=$1 -keyspace=tinyr -output=./ -pkgname=cqlschema
