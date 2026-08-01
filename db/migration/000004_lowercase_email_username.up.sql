-- Drop foreign key first because users.username is referenced by accounts.owner
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_owner_fkey;

-- Drop constraints/indexes that depend on username/email uniqueness
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_email_lower_idx;

-- Normalize existing data
UPDATE users
SET
  username = lower(username),
  email = lower(email);

-- Recreate username primary key
ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (username);

-- Recreate case-insensitive unique email protection
CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

-- Restore foreign key
ALTER TABLE accounts
ADD CONSTRAINT accounts_owner_fkey
FOREIGN KEY (owner) REFERENCES users(username)
DEFERRABLE INITIALLY IMMEDIATE;