DROP INDEX IF EXISTS idx_api_keys_tenant_role;
ALTER TABLE api_keys DROP COLUMN IF EXISTS role;
