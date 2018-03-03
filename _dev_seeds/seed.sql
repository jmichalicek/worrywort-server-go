/* Create a fake user with the password 'password' */
INSERT INTO users (first_name, last_name, email, password, is_active, is_admin, updated_at)
  VALUES ('First', 'Last', 'user@example.org', '$2a$13$ziIoEVxTifUjLqxgQr6p/OyVlfKqdET9m/t5rDEzXmRcaJNjPCINW', 't', 'f', now());
