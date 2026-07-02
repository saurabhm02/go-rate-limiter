-- Dev seed: demo tenant, API key, and example-service route rules.
-- Raw API key (documented in README): rl_demo_abc123xyz

INSERT INTO tenants (id, name, status)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'demo-corp', 'active')
ON CONFLICT (name) DO NOTHING;

INSERT INTO api_keys (id, tenant_id, key_hash, key_prefix, status)
VALUES (
    '660e8400-e29b-41d4-a716-446655440000',
    '550e8400-e29b-41d4-a716-446655440000',
    'b16ce1a03569f675496aefeafa235813dd7854f058d32f30793064a09b17fcb3',
    'rl_demo_',
    'active'
)
ON CONFLICT (key_hash) DO NOTHING;

INSERT INTO rules (id, tenant_id, route_pattern, algorithm, limit_count, window_seconds, bucket_capacity, refill_rate, enabled)
VALUES
    (
        '770e8400-e29b-41d4-a716-446655440001',
        '550e8400-e29b-41d4-a716-446655440000',
        '/v1/check',
        'sliding_window',
        100,
        60,
        NULL,
        NULL,
        true
    ),
    (
        '770e8400-e29b-41d4-a716-446655440002',
        '550e8400-e29b-41d4-a716-446655440000',
        '/api/payments*',
        'token_bucket',
        NULL,
        NULL,
        10,
        2.00,
        true
    ),
    (
        '770e8400-e29b-41d4-a716-446655440003',
        '550e8400-e29b-41d4-a716-446655440000',
        '/api/orders*',
        'sliding_window',
        50,
        60,
        NULL,
        NULL,
        true
    ),
    (
        '770e8400-e29b-41d4-a716-446655440004',
        '550e8400-e29b-41d4-a716-446655440000',
        '*',
        'sliding_window',
        1000,
        3600,
        NULL,
        NULL,
        true
    )
ON CONFLICT (tenant_id, route_pattern) DO NOTHING;
