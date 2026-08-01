ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_owner_fkey;

DROP INDEX IF EXISTS users_email_lower_idx;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (username);
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE accounts
ADD CONSTRAINT accounts_owner_fkey
FOREIGN KEY (owner) REFERENCES users(username)
DEFERRABLE INITIALLY IMMEDIATE;