INSERT INTO users (id, name, email) VALUES
    ('demo-ada', 'Ada Lovelace', 'ada@example.com'),
    ('demo-grace', 'Grace Hopper', 'grace@example.com')
ON CONFLICT DO NOTHING;
