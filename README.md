[![Build Status](https://travis-ci.org/jmichalicek/worrywort-server-go.svg?branch=master)](https://travis-ci.org/jmichalicek/worrywort-server-go)

# WorryWort

This is an experiment in rewriting the current WorryWort server Elixir/Phoenix codebase in go.


# Testing/Development

* Install GraphiQL app from https://github.com/skevy/graphiql-app/ for easy testing
* Use github.com/mattes/migrate for db migrations.  The dev dockerfile installs this into /go/bin/.
  * May test out pressly/goose as well

## Starting development docker container/compose stack

For easy development, use `docker-compose run` to get an active shell in a golang-stretch container connected to a postgresql 9.6 and redis container.  A database for worrywortd will already be created.  The database data is in a named volume to make it easy to recognize in the `docker volume ls` output and to delete for starting over if needed.

To start development:

* `docker-compose pull`
* `docker-compose build`
* `make docker-dev`

To stop:
* ctrl-d until you have exited out of docker
* `docker-compose down` to tear down the stack

## Database migrations

This will eventually apply to an initial production setup as well.

Optional pre-made database migrations for use with github.com/mattes/migrate and a postgresql database are provided.  The development docker image also automatically installs migrate into `/go/bin/migrate` for easy use.  Several make commands have also been added to simplify migrations.  These each take an optional argument `target` with the target migration number.

* `make migrate-up` or `make migrate -target=3` to migrate up to migration 3.
* `make migrate-force`
* `make migrate-down`


## Database seeds

migrate does not play well with separate data for seeds, so a seed.sql file is provided in  _dev_seeds/seed.sql.  The make command `make seed-dev` will run it to create any seed data. Currently a dev user is seeded with the email `user@example.org` and a password of `password`.

## Logging into the server

You'll need to log in when first starting the server up to generate a token.  The easiest way to do this is to use GraphiQL as a client.  You can then run the mutation:

```
mutation logIn($username: String!, $password: String!) {
  login(username: $username, password: $password) {
    token
  }
}
```

### TODO:

* Auth integration with ory hydra
  * Started but need to do it.  Make some middleware like this https://github.com/janekolszak/gin-hydra/blob/master/ginhydra.go
  * Pluggable auth so that auth0 could be used?
* The rest of the GraphQL types
  * Mutations - login, put batches, put fermenter, put measurement, etc.
  * Batches list, fermenter list, etc.
* Custom DateTime type in the GraphQL Schema rather than String
* DB stuff - github.com/mgutz/dat or just sqlx?
* Actual http interface - chi or echo?
* Password reset mutation/grapql flow and then web views somewhere, change password mutation,
* Helper command line stuff to create user, manage initial data, etc.
