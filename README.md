# WorryWort

This is an experiment in rewriting the current WorryWort server Elixir/Phoenix codebase in go.


### TODO:

* The rest of the GraphQL types
  * Mutations - login, put batches, put fermenter, put measurement, etc.
  * Batches list, fermenter list, etc.
* Custom DateTime type in the GraphQL Schema rather than String
* Auth stuff
* DB stuff - github.com/mgutz/dat ?
* Actual http interface - chi or echo?
