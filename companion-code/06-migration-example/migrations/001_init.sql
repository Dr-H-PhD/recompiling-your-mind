-- Shared schema between PHP and Go
-- Both applications use the same tables

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- Seed data
INSERT INTO users (name, email, password) VALUES
    ('Alice', 'alice@example.com', '$2y$10$hashed'),
    ('Bob', 'bob@example.com', '$2y$10$hashed')
ON CONFLICT (email) DO NOTHING;
