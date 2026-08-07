/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package documents implements Digital Documents & E-Signatures Go module (io.example.documents).
 */

package documents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/appregistry"
	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/tenant"
)

// DocTypes enumerates the accepted document classifications. The column has no
// CHECK constraint, so validation happens here.
var DocTypes = []string{"CONTRACT", "REQUEST", "APPROVAL"}

type Document struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Title     string    `json:"title"`
	DocType   string    `json:"doc_type"` // CONTRACT, REQUEST, APPROVAL
	Status    string    `json:"status"`   // DRAFT, PENDING_APPROVAL, APPROVED, REJECTED
	SignedBy  string    `json:"signed_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DocumentsModule struct {
	db *pgxpool.Pool
}

// New builds the module and registers it in the compile-time app registry so
// the app store can resolve and install io.example.documents.
func New(db *pgxpool.Pool) *DocumentsModule {
	m := &DocumentsModule{db: db}
	appregistry.Register(m)
	return m
}

func (m *DocumentsModule) ID() string      { return "io.example.documents" }
func (m *DocumentsModule) Name() string    { return "Digital Documents & Signatures" }
func (m *DocumentsModule) Version() string { return "1.0.0" }

func (m *DocumentsModule) Dependencies() []internal.Dependency { return nil }

func (m *DocumentsModule) Permissions() []internal.PermissionDefinition {
	return []internal.PermissionDefinition{
		{Code: "documents.read", Name: "Read Documents", Description: "View documents and signature status"},
		{Code: "documents.manage", Name: "Manage Documents", Description: "Create documents and route them for approval"},
	}
}

func (m *DocumentsModule) Menus() []internal.MenuDefinition {
	return []internal.MenuDefinition{
		{ID: "documents", ParentID: "operations", Label: "Documents & E-Sign", Path: "/documents", Icon: "file-text", Order: 30, Labels: map[string]string{"mn": "Баримт ба цахим гарын үсэг"}},
	}
}

func (m *DocumentsModule) RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/documents", func(dr chi.Router) {
		dr.Use(tenantAuthMiddleware)
		dr.Get("/", m.listDocumentsHandler)
		dr.Post("/", m.createDocumentHandler)
	})
}

func (m *DocumentsModule) listDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	list, err := m.ListDocuments(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch documents")
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (m *DocumentsModule) createDocumentHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Title   string `json:"title"`
		DocType string `json:"doc_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeError(w, http.StatusBadRequest, "invalid document parameters: title is required")
		return
	}

	doc, err := m.CreateDocument(r.Context(), tenantID, req.Title, req.DocType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, doc)
}

func (m *DocumentsModule) CreateDocument(ctx context.Context, tenantID, title, docType string) (*Document, error) {
	if title == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}
	if docType == "" {
		docType = "CONTRACT"
	}
	if !slices.Contains(DocTypes, docType) {
		return nil, fmt.Errorf("unsupported doc_type %q", docType)
	}

	var doc Document
	const query = `
		INSERT INTO document_records (tenant_id, title, doc_type, status)
		VALUES ($1, $2, $3, 'PENDING_APPROVAL')
		RETURNING id, tenant_id, title, doc_type, status, created_at
	`
	// The old code answered a failed INSERT with a canned "doc_demo_200"
	// record, reporting success for a document that was never stored.
	err := m.db.QueryRow(ctx, query, tenantID, title, docType).Scan(
		&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status, &doc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return &doc, nil
}

func (m *DocumentsModule) ListDocuments(ctx context.Context, tenantID string) ([]Document, error) {
	const query = `SELECT id, tenant_id, title, doc_type, status, COALESCE(signed_by, ''), created_at
	                 FROM document_records WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := m.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	list := make([]Document, 0)
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.DocType, &doc.Status, &doc.SignedBy, &doc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		list = append(list, doc)
	}
	return list, rows.Err()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
