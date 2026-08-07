/*
 * Gerege Open ERP
 * Copyright (c) 2026 Gerege Systems Development Team & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * eID Mongolia qualified remote signing. The ceremony is asynchronous because
 * the signing key never leaves the citizen's phone:
 *
 *   POST /esign/sign/init      upload, hash, push the PIN2 prompt, return a
 *                              verification code
 *   GET  /esign/sign/{id}      poll until completed / rejected / expired
 *   GET  /esign/sign/{id}/download   stream the PAdES-signed PDF
 *
 * The signed document is assembled by eID's own doc-signer (the stamp
 * endpoint), which embeds the PKCS#7 together with OCSP and CRL data. This
 * service never holds a signing key.
 */

package esign

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/audit"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/eidsign"
)

// sessionIDPattern is the identifier shape the browser validates before it
// starts polling. Keeping the server's generator and the client's guard in
// agreement is what stops a mistyped URL turning into a database round trip.
var sessionIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// newSessionID returns a 32-character lowercase hex identifier.
func newSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// signInitHandler starts a ceremony. It accepts either a multipart upload — a
// file signed directly, which is the flow the signing view uses — or a JSON
// body naming a document already in the store.
func (m *Module) signInitHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := m.require(w, r, PermSign)
	if !ok {
		return
	}
	if !m.eid.Enabled() {
		writeDomainError(w, &Error{
			Code:    "EID_NOT_CONFIGURED",
			Message: "eID Mongolia signing is not configured on this deployment",
			Status:  http.StatusServiceUnavailable,
		})
		return
	}

	_, policy, _, err := m.store.loadSettings(r.Context(), tenantID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	pdf, fileName, documentID, onBehalfOf, err := m.readSignInput(r, tenantID, policy)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if onBehalfOf != "" && !policy.AllowOnBehalfOf {
		writeDomainError(w, forbidden("this tenant does not allow signing on behalf of an organisation"))
		return
	}

	// Who signs. A linked eID account needs no input; anything else must name
	// the citizen explicitly, and a typo there would push the PIN2 prompt at
	// somebody else's phone, so the value is validated rather than trusted.
	signerEtsi := actor.Etsi
	if raw := strings.TrimSpace(r.FormValue("signer_id")); raw != "" {
		signerEtsi = eidsign.PersonEtsiFor(raw)
	}
	if signerEtsi == "" {
		writeDomainError(w, badRequest("NO_SIGNER_IDENTITY",
			"this account is not linked to eID Mongolia; sign in with eID or supply signer_id"))
		return
	}
	if err := validateEtsi(signerEtsi); err != nil {
		writeDomainError(w, err)
		return
	}

	sessionID, err := newSessionID()
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// The digest is taken over exactly the bytes stored on the session. That
	// pairing is the whole integrity claim: what the citizen approves on their
	// phone is what the stamp endpoint later signs.
	digest := eidsign.DigestOf(pdf)

	started, err := m.eid.Sign(r.Context(), eidsign.SignRequest{
		PersonEtsi:     signerEtsi,
		DocumentNumber: actor.DocumentNumber,
		Digest:         digest,
		DisplayText:    eidsign.DisplayText(fileName),
		FileName:       fileName,
		OnBehalfOf:     onBehalfOf,
	})
	if err != nil {
		m.log(r, logEntry{
			TenantID: tenantID, DocumentID: documentID, Provider: ProviderEID,
			Action: ActionSignStart, Outcome: OutcomeFailed, RegNo: signerEtsi,
			ActorUserID: actor.UserID, Detail: err.Error(),
		})
		writeDomainError(w, translateEIDError(err))
		return
	}

	session, err := m.store.createSession(r.Context(), newSession{
		ID:               sessionID,
		TenantID:         tenantID,
		DocumentID:       documentID,
		EIDSessionID:     started.SessionID,
		FileName:         fileName,
		DocumentHash:     hexDigest(pdf),
		VerificationCode: started.VerificationCode,
		SignerUserID:     actor.UserID,
		SignerEtsi:       signerEtsi,
		SignerName:       actor.FullName,
		OnBehalfOfEtsi:   onBehalfOf,
		OriginalPDF:      pdf,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	m.log(r, logEntry{
		TenantID: tenantID, DocumentID: documentID, SessionID: sessionID,
		Provider: ProviderEID, Action: ActionSignStart, Outcome: OutcomeOK,
		RegNo: signerEtsi, ActorUserID: actor.UserID,
	})
	audit.Record(r.Context(), tenantID, actor.UserID, "esign.sign_started", "esign", map[string]any{
		"session_id": sessionID, "document_id": documentID, "on_behalf_of": onBehalfOf,
	})

	writeJSON(w, http.StatusOK, session)
}

// readSignInput accepts either shape of request and returns the bytes to sign.
func (m *Module) readSignInput(r *http.Request, tenantID string, policy Policy) (pdf []byte, fileName, documentID, onBehalfOf string, err error) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(nil, r.Body, maxUploadBody)
		if parseErr := r.ParseMultipartForm(8 << 20); parseErr != nil {
			return nil, "", "", "", &Error{
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "the upload exceeds the " + strconv.Itoa(policy.MaxUploadMB) + "MB limit",
				Status:  http.StatusRequestEntityTooLarge,
			}
		}
		file, header, formErr := r.FormFile("file")
		if formErr != nil {
			return nil, "", "", "", badRequest("MISSING_FILE", "a PDF file is required in the 'file' field")
		}
		defer func() { _ = file.Close() }()

		pdf, err = io.ReadAll(io.LimitReader(file, int64(policy.MaxUploadMB<<20)+1))
		if err != nil {
			return nil, "", "", "", badRequest("UNREADABLE_FILE", "the uploaded file could not be read")
		}
		if len(pdf) > policy.MaxUploadMB<<20 {
			return nil, "", "", "", &Error{
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "the PDF exceeds the " + strconv.Itoa(policy.MaxUploadMB) + "MB limit",
				Status:  http.StatusRequestEntityTooLarge,
			}
		}
		if err = validatePDF(pdf); err != nil {
			return nil, "", "", "", err
		}
		fileName = sanitizeFileName(header.Filename)
		onBehalfOf = normalizeOrgEtsi(r.FormValue("onBehalfOf"))

		// A multipart ceremony may still attach to a stored document, which is
		// how batch signing reuses this path.
		if id := strings.TrimSpace(r.FormValue("document_id")); id != "" {
			doc, docErr := m.store.getDocument(r.Context(), tenantID, id)
			if docErr != nil {
				return nil, "", "", "", docErr
			}
			documentID = doc.ID
		}
		return pdf, fileName, documentID, onBehalfOf, nil
	}

	var req struct {
		DocumentID string `json:"document_id"`
		OnBehalfOf string `json:"on_behalf_of"`
	}
	if err = decodeJSON(r, &req); err != nil {
		return nil, "", "", "", err
	}
	if strings.TrimSpace(req.DocumentID) == "" {
		return nil, "", "", "", badRequest("MISSING_DOCUMENT",
			"upload a file or supply document_id")
	}

	pdf, _, title, err := m.store.documentForSigning(r.Context(), tenantID, req.DocumentID)
	if err != nil {
		return nil, "", "", "", err
	}
	doc, err := m.store.getDocument(r.Context(), tenantID, req.DocumentID)
	if err != nil {
		return nil, "", "", "", err
	}
	fileName = doc.FileName
	if fileName == "" {
		fileName = sanitizeFileName(title)
	}
	return pdf, fileName, doc.ID, normalizeOrgEtsi(req.OnBehalfOf), nil
}

