[![Build Status](https://travis-ci.org/jmichalicek/worrywort-server-go.svg?branch=master)](https://travis-ci.org/jmichalicek/worrywort-server-go)

# WorryWort

This is an experiment in rewriting the current WorryWort server Elixir/Phoenix codebase in go.


# Testing/Development

* Install GraphiQL app from https://github.com/skevy/graphiql-app/ for easy testing

### TODO:

* The rest of the GraphQL types
  * Mutations - login, put batches, put fermenter, put measurement, etc.
  * Batches list, fermenter list, etc.
* Custom DateTime type in the GraphQL Schema rather than String
* DB stuff - github.com/mgutz/dat or just sqlx?
* Actual http interface - chi or echo?
* Password reset mutation/grapql flow and then web views somewhere, change password mutation,
* Helper command line stuff to create user, manage initial data, etc.
