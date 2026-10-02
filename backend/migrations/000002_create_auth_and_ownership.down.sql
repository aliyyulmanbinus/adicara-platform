DROP INDEX IF EXISTS invitations_user_status_idx;
DROP INDEX IF EXISTS invitations_user_id_idx;
ALTER TABLE invitations DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS sessions;
DROP INDEX IF EXISTS users_email_lower_idx;
DROP TABLE IF EXISTS users;