// signStatusHandler is what the browser polls. It reconciles our session with
// eID's on every call: eID is authoritative, and this row is a cache of it.
func (m *Module) signStatusHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := m.require(w, r, PermSign)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !sessionIDPattern.MatchString(id) {
		writeDomainError(w, badRequest("INVALID_SESSION_ID", "the signing session id is malformed"))
		return
	}

	session, err := m.store.getSession(r.Context(), tenantID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	// A terminal session is a settled fact; re-asking eID about it would only
	// add latency to a poll that has already finished.
	if session.State != SessionPending {
		writeJSON(w, http.StatusOK, session)
		return
	}

	settled, err := m.reconcile(r, tenantID, actor, session)
	if err != nil {
		// A transient upstream failure must not fail the poll — the browser
		// treats a non-answer as "keep waiting", which is the correct
		// behaviour for a ceremony that is still open.
		slog.Warn("esign: could not reconcile signing session", "session_id", id, "error", err)
		writeJSON(w, http.StatusOK, session)
		return
	}
	writeJSON(w, http.StatusOK, settled)
}

// reconcile long-polls eID and, on success, stamps and stores the signed PDF.
func (m *Module) reconcile(r *http.Request, tenantID string, actor Actor, session *SignSession) (*SignSession, error) {
	eidSessionID, pdf, err := m.store.sessionUpstream(r.Context(), tenantID, session.ID)
	if err != nil {
		return nil, err
	}
	if eidSessionID == "" {
		return session, nil
	}

	// The long poll is bounded well inside the API's write deadline. A request
	// that outlives that deadline is closed with nothing written, and the
	// citizen sees the proxy's 502 page on every check.
	ctx, cancel := context.WithTimeout(r.Context(), pollWindow+5*time.Second)
	defer cancel()

	result, err := m.eid.Session(ctx, eidSessionID, pollWindow)
	if err != nil {
		if errors.Is(err, eidsign.ErrSessionNotFound) {
			_ = m.store.failSession(r.Context(), tenantID, session.ID, SessionExpired, "upstream_session_gone")
			return m.store.getSession(r.Context(), tenantID, session.ID)
		}
		return nil, err
	}

	switch result.State {
	case eidsign.StateRunning:
		return session, nil

	case eidsign.StateRefused, eidsign.StateExpired, eidsign.StateFailed:
		state := map[string]string{
			eidsign.StateRefused: SessionRejected,
			eidsign.StateExpired: SessionExpired,
			eidsign.StateFailed:  SessionFailed,
		}[result.State]
		reason := strings.ToLower(result.EndResult)
		if err := m.store.failSession(r.Context(), tenantID, session.ID, state, reason); err != nil {
			return nil, err
		}
		m.log(r, logEntry{
			TenantID: tenantID, DocumentID: session.DocumentID, SessionID: session.ID,
			Provider: ProviderEID, Action: ActionSign,
			Outcome:     map[string]string{SessionRejected: OutcomeRejected, SessionExpired: OutcomeExpired, SessionFailed: OutcomeFailed}[state],
			RegNo:       session.SignerEtsi,
			ActorUserID: actor.UserID, Detail: result.EndResult,
		})
		return m.store.getSession(r.Context(), tenantID, session.ID)
	}

	// COMPLETE. Hand eID back the exact bytes whose digest was approved and
	// let its doc-signer produce the PAdES document.
	signed, err := m.eid.Stamp(r.Context(), eidSessionID, session.FileName, pdf)
	if err != nil {
		return nil, err
	}

	won, err := m.store.completeSession(r.Context(), tenantID, session.ID, sessionCompletion{
		SignedPDF:          signed,
		CertificateLevel:   result.CertificateLevel,
		SignatureAlgorithm: result.SignatureAlgorithm,
		OnBehalfOfEtsi:     result.OnBehalfOfEtsi,
		OnBehalfOfName:     result.OnBehalfOfName,
	})
	if err != nil {
		return nil, err
	}
	// Two pollers can race here. Only the one that flipped the row writes the
	// document and the log, so a signature is never recorded twice.
	if won && session.DocumentID != "" {
		if err := m.store.markSigned(r.Context(), tenantID, session.DocumentID, signedDocument{
			Provider:         ProviderEID,
			SignedPDF:        signed,
			SignerName:       session.SignerName,
			SignerRegNo:      civilIDFromEtsi(session.SignerEtsi),
			SignerEtsi:       session.SignerEtsi,
			OnBehalfOfEtsi:   result.OnBehalfOfEtsi,
			OnBehalfOfName:   result.OnBehalfOfName,
			CertificateLevel: result.CertificateLevel,
			SignedAt:         time.Now(),
		}); err != nil {
			return nil, err
		}
	}
	if won {
		m.log(r, logEntry{
			TenantID: tenantID, DocumentID: session.DocumentID, SessionID: session.ID,
			Provider: ProviderEID, Action: ActionSign, Outcome: OutcomeOK,
			RegNo: session.SignerEtsi, FirstName: session.SignerName,
			ActorUserID: actor.UserID, Detail: result.CertificateLevel,
		})
		audit.Record(r.Context(), tenantID, actor.UserID, "esign.document_signed", "esign", map[string]any{
			"session_id": session.ID, "document_id": session.DocumentID,
			"provider": ProviderEID, "certificate_level": result.CertificateLevel,
		})
	}
	return m.store.getSession(r.Context(), tenantID, session.ID)
}

