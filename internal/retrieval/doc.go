// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package retrieval provides a generic retrieval platform for the bot.
//
// Step by step:
//  1. A Source describes where knowledge came from.
//     Examples: curriculum, website, PDF, book, seminar, YouTube.
//  2. A Collection groups related material into a search scope.
//     Examples: curriculum:form-2-math, teacher-notes, seminar-transcripts.
//  3. A Document is the unit callers create, activate, list, and fetch.
//     Today the stored document is also the indexed chunk.
//  4. Search runs BM25 over the indexed document fields plus metadata filters.
//  5. Agent code consumes the same retrieval service instead of owning its own
//     separate search system.
//
// Public service surface:
//   - CreateSource / UpdateSource / GetSource / ListSources / DeleteSource / SetSourceActive
//   - CreateCollection / UpsertCollection / UpdateCollection / GetCollection / ListCollections / DeleteCollection / SetCollectionActive
//   - CreateDocument / UpsertDocument / UpdateDocument / GetDocument / ListDocuments / ListDocument / DeleteDocument / SetDocumentActive
//   - Search
//
// Public HTTP surface:
//   - GET|POST /api/admin/retrieval/sources
//   - GET|PUT|DELETE /api/admin/retrieval/sources/{id}
//   - POST /api/admin/retrieval/sources/{id}/activate
//   - GET|POST /api/admin/retrieval/collections
//   - GET|PUT|DELETE /api/admin/retrieval/collections/{id}
//   - POST /api/admin/retrieval/collections/{id}/activate
//   - GET|POST /api/admin/retrieval/documents
//   - GET|PUT|DELETE /api/admin/retrieval/documents/{id}
//   - POST /api/admin/retrieval/documents/{id}/activate
//   - POST /api/admin/retrieval/search
//
// Internal flow:
//   - curriculum_seed.go normalizes OSS curriculum into Source + Collection + Document records
//   - service.go owns in-memory storage, BM25 indexing, filtering, and scoring
//   - context_resolver.go and curriculum_retriever.go call the shared retrieval service
//     instead of maintaining a separate private search stack
package retrieval
