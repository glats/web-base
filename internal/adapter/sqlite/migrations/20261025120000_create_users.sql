-- migrate:up
CREATE TABLE users (
    id    TEXT NOT NULL PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);

-- migrate:down
DROP TABLE users;
