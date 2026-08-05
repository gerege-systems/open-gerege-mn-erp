-- +goose Up

-- Tenant-scoped AI configuration and retrieval corpus. Embeddings are stored as
-- JSON so the ERP remains compatible with plain PostgreSQL; keyword retrieval
-- continues to work when Gemini embeddings are unavailable.
CREATE TABLE IF NOT EXISTS ai_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    prompt_key VARCHAR(64) NOT NULL,
    content TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, prompt_key)
);

CREATE TABLE IF NOT EXISTS ai_knowledge (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    embedding JSONB,
    embedding_model VARCHAR(128) NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_tenant ON ai_knowledge(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_search ON ai_knowledge
USING GIN (to_tsvector('simple', title || ' ' || content));

INSERT INTO ai_prompts (tenant_id, prompt_key, content) VALUES
(NULL, 'scope', 'You are Gerege ERP AI Copilot. Answer only about ERP operations and the information returned by approved tools. Never invent database values or expose another tenant''s data.'),
(NULL, 'instructions', 'Be concise and practical. Reply in the requested language. Use tools whenever a question depends on live ERP data.')
ON CONFLICT (tenant_id, prompt_key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS ai_knowledge;
DROP TABLE IF EXISTS ai_prompts;
