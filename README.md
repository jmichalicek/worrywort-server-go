[![Build Status](https://travis-ci.org/jmichalicek/worrywort-server-go.svg?branch=master)](https://travis-ci.org/jmichalicek/worrywort-server-go)

# WorryWort

A beer fermentation logging and alerting system

This started life as an experiment in rewriting the current WorryWort server Elixir/Phoenix codebase in go, which in itself was a toy project to learn Elixir. Similar products have poppped up, but I feel this is still worthwhile.

## Open Source Involvement

I do not currently desire to grow a large open source community of contributors. Maintaining an open source community is very different from just maintaining code and a platform to run it. I do want the system to be open and available for use, though. I have built it and my entire career on freely available open source software. That said, if there's an obviously better way something could or should be done, let me know.

# Testing/Development

* Install GraphiQL app from https://github.com/skevy/graphiql-app/ for easy testing
* Use github.com/mattes/migrate for db migrations.  The dev dockerfile installs this into /go/bin/.
  * May test out pressly/goose as well
* DATA-DOG/txdb is now used.  For now it assumes a database named `worrywort_test` and a user and password
  matching the main user and password for the db

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

### Roadmap:

* improve core worrywortd server code a bit to be more testable
* graphql user registration flow and supporting bits
* graphql user management - password reset, update profile stuff, etc.
* graphql fleshed out - several things need to be able to be updated, etc.
* graphql subscriptions for temperature readings for a batch
* push notifications to upcoming mobile apps for sensor reading alerts
* webhook notifications for sensor reading alerts
* support for manual temperature readings
* support for sensor associated with multiple batches - ambient air in a room or chamber, etc.
* integration with tilt sensor
* nice way of handling/distributing migrations?


### Maybe TODO:
* Potential use of InfluxDb or TimescaleDB for temperature readings
* just use db url for db connection
