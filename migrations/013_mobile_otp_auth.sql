-- 013_mobile_otp_auth.sql
-- Phone/OTP accounts have no email or password up front; the schema so far
-- assumed email+password signup was the only path in. Relax both to nullable
-- (email keeps its UNIQUE constraint — Postgres allows multiple NULLs under a
-- UNIQUE constraint, so email-based accounts still can't collide).
ALTER TABLE public.users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE public.users ALTER COLUMN password DROP NOT NULL;

-- NOTE: a UNIQUE index on mobileno was deliberately left out of this
-- migration. This DB already has pre-existing rows sharing one mobile number
-- (e.g. '7303671362' x3) predating this change; enforcing uniqueness now
-- would require deciding how to merge/delete real account rows, which isn't
-- this migration's call to make. GetUserByMobile takes the earliest-created
-- match in the meantime. Add the constraint once those rows are reconciled.
