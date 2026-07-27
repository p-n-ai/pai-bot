// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTeacherResourceTenantAndClassIsolationIntegration(t *testing.T) {
	databaseURL := os.Getenv("LEARN_RETRIEVAL_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LEARN_RETRIEVAL_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var tenantA, tenantB, uploaderA, classA, classAOther, classB string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug) VALUES ('Retrieval A', 'retrieval-integration-a-' || gen_random_uuid()) RETURNING id::text`,
	).Scan(&tenantA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug) VALUES ('Retrieval B', 'retrieval-integration-b-' || gen_random_uuid()) RETURNING id::text`,
	).Scan(&tenantB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = ANY($1::uuid[])`, []string{tenantA, tenantB})
	})
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, external_id, channel)
		VALUES ($1::uuid, 'teacher', 'Teacher A', gen_random_uuid()::text, 'web')
		RETURNING id::text`, tenantA).Scan(&uploaderA); err != nil {
		t.Fatal(err)
	}
	classA = seedRetrievalClass(t, ctx, pool, tenantA, "Class A")
	classAOther = seedRetrievalClass(t, ctx, pool, tenantA, "Class A Other")
	classB = seedRetrievalClass(t, ctx, pool, tenantB, "Class B")

	service, err := NewTeacherResourceService(pool, TeacherResourceOptions{AllowGraphFallback: true})
	if err != nil {
		t.Fatal(err)
	}
	resource, err := service.Upload(ctx, TeacherUploadInput{
		TenantID: tenantA, UploaderID: uploaderA, Filename: "algebra.docx",
		Title: "Algebra lesson", MediaType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		ClassIDs: []string{classA},
		Data: officeArchive(t, map[string]string{
			"word/document.xml": `<w:document xmlns:w="w"><w:body><w:p><w:r><w:t>Factorisation uses common factors.</w:t></w:r></w:p></w:body></w:document>`,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resource.ChunkCount != 1 {
		t.Fatalf("upload chunk_count = %d, want 1", resource.ChunkCount)
	}

	visible, err := service.List(ctx, tenantA, []string{classA}, false)
	if err != nil || len(visible) != 1 || visible[0].ID != resource.ID || visible[0].ChunkCount != 1 {
		t.Fatalf("class A list = %#v, %v", visible, err)
	}
	hidden, err := service.List(ctx, tenantA, []string{classAOther}, false)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("other class list = %#v, %v", hidden, err)
	}
	if _, err := service.List(ctx, tenantA, []string{classB}, false); !errors.Is(err, ErrTeacherResourceScope) {
		t.Fatalf("cross-tenant class list error = %v", err)
	}
	if err := service.SetActive(ctx, tenantA, resource.ID, []string{classAOther}, false); !errors.Is(err, ErrTeacherResourceNotFound) {
		t.Fatalf("other class mutation error = %v", err)
	}
	if err := service.SetActive(ctx, tenantA, resource.ID, []string{classA}, false); err != nil {
		t.Fatalf("deactivate error = %v", err)
	}
	if err := service.SetActive(ctx, tenantA, resource.ID, []string{classA}, true); err != nil {
		t.Fatalf("reactivate error = %v", err)
	}
	evidence, err := service.Search(ctx, TeacherEvidenceRequest{
		TenantID: tenantA, ClassIDs: []string{classA}, Query: "factorisation", Limit: 5,
	})
	if err != nil || len(evidence) != 1 || evidence[0].SourceTitle != "Algebra lesson" {
		t.Fatalf("evidence = %#v, %v", evidence, err)
	}
	hiddenEvidence, err := service.Search(ctx, TeacherEvidenceRequest{
		TenantID: tenantA, ClassIDs: []string{classAOther}, Query: "factorisation", Limit: 5,
	})
	if err != nil || len(hiddenEvidence) != 0 {
		t.Fatalf("other class evidence = %#v, %v", hiddenEvidence, err)
	}
}

func TestTeacherResourcePgGraphExpansionIntegration(t *testing.T) {
	databaseURL := os.Getenv("LEARN_RETRIEVAL_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("LEARN_RETRIEVAL_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var tenantID, uploaderID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug)
		VALUES ('Graph Retrieval', 'graph-retrieval-' || gen_random_uuid())
		RETURNING id::text`).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1::uuid`, tenantID)
	})
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, external_id, channel)
		VALUES ($1::uuid, 'teacher', 'Graph Teacher', gen_random_uuid()::text, 'web')
		RETURNING id::text`, tenantID).Scan(&uploaderID); err != nil {
		t.Fatal(err)
	}
	classID := seedRetrievalClass(t, ctx, pool, tenantID, "Graph Class")
	service, err := NewTeacherResourceService(pool, TeacherResourceOptions{
		AllowGraphFallback: false,
		GraphDepth:         1,
		GraphFrontier:      20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyGraph(ctx); err != nil {
		t.Fatalf("VerifyGraph() error = %v", err)
	}

	words := []string{"graphseed"}
	for range 135 {
		words = append(words, "mathematics")
	}
	words = append(words, "neighborproof")
	resource, err := service.Upload(ctx, TeacherUploadInput{
		TenantID: tenantID, UploaderID: uploaderID, Filename: "graph.docx",
		Title: "Graph expansion lesson", ClassIDs: []string{classID},
		Data: officeArchive(t, map[string]string{
			"word/document.xml": `<w:document xmlns:w="w"><w:body><w:p><w:r><w:t>` +
				strings.Join(words, " ") + `</w:t></w:r></w:p></w:body></w:document>`,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resource.ChunkCount < 2 {
		t.Fatalf("upload chunk_count = %d, want multiple chunks", resource.ChunkCount)
	}

	var lexicalNeighborCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM teacher_resource_chunks
		WHERE resource_id = $1::uuid
		  AND content LIKE '%neighborproof%'
		  AND search_vector @@ websearch_to_tsquery('simple', 'graphseed')`,
		resource.ID).Scan(&lexicalNeighborCount); err != nil {
		t.Fatal(err)
	}
	if lexicalNeighborCount != 0 {
		t.Fatalf("neighbor unexpectedly matched lexical seed: %d", lexicalNeighborCount)
	}

	evidence, err := service.Search(ctx, TeacherEvidenceRequest{
		TenantID: tenantID, ClassIDs: []string{classID}, Query: "graphseed", Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	for _, item := range evidence {
		if strings.Contains(item.Excerpt, "neighborproof") {
			return
		}
	}
	t.Fatalf("graph-expanded neighbor missing from evidence: %#v", evidence)
}

func seedRetrievalClass(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO groups (tenant_id, name, type, join_code)
		VALUES ($1::uuid, $2, 'class', gen_random_uuid()::text)
		RETURNING id::text`, tenantID, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
