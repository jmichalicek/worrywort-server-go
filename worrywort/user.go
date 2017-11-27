package worrywort;
// Models and functions for user management

type user struct {
    first_name string
    last_name string
    email string

    created_at time.Date
    updated_at time.Date
}
