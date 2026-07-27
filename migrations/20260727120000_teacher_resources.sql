-- +goose Up
-- +goose NO TRANSACTION
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS graph;

-- +goose StatementBegin
DO $$
DECLARE
    graph_version TEXT;
BEGIN
    SELECT extversion INTO graph_version FROM pg_extension WHERE extname = 'graph';
    IF graph_version <> '1.0.0' THEN
        RAISE EXCEPTION 'teacher retrieval requires graph extension 1.0.0, found %', COALESCE(graph_version, 'missing');
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE teacher_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    uploader_id UUID NOT NULL REFERENCES users(id),
    filename TEXT NOT NULL,
    title TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('pdf', 'docx', 'pptx')),
    media_type TEXT NOT NULL,
    byte_size INTEGER NOT NULL CHECK (byte_size > 0 AND byte_size <= 20971520),
    sha256 BYTEA NOT NULL CHECK (octet_length(sha256) = 32),
    original_file BYTEA NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, sha256),
    UNIQUE (id, tenant_id)
);

CREATE TABLE teacher_resource_classes (
    resource_id UUID NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    class_id UUID NOT NULL REFERENCES groups(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (resource_id, class_id),
    FOREIGN KEY (resource_id, tenant_id) REFERENCES teacher_resources(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE teacher_resource_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    ordinal INTEGER NOT NULL CHECK (ordinal >= 0),
    locator_type TEXT NOT NULL CHECK (locator_type IN ('page', 'slide', 'section')),
    locator_start INTEGER NOT NULL CHECK (locator_start > 0),
    locator_end INTEGER NOT NULL CHECK (locator_end >= locator_start),
    content TEXT NOT NULL CHECK (length(btrim(content)) > 0),
    search_vector TSVECTOR GENERATED ALWAYS AS
        (setweight(to_tsvector('simple', content), 'A')) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resource_id, ordinal),
    UNIQUE (id, tenant_id),
    FOREIGN KEY (resource_id, tenant_id) REFERENCES teacher_resources(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE teacher_resource_embeddings (
    chunk_id UUID PRIMARY KEY,
    resource_id UUID NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    model TEXT NOT NULL,
    dimensions INTEGER NOT NULL CHECK (dimensions = 1536),
    embedding VECTOR(1536) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (chunk_id, tenant_id) REFERENCES teacher_resource_chunks(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (resource_id, tenant_id) REFERENCES teacher_resources(id, tenant_id) ON DELETE CASCADE
);

CREATE TABLE teacher_resource_chunk_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    resource_id UUID NOT NULL,
    source_chunk_id UUID NOT NULL,
    target_chunk_id UUID NOT NULL,
    edge_type TEXT NOT NULL CHECK (edge_type IN ('adjacent', 'related')),
    weight REAL NOT NULL DEFAULT 1 CHECK (weight > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (source_chunk_id <> target_chunk_id),
    UNIQUE (resource_id, source_chunk_id, target_chunk_id, edge_type),
    FOREIGN KEY (resource_id, tenant_id) REFERENCES teacher_resources(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (source_chunk_id, tenant_id) REFERENCES teacher_resource_chunks(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (target_chunk_id, tenant_id) REFERENCES teacher_resource_chunks(id, tenant_id) ON DELETE CASCADE
);

CREATE INDEX teacher_resources_tenant_active_idx ON teacher_resources (tenant_id, active, created_at DESC);
CREATE INDEX teacher_resource_classes_scope_idx ON teacher_resource_classes (tenant_id, class_id, resource_id);
CREATE INDEX teacher_resource_chunks_scope_idx ON teacher_resource_chunks (tenant_id, resource_id, ordinal);
CREATE INDEX teacher_resource_chunks_fts_idx ON teacher_resource_chunks USING GIN (search_vector);
CREATE INDEX teacher_resource_embeddings_scope_idx ON teacher_resource_embeddings (tenant_id, resource_id);
CREATE INDEX teacher_resource_embeddings_vector_idx ON teacher_resource_embeddings
    USING hnsw (embedding vector_cosine_ops);
CREATE INDEX teacher_resource_chunk_edges_scope_idx ON teacher_resource_chunk_edges
    (tenant_id, source_chunk_id, edge_type);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_teacher_resource_uploader_scope() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM users u
        WHERE u.id = NEW.uploader_id
          AND (u.tenant_id = NEW.tenant_id OR (u.role = 'platform_admin' AND u.tenant_id IS NULL))
    ) THEN
        RAISE EXCEPTION 'teacher resource uploader must belong to the resource tenant or be a platform admin';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER teacher_resource_uploader_scope
    BEFORE INSERT OR UPDATE ON teacher_resources
    FOR EACH ROW EXECUTE FUNCTION check_teacher_resource_uploader_scope();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_teacher_resource_class_scope() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM groups g
        WHERE g.id = NEW.class_id
          AND g.tenant_id = NEW.tenant_id
          AND g.type = 'class'
    ) THEN
        RAISE EXCEPTION 'teacher resource class must be a class in the resource tenant';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER teacher_resource_class_scope
    BEFORE INSERT OR UPDATE ON teacher_resource_classes
    FOR EACH ROW EXECUTE FUNCTION check_teacher_resource_class_scope();

SELECT graph.add_table(
    table_name := 'public.teacher_resource_chunks'::regclass,
    id_column := 'id',
    columns := ARRAY['content'],
    tenant_column := 'tenant_id'
);
SELECT graph.add_edge(
    from_table := 'public.teacher_resource_chunk_edges'::regclass,
    from_column := 'source_chunk_id',
    to_table := 'public.teacher_resource_chunks'::regclass,
    to_column := 'target_chunk_id',
    label := 'teacher_resource_chunk',
    bidirectional := true,
    weight_column := 'weight'
);
SELECT graph.add_filter_column('public.teacher_resource_chunks'::regclass, 'tenant_id', 'uuid');
SELECT * FROM graph.build();
SELECT graph.enable_sync();

-- +goose Down
SELECT graph.remove_edge('teacher_resource_chunk');
SELECT graph.remove_table('public.teacher_resource_chunks'::regclass);
SELECT * FROM graph.build();
DROP TRIGGER IF EXISTS teacher_resource_class_scope ON teacher_resource_classes;
DROP FUNCTION IF EXISTS check_teacher_resource_class_scope();
DROP TRIGGER IF EXISTS teacher_resource_uploader_scope ON teacher_resources;
DROP FUNCTION IF EXISTS check_teacher_resource_uploader_scope();
DROP TABLE IF EXISTS teacher_resource_chunk_edges;
DROP TABLE IF EXISTS teacher_resource_embeddings;
DROP TABLE IF EXISTS teacher_resource_chunks;
DROP TABLE IF EXISTS teacher_resource_classes;
DROP TABLE IF EXISTS teacher_resources;
