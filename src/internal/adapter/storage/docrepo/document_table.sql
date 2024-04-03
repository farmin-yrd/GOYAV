CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    document_id VARCHAR(255) NOT NULL UNIQUE,
    hash VARCHAR(255) NOT NULL,
    tag VARCHAR(255) NOT NULL,
    status INTEGER NOT NULL,
    analyzed_at TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_document_id ON documents(document_id);
CREATE INDEX IF NOT EXISTS idx_hash ON documents(hash);
CREATE INDEX IF NOT EXISTS idx_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_analyzed_at ON documents(analyzed_at);

-- Check Constraints 
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'chk_status' AND conrelid = 'documents'::regclass
    ) THEN
        ALTER TABLE documents ADD CONSTRAINT chk_status CHECK (status IN (0, 1, 2));
    END IF;
END
$$;