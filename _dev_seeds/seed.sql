/* Create a fake user with the password 'password' */
INSERT INTO users (id, first_name, last_name, email, password, is_active, is_admin, updated_at)
  VALUES (1, 'First', 'Last', 'user@example.org', '$2a$13$ziIoEVxTifUjLqxgQr6p/OyVlfKqdET9m/t5rDEzXmRcaJNjPCINW', 't', 'f', now())
  ON CONFLICT DO NOTHING;

INSERT INTO batches (id, created_by_user_id, name, tasting_notes, brewed_date, bottled_date, volume_boiled, volume_in_fermenter, volume_units, original_gravity, final_gravity, updated_at)
  VALUES (1, 1, 'Seeded Brew', 'Tastes good', now() - interval '24 hours', now() - interval '1 hour', 2, 2, 1, 1.060, 1.020, now())
  ON CONFLICT DO NOTHING;
