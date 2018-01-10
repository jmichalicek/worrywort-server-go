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

* docker-compose pull
* docker-compose build
* docker-compse run --service-ports worrywortd

To stop:
* docker-compose down

## Database migrations

This will eventually apply to an initial production setup as well.

Optional pre-made database migrations for use with github.com/mattes/migrate and a postgresql database are provided.  The development docker image also automatically installs migrate into `/go/bin/migrate` for easy use.

I may try to automate the postgres connection string a bit since the required data should already mostly be in environment variables.

* migrate -source file://./_migrations -database postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:5432/$DB_NAME?sslmode=disable up

### TODO:

* The rest of the GraphQL types
  * Mutations - login, put batches, put fermenter, put measurement, etc.
  * Batches list, fermenter list, etc.
* Custom DateTime type in the GraphQL Schema rather than String
* DB stuff - github.com/mgutz/dat or just sqlx?
* Actual http interface - chi or echo?
* Password reset mutation/grapql flow and then web views somewhere, change password mutation,
* Helper command line stuff to create user, manage initial data, etc.
