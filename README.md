# tinyr: a barebones URL shortner

`tinyr` is a (extremely hacked-together) barebones URL shortener that allows
users to create, delete, and follow short links.

It can use four storage backends:

- In-memory (ephemeral)
- Local files (using Pebble)
- MySQL
- Cassandra (and Cassandra-like databases, like scylla, etc.)

User authentication (used for url ownership) is via OIDC (tested with Keycloak
and Google).

A command line interface is in `cli/`, the main service is in `service/`,
`helm/` contains a helm chart, and `tf/` contains a terraform config for
bringing up on GCP.

To run locally, I use Keycloak as an OIDC provider (there's a TODO to add a
fake).

```
> go run ./service/app [flags]
```
or
```
> docker run --rm -p 8080:8080 --name tinyr_svc \
  -e TINYR_ENV_VAR2='my_value' \
  -e TINYR_ENV_VAR1='my_value' \
  tinyr:latest 
```
