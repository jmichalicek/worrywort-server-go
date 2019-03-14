-- WARNING: This file WILL change for awhile.  It is intended to have all initial tables
-- Creates the initial worrywort tables for a postgresql 9.6 database for github.com/mattes/migrate

-- May need to do this for generating the uuid... but may already be loaded?
CREATE EXTENSION IF NOT EXISTS pgcrypto;

BEGIN;
-- id for PK because I'm lazy and so used to standard ORM rules
-- BIGSERIAL for slim chance of running out of PKs while providing higher performance and lower
-- memory usage than uuid for PK.  Everything will get a uuid, though.
-- Use of text rather than varchar(n) based on info at https://www.postgresql.org/docs/current/static/datatype-character.html

-- Using is_admin boolean for simplicity.  Only need 2 "groups" right now, admins and users.  No point in complicating
-- user lookups and permission checks by checking the group

CREATE TABLE IF NOT EXISTS users(
  id BIGSERIAL PRIMARY KEY,
  first_name text DEFAULT '',
  last_name text DEFAULT '',
  email text DEFAULT '',
  password text DEFAULT '',
  is_active boolean DEFAULT FALSE,
  is_admin boolean DEFAULT FALSE,

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS users_email_lower_idx ON users ((lower(email)));

CREATE TABLE IF NOT EXISTS user_authtokens(
  -- token_id text PRIMARY KEY,
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  token text DEFAULT '',
  is_active boolean DEFAULT TRUE,
  user_id integer REFERENCES users (id) ON DELETE CASCADE,
  scope integer,
  expires_at timestamp with time zone DEFAULT NULL,

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);

/* I don't think this needs to worry so much about precision as to use a numeric or decimal
 * and while the volumes and gravity values are not likely to need a double, these are pretty much
 * intended to go out a graphql interface which specifies using doubles
 */
CREATE TABLE IF NOT EXISTS batches(
  id BIGSERIAL PRIMARY KEY,
  user_id integer REFERENCES users (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  brew_notes text NOT NULL DEFAULT '',
  tasting_notes text NOT NULL DEFAULT '',
  brewed_date timestamp with time zone,
  bottled_date timestamp with time zone,
  volume_boiled double precision,
  volume_in_fermentor double precision,
  volume_units integer,
  original_gravity double precision,
  final_gravity double precision,
  recipe_url text NOT NULL DEFAULT '',
  max_temperature double precision NOT NULL DEFAULT 0.0,
  min_temperature double precision NOT NULL DEFAULT 0.0,
  average_temperature double precision NOT NULL DEFAULT 0.0,

  created_at timestamp with time zone  DEFAULT current_timestamp,
  updated_at timestamp with time zone
);

-- May remove Fermentors for now - they have no real use case
CREATE TABLE IF NOT EXISTS fermentors(
  id BIGSERIAL PRIMARY KEY,
  user_id integer REFERENCES users (id) ON DELETE SET NULL,
  batch_id integer REFERENCES batches (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',
  volume double precision NOT NULL DEFAULT 0.0,
  volume_units integer NOT NULL DEFAULT 0,
  fermentor_type integer NOT NULL DEFAULT 0,
  is_active boolean DEFAULT FALSE,
  is_available boolean DEFAULT TRUE,

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS sensors(
  id BIGSERIAL PRIMARY KEY,
  user_id integer REFERENCES users (id) ON DELETE SET NULL,
  name text NOT NULL DEFAULT '',
  description text NOT NULL DEFAULT '',

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);

-- store the association between a batch and a sensor permanently rather than making it ephemeral
-- and using the temperature measurements to track that long term
CREATE TABLE IF NOT EXISTS batch_sensor_association(
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  -- used to be PK of (batch_id, sensor_id).. but maybe you disassociate then re-associate. meh.
  batch_id integer REFERENCES batches (id) ON DELETE CASCADE NOT NULL,
  sensor_id integer REFERENCES sensors (id) ON DELETE CASCADE NOT NULL,
  associated_at timestamp with time zone DEFAULT current_timestamp,
  disassociated_at timestamp with time zone,
  description text NOT NULL DEFAULT '',

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS batch_sensor_association_associated_at_index ON batch_sensor_association (associated_at);

-- Storing this here for now, but it may move to a timeseries db such as influxdb
CREATE TABLE IF NOT EXISTS temperature_measurements(
  -- use a UUID, there will be a LOT of these
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id integer REFERENCES users (id) ON DELETE SET NULL,
  -- Not sure these are really necessary now if the batch_sensor_association is used instead
  -- sensor_id will still be needed, but batch_id is really not
  sensor_id integer REFERENCES sensors (id) ON DELETE SET NULL,
  temperature double precision NOT NULL DEFAULT 0.0,
  units integer NOT NULL DEFAULT 0,
  recorded_at timestamp with time zone,

  created_at timestamp with time zone DEFAULT current_timestamp,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS temperature_measurements_recorded_index ON temperature_measurements (recorded_at);
COMMIT;