func (m *Module) signDownloadHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := m.require(w, r, PermSign)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !sessionIDPattern.MatchString(id) {
		writeDomainError(w, badRequest("INVALID_SESSION_ID", "the signing session id is malformed"))
		return
	}

	pdf, fileName, err := m.store.sessionSignedPDF(r.Context(), tenantID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	m.log(r, logEntry{
		TenantID: tenantID, SessionID: id, Provider: ProviderEID,
		Action: ActionDownload, Outcome: OutcomeOK, ActorUserID: actor.UserID,
	})
	writePDF(w, signedName(fileName), pdf)
}

// signCancelHandler abandons a ceremony from this side. eID's own session is
// left to expire: there is no cancel on the RP API, and the citizen's phone
// simply stops mattering once we refuse the result.
func (m *Module) signCancelHandler(w http.ResponseWriter, r *http.Request) {
	tenantID, actor, ok := m.require(w, r, PermSign)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !sessionIDPattern.MatchString(id) {
		writeDomainError(w, badRequest("INVALID_SESSION_ID", "the signing session id is malformed"))
		return
	}
	if err := m.store.failSession(r.Context(), tenantID, id, SessionFailed, "cancelled_by_user"); err != nil {
		writeDomainError(w, err)
		return
	}
	m.log(r, logEntry{
		TenantID: tenantID, SessionID: id, Provider: ProviderEID,
		Action: ActionSign, Outcome: OutcomeCancelled, ActorUserID: actor.UserID,
	})
	session, err := m.store.getSession(r.Context(), tenantID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

// organizationsHandler lists the organisations the signer may act for, so the
// signing view can offer "sign on behalf of". Representation rights are read
// live from the national registry rather than from a certificate, because a
// director who resigned yesterday still holds yesterday's certificate.
func (m *Module) organizationsHandler(w http.ResponseWriter, r *http.Request) {
	_, actor, ok := m.require(w, r, PermSign)
	if !ok {
		return
	}
	if actor.Etsi == "" {
		// Not an error: an unlinked account simply has nothing to represent.
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	orgs, err := m.eid.Representations(r.Context(), actor.Etsi)
	if err != nil {
		// The dropdown is an enhancement; failing it would block signing
		// entirely for a permission the tenant may not even hold.
		slog.Warn("esign: could not list representations", "error", err)
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, orgs)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// translateEIDError converts an upstream failure into something a citizen can
// act on, without leaking the upstream body — which can carry identifiers.
func translateEIDError(err error) error {
	switch {
	case errors.Is(err, eidsign.ErrNotEnrolled):
		return &Error{
			Code:    "SIGNER_NOT_ENROLLED",
			Message: "this citizen is not enrolled for signing in eID Mongolia",
			Status:  http.StatusBadRequest,
		}
	case errors.Is(err, eidsign.ErrNotRepresentative):
		return forbidden("you are not registered as a representative of this organisation")
	case errors.Is(err, eidsign.ErrRPRejected):
		// An operator problem. Saying "try again" would send the citizen in
		// circles, so it is named as a configuration fault.
		return &Error{
			Code:    "EID_RP_REJECTED",
			Message: "this deployment is not authorised to sign with eID Mongolia; contact your administrator",
			Status:  http.StatusServiceUnavailable,
		}
	}
	return upstream("EID_UNAVAILABLE", "eID Mongolia could not start the signature; please try again")
}

// validateEtsi guards the identifier before it is put in a URL path.
var etsiPattern = regexp.MustCompile(`^(PNOMN|NTRMN)-[A-Za-z0-9]{1,32}$`)

func validateEtsi(etsi string) error {
	if !etsiPattern.MatchString(etsi) {
		return badRequest("INVALID_SIGNER", "the signer identifier is not a valid registration or civil ID")
	}
	return nil
}

func normalizeOrgEtsi(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return eidsign.OrgEtsiFor(raw)
}

// civilIDFromEtsi recovers the bare identifier for display and for the
// signature log, which predates ETSI identifiers.
func civilIDFromEtsi(etsi string) string {
	if idx := strings.Index(etsi, "-"); idx >= 0 {
		return etsi[idx+1:]
	}
	return etsi
}

// hexDigest is the human-readable form of the same SHA-256 the citizen's
// device signs. It is shown in the signing view and stored on the session so a
// document can be matched to its ceremony after the fact.
func hexDigest(pdf []byte) string {
	sum := sha256.Sum256(pdf)
	return hex.EncodeToString(sum[:])
}

// log records an auditable event. Failures are logged and swallowed: losing an
// audit row must never fail a signature that has already been made, because
// the signed document is the record of consequence.
func (m *Module) log(r *http.Request, entry logEntry) {
	if err := m.store.recordLog(r.Context(), entry); err != nil {
		slog.Error("esign: could not write the signature log",
			"action", entry.Action, "outcome", entry.Outcome, "error", err)
	}
}
