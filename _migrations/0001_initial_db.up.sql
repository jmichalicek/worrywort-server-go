/* WARNING: This file WILL change for awhile.  It is intended to have all initial tables */
/* Creates the initial worrywort tables for a postgresql 9.6 database for github.com/mattes/migrate */

/* id for PK because I'm lazy and so used to standard ORM rules */
/* SERIAL and not BIGSERIAL because graphql only does 32 bit signed int anyway */
/* Use of text rather than varchar(n) based on info at https://www.postgresql.org/docs/current/static/datatype-character.html */
/* Using is_admin boolean for simplicity.  Only need 2 "groups" right now, admins and users.  No point in complicating
 * user lookups and permission checks by checking the group
 */
CREATE TABLE IF NOT EXISTS users(
  id SERIAL PRIMARY KEY,
  first_name text DEFAULT "",
  last_name text DEFAULT "",
  email text DEFAULT "",
  is_active boolean DEFAULT FALSE,
  is_admin boolean DEFAULT FALSE,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);
CREATE INDEX users_email_lower_idx ON users ((lower(email)));

CREATE TABLE IF NOT EXISTS user_authtokens(
  token text PRIMARY KEY,
  is_active boolean DEFAULT TRUE,
  user_id integer REFERENCES users (id) ON DELETE CASCADE,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);

/* TODO:
 * batches
 * fermenters
 * temp sensors/thermometers
 * temp recordings
 */
