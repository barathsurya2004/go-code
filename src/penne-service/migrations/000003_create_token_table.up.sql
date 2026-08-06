-- Define the types of tokens your system supports
CREATE TYPE token_purpose AS ENUM (
    'api_key', 
    'session', 
    'magic_link', 
    'password_reset'
);

CREATE TABLE user_tokens (
    user_id UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    token_UUID uuid PRIMARY KEY,
    prefix VARCHAR(50) NOT NULL,
    name VARCHAR(255),
    scopes TEXT[], 
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_tokens_uuid ON user_tokens(token_UUID);
CREATE INDEX idx_user_tokens_expires ON user_tokens(expires_at);