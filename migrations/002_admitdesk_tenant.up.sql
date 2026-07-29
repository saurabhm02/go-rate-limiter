-- AdmitDesk tenant, API key, and route rules.
--
-- Only the SHA-256 hash of the API key lives here; the raw key is handed over
-- out of band and stored in the deploy host's secret store. Rotating the key
-- means adding a migration that inserts the new hash and revokes this row.
--
-- Rules are per-route-prefix. AdmitDesk sends the bucket discriminator (client
-- IP, account id) in the request's `subject` field, which splits the counter
-- without affecting which rule matches.

INSERT INTO tenants (id, name, status)
VALUES ('8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11', 'admitdesk', 'active')
ON CONFLICT (name) DO NOTHING;

INSERT INTO api_keys (id, tenant_id, key_hash, key_prefix, status)
VALUES (
    '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d12',
    '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11',
    'cf3186cdc08eee7c8c16b40d70c34d330941a861c9a64e8afa2870d607cd22bb',
    'rl_admitdesk_',
    'active'
)
ON CONFLICT (key_hash) DO NOTHING;

INSERT INTO rules (id, tenant_id, route_pattern, algorithm, limit_count, window_seconds, bucket_capacity, refill_rate, enabled)
VALUES
    -- public application form: 5 per 10 minutes, per IP
    (
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d21',
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11',
        '/admitdesk/intake/*',
        'sliding_window',
        5,
        600,
        NULL,
        NULL,
        true
    ),
    -- college signup: 3 per 60 minutes, per IP
    (
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d22',
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11',
        '/admitdesk/signup/*',
        'sliding_window',
        3,
        3600,
        NULL,
        NULL,
        true
    ),
    -- login attempts: 10 per 15 minutes, per account
    (
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d23',
        '8a1f1c62-7b5e-4d0a-9d1e-0c2a5f3b6d11',
        '/admitdesk/login/*',
        'sliding_window',
        10,
        900,
        NULL,
        NULL,
        true
    )
ON CONFLICT (tenant_id, route_pattern) DO NOTHING;
