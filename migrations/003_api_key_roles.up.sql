-- Keys now have a job.
--   check : the app uses it to call /v1/check
--   admin : the person uses it to manage the project in the console
--
-- Splitting them means a key that leaks out of a deployed app cannot be used to
-- raise that project's own limits, which is the whole point of limiting it.
--
-- Existing keys become 'check'. They are all in running apps.

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'check'
    CHECK (role IN ('admin', 'check'));

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_role ON api_keys(tenant_id, role);
