-- WARNING: This file WILL change for awhile.  It is intended to have all initial tables
-- Creates the initial worrywort tables for a postgresql 9.6 database for github.com/mattes/migrate

-- May need to do this for generating the uuid... but may already be loaded?
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- id for PK because I'm lazy and so used to standard ORM rules
-- SERIAL and not BIGSERIAL because graphql only does 32 bit signed int anyway
-- Use of text rather than varchar(n) based on info at https://www.postgresql.org/docs/current/static/datatype-character.html

-- Using is_admin boolean for simplicity.  Only need 2 "groups" right now, admins and users.  No point in complicating
-- user lookups and permission checks by checking the group

CREATE TABLE IF NOT EXISTS users(
  id SERIAL PRIMARY KEY,
  first_name text DEFAULT '',
  last_name text DEFAULT '',
  email text DEFAULT '',
  password text DEFAULT '',
  is_active boolean DEFAULT FALSE,
  is_admin boolean DEFAULT FALSE,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS users_email_lower_idx ON users ((lower(email)));

CREATE TABLE IF NOT EXISTS user_authtokens(
  token text PRIMARY KEY,
  is_active boolean DEFAULT TRUE,
  user_id integer REFERENCES users (id) ON DELETE CASCADE,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);

/* I don't think this needs to worry so much about precision as to use a numeric or decimal
 * and while the volumes and gravity values are not likely to need a double, these are pretty much
 * intended to go out a graphql interface which specifies using doubles
 */
/* May make all of the doubles/ints here NOT NULL DEFAULT 0 but I am not sure.
* It will work more cleanly with Go's defaults and 0 is rarely a valid value
* but I would rather differentiate between not set and 0.
*/
CREATE TABLE IF NOT EXISTS batches(
  id SERIAL PRIMARY KEY,
  created_by_user_id integer REFERENCES users (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  brew_notes text NOT NULL DEFAULT '',
  tasting_notes text NOT NULL DEFAULT '',
  brewed_date timestamp with time zone,
  bottled_date timestamp with time zone,
  volume_boiled double precision,
  volume_in_fermenter double precision,
  volume_units integer,
  original_gravity double precision,
  final_gravity double precision,
  recipe_url text NOT NULL DEFAULT '',
  max_temperature double precision NOT NULL DEFAULT 0.0,
  min_temperature double precision NOT NULL DEFAULT 0.0,
  average_temperature double precision NOT NULL DEFAULT 0.0,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS fermenters(
  id SERIAL PRIMARY KEY,
  created_by_user_id integer REFERENCES users (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  volume double precision NOT NULL DEFAULT 0.0,
  volume_units integer NOT NULL DEFAULT 0,
  fermenter_type integer NOT NULL DEFAULT 0,
  is_active boolean DEFAULT FALSE,
  is_available boolean DEFAULT TRUE,

  created_at timestamp with time zone,
  updated_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS thermometers(
  id SERIAL PRIMARY KEY,
  created_by_user_id integer REFERENCES users (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  created_at timestamp with time zone,
  updated_at timestamp with time zone
);

-- Storing this here for now, but it may move to a timeseries db such as influxdb
CREATE TABLE IF NOT EXISTS temperature_measurements(
  -- use a UUID, there will be a LOT of these
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  created_by_user_id integer REFERENCES users (id) ON DELETE SET NULL,
  batch_id integer REFERENCES batches (id) ON DELETE SET NULL,
  thermometer_id integer REFERENCES thermometers (id) ON DELETE SET NULL,
  fermenter_id integer REFERENCES fermenters (id) ON DELETE SET NULL,
  temperature double precision NOT NULL DEFAULT 0.0,
  units integer NOT NULL DEFAULT 0,
  recorded_at timestamp with time zone,
  created_at timestamp with time zone,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS temperature_measurements_recorded_index ON temperature_measurements (recorded_at);