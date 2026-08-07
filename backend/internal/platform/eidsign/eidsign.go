/*
 * Gerege Open ERP
 * Copyright (c) 2026 Gerege Systems Development Team & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package eidsign speaks the eID Mongolia v3 Relying Party signature API —
 * qualified remote signing where the citizen approves with PIN2 on their
 * phone. The wire protocol is Smart-ID compatible.
 *
 * The contract is the one published at https://eidmongolia.mn/.well-known/eid
 * and documented in the platform's RP integration guide:
 *
 *	POST /v3/signature/notification/etsi/{semanticsIdentifier}  push by PNOMN-<civilId>
 *	POST /v3/signature/notification/document/{documentNumber}   push by device UUID
 *	GET  /v3/session/{sessionId}?timeoutMs=…                    long-poll
 *	POST /v3/signature/stamp/{sessionId}?fileName=…             PAdES assembly
 *
 * Note the split of responsibilities. The device signs a *digest*, never the
 * document: we hash the PDF, eID pushes that hash to the phone, and the
 * citizen's signing key (PIN2, non-repudiation) signs it. The signed PDF is
 * then assembled by eID's own doc-signer via the stamp endpoint, which embeds
 * the PKCS#7 together with OCSP/CRL revocation data. That is deliberate — a
 * relying party that assembled PAdES itself would have to reproduce eID's
 * timestamping and revocation policy to stay verifiable in Adobe Reader.
 */
package eidsign

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/config"
)

// Session states, normalised from the v3 wire protocol. eID reports only
// RUNNING and COMPLETE in `state`; the real outcome of a COMPLETE session sits
// in `result.endResult`, so terminal failures are folded into these.
const (
	StateRunning   = "RUNNING"
	StateComplete  = "COMPLETE"
	StateExpired   = "EXPIRED"
	StateRefused   = "REFUSED"
	StateFailed    = "FAILED"
	defaultBase    = "https://eidmongolia.mn/v3"
	defaultRPName  = "Gerege ERP"
	defaultLevel   = "QUALIFIED"
	protocolACSPv2 = "ACSP_V2"
	hashTypeSHA256 = "SHA256"

	// maxRespBytes caps every JSON response read. The stamp endpoint returns a
	// PDF and is read under its own, larger limit.
	maxRespBytes = 256 << 10
	// maxStampBytes caps the signed PDF eID returns. The verification page eID
	// appends grows the document, so this sits above the upload limit.
	maxStampBytes = 48 << 20
)

var (
	// ErrNotEnrolled is returned when eID has no signing certificate for the
	// citizen — they have not enrolled, or hold no PIN2 (signature) key.
	ErrNotEnrolled = errors.New("eidsign: citizen is not enrolled for signing in eID Mongolia")
	// ErrNotRepresentative is returned when signing on behalf of an
	// organisation the citizen is not an active representative of.
	ErrNotRepresentative = errors.New("eidsign: not authorized to sign for this organization")
	// ErrRPRejected is returned when eID refuses the relying party itself —
	// bad secret, unregistered UUID, missing SIGNATURE permission or an IP
	// outside the allowlist. It is an operator problem, never the citizen's.
	ErrRPRejected = errors.New("eidsign: relying party credentials rejected by eID")
	// ErrSessionNotFound is returned when a session id is unknown or has been
	// reaped by eID.
	ErrSessionNotFound = errors.New("eidsign: session not found")
)

// Interaction is the prompt the eID app renders on the confirmation screen.
// displayText60 is capped at 60 characters by the protocol.
type Interaction struct {
	Type          string `json:"type"`
	DisplayText60 string `json:"displayText60,omitempty"`
}

// SignRequest is one signing ceremony. Exactly one of PersonEtsi or
// DocumentNumber identifies who signs.
type SignRequest struct {
	// PersonEtsi is the ETSI EN 319 412-1 identifier of the signer,
	// PNOMN-<civilId>. Use PersonEtsiFor to build it.
	PersonEtsi string
	// DocumentNumber routes the push at one enrolled device instead of the
	// person. It is what a prior authentication session returned, so it is the
	// precise channel when the signer just logged in.
	DocumentNumber string
	// Digest is the base64 SHA-256 of the bytes being signed.
	Digest string
	// DisplayText is shown on the phone above the PIN2 prompt.
	DisplayText string
	// FileName is stored on the session and shown on eID's verification page.
	FileName string
	// OnBehalfOf is an organisation ETSI id (NTRMN-<register>) when the
	// citizen signs for a company. eID checks the representation right at
	// session creation and refuses with 403 when it is missing. The signature
	// itself is still made with the citizen's personal PIN2 certificate — this
	// is a delegation record, not a company seal.
	OnBehalfOf string
	// CallbackURL turns the ceremony into same-device App2App: the eID app
	// returns the browser here after approval. Leave empty for cross-device,
	// where the browser polls on its own.
	CallbackURL string
}

// StartResult is what a started ceremony hands back to the caller.
type StartResult struct {
	SessionID        string `json:"session_id"`
	VerificationCode string `json:"verification_code"`
}

// SessionResult is a normalised poll answer.
type SessionResult struct {
	State          string `json:"state"`
	EndResult      string `json:"end_result,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	// SignatureValue is the detached signature over the digest, base64.
	SignatureValue     string `json:"signature_value,omitempty"`
	SignatureAlgorithm string `json:"signature_algorithm,omitempty"`
	// CertificateDER is the signer's X.509 certificate, base64 DER.
	CertificateDER   string `json:"certificate_der,omitempty"`
	CertificateLevel string `json:"certificate_level,omitempty"`
	OnBehalfOfEtsi   string `json:"on_behalf_of_etsi,omitempty"`
	OnBehalfOfName   string `json:"on_behalf_of_name,omitempty"`
}

// Client is the eID Mongolia v3 signature surface this platform depends on.
// It is an interface so the esign app can be tested without a live RP.
type Client interface {
	// Sign starts a signing ceremony and pushes the PIN2 prompt to the phone.
	Sign(ctx context.Context, req SignRequest) (*StartResult, error)
	// Session long-polls a ceremony for up to timeout.
	Session(ctx context.Context, sessionID string, timeout time.Duration) (*SessionResult, error)
	// Stamp submits the original PDF against a completed session and returns
	// the PAdES-signed document eID assembles.
	Stamp(ctx context.Context, sessionID, fileName string, pdf []byte) ([]byte, error)
	// Representations lists the organisations a citizen may currently sign
	// for. Rights are read live from the registry rather than from a
	// certificate, because a director who resigned yesterday still holds
	// yesterday's certificate.
	Representations(ctx context.Context, personEtsi string) ([]Representation, error)
	// Enabled reports whether the client is configured to reach a live RP API.
	Enabled() bool
	// Mock reports whether canned results are being served.
	Mock() bool
}

type client struct {
	base      string
	rpUUID    string
	rpName    string
	secret    string
	certLevel string
	mock      bool
	http      *http.Client
	// mockStore backs the mock ceremony lifecycle.
	mockStore *mockSessions
}

// New builds the client from the environment. It reuses the EID_* relying
// party identity the authentication connector already uses — the same RP
// registration carries both permissions — and layers signature-specific
// overrides on top:
//
//	EID_SIGN_BASE_URL    RP API base, defaults to EID_BASE_URL then eidmongolia.mn/v3
//	EID_SIGN_CERT_LEVEL  minimum certificate level, defaults to QUALIFIED
//	EID_SIGN_DISPLAY_TEXT prompt shown above the PIN2 entry
//	EID_SIGN_MOCK_MODE   serve canned ceremonies instead of calling eID
//
// The certificate level defaults to QUALIFIED rather than authentication's
// ADVANCED on purpose: an advanced signature is not what "тоон гарын үсэг"
// means in law, and accepting one here would silently downgrade every
// document the ERP produces.
func New() Client {
	base := firstNonEmpty(os.Getenv("EID_SIGN_BASE_URL"), os.Getenv("EID_BASE_URL"), defaultBase)
	return &client{
		base:      strings.TrimRight(strings.TrimSpace(base), "/"),
		rpUUID:    strings.TrimSpace(os.Getenv("EID_RP_UUID")),
		rpName:    firstNonEmpty(os.Getenv("EID_RP_NAME"), defaultRPName),
		secret:    strings.TrimSpace(os.Getenv("EID_RP_SECRET")),
		certLevel: firstNonEmpty(os.Getenv("EID_SIGN_CERT_LEVEL"), defaultLevel),
		mock:      config.MockEnabled("EID_SIGN_MOCK_MODE"),
		// The poll is a long-poll: the HTTP deadline has to outlast the
		// timeoutMs we ask eID to hold the request open for.
		http:      &http.Client{Timeout: 150 * time.Second},
		mockStore: newMockSessions(),
	}
}

func (c *client) Enabled() bool { return c.mock || (c.rpUUID != "" && c.secret != "") }
func (c *client) Mock() bool    { return c.mock }

// DisplayText returns the configured PIN2 prompt, falling back to a document
// specific default.
func DisplayText(fileName string) string {
	if text := strings.TrimSpace(os.Getenv("EID_SIGN_DISPLAY_TEXT")); text != "" {
		return truncate60(text)
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		return "Баримтад гарын үсэг зурах"
	}
	return truncate60("Гарын үсэг: " + name)
}

// PersonEtsiFor builds the ETSI EN 319 412-1 semantics identifier eID keys
// citizens by. A bare civil ID becomes PNOMN-<civilId>; an identifier that
// already carries the PNO prefix is passed through upper-cased.
func PersonEtsiFor(id string) string {
	s := strings.TrimSpace(id)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToUpper(s), "PNO") {
		return strings.ToUpper(s)
	}
	return "PNOMN-" + s
}

// OrgEtsiFor builds the organisation identifier, NTRMN-<registrationNumber>.
func OrgEtsiFor(register string) string {
	s := strings.TrimSpace(register)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToUpper(s), "NTR") {
		return strings.ToUpper(s)
	}
	return "NTRMN-" + s
}

// DigestOf returns the base64 SHA-256 digest eID expects in `digest`.
func DigestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// signBody is the signature initiate payload. It differs from authentication
// in one field that matters: authentication carries `rpChallenge` (a base64
// nonce), signing carries `digest` + `hashType`. Sending the wrong one makes
// eID treat the challenge as empty and the ACSP payload breaks on the phone
// with an opaque "processing error".
type signBody struct {
	RelyingPartyUUID   string        `json:"relyingPartyUUID"`
	RelyingPartyName   string        `json:"relyingPartyName"`
	CertificateLevel   string        `json:"certificateLevel"`
	SignatureProtocol  string        `json:"signatureProtocol"`
	Digest             string        `json:"digest"`
	HashType           string        `json:"hashType"`
	Interactions       []Interaction `json:"interactions"`
	InitialCallbackURL string        `json:"initialCallbackUrl,omitempty"`
	OnBehalfOf         string        `json:"onBehalfOf,omitempty"`
	FileName           string        `json:"fileName,omitempty"`
}

func (c *client) Sign(ctx context.Context, req SignRequest) (*StartResult, error) {
	if strings.TrimSpace(req.Digest) == "" {
		return nil, errors.New("eidsign: digest is required")
	}
	if strings.TrimSpace(req.PersonEtsi) == "" && strings.TrimSpace(req.DocumentNumber) == "" {
		return nil, errors.New("eidsign: person_etsi or document_number is required")
	}
	if c.mock {
		return c.mockStore.start(req), nil
	}
	if c.rpUUID == "" || c.secret == "" {
		return nil, ErrRPRejected
	}

	body := signBody{
		RelyingPartyUUID:   c.rpUUID,
		RelyingPartyName:   c.rpName,
		CertificateLevel:   c.certLevel,
		SignatureProtocol:  protocolACSPv2,
		Digest:             req.Digest,
		HashType:           hashTypeSHA256,
		Interactions:       []Interaction{{Type: "displayTextAndPIN", DisplayText60: truncate60(req.DisplayText)}},
		InitialCallbackURL: strings.TrimSpace(req.CallbackURL),
		OnBehalfOf:         strings.TrimSpace(req.OnBehalfOf),
		FileName:           strings.TrimSpace(req.FileName),
	}

	path := "/signature/notification/document/" + url.PathEscape(strings.TrimSpace(req.DocumentNumber))
	if etsi := strings.TrimSpace(req.PersonEtsi); etsi != "" {
		path = "/signature/notification/etsi/" + url.PathEscape(etsi)
	}

	raw, status, err := c.post(ctx, path, body)
	if err != nil {
		return nil, err
	}
	if err := initiateError(raw, status); err != nil {
		return nil, err
	}

	var out struct {
		SessionID string          `json:"sessionID"`
		VC        json.RawMessage `json:"vc"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.SessionID == "" {
		return nil, fmt.Errorf("eidsign: initiate returned no sessionID: %s", snippet(raw))
	}
	return &StartResult{SessionID: out.SessionID, VerificationCode: parseVC(out.VC)}, nil
}

func (c *client) Session(ctx context.Context, sessionID string, timeout time.Duration) (*SessionResult, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("eidsign: session_id is required")
	}
	if c.mock {
		return c.mockStore.poll(sessionID), nil
	}
	// eID caps the long-poll at 120s.
	ms := int(timeout / time.Millisecond)
	if ms <= 0 || ms > 120000 {
		ms = 120000
	}

	raw, status, err := c.get(ctx, fmt.Sprintf("/session/%s?timeoutMs=%d", url.PathEscape(sessionID), ms))
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound || status == http.StatusGone {
		return nil, ErrSessionNotFound
	}
	if status >= 300 {
		return nil, fmt.Errorf("eidsign: session status %d: %s", status, snippet(raw))
	}

	var out struct {
		State  string `json:"state"`
		Result *struct {
			EndResult      string `json:"endResult"`
			DocumentNumber string `json:"documentNumber"`
		} `json:"result"`
		Signature *struct {
			Value              string `json:"value"`
			SignatureAlgorithm string `json:"signatureAlgorithm"`
		} `json:"signature"`
		Cert *struct {
			Value            string `json:"value"`
			CertificateLevel string `json:"certificateLevel"`
		} `json:"cert"`
		OnBehalfOf *struct {
			OrgEtsi string `json:"orgEtsi"`
			OrgName string `json:"orgName"`
		} `json:"onBehalfOf"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.State == "" {
		return nil, fmt.Errorf("eidsign: invalid session response: %s", snippet(raw))
	}

	if out.State != StateComplete {
		return &SessionResult{State: StateRunning}, nil
	}

	// COMPLETE only says the ceremony ended. Whether it succeeded is in
	// endResult, so a COMPLETE session with USER_REFUSED must not be mistaken
	// for a signature.
	endResult := ""
	if out.Result != nil {
		endResult = out.Result.EndResult
	}
	if endResult != "OK" {
		return &SessionResult{State: terminalState(endResult), EndResult: endResult}, nil
	}

	res := &SessionResult{State: StateComplete, EndResult: endResult}
	if out.Result != nil {
		res.DocumentNumber = out.Result.DocumentNumber
	}
	if out.Signature != nil {
		res.SignatureValue = out.Signature.Value
		res.SignatureAlgorithm = out.Signature.SignatureAlgorithm
	}
	if out.Cert != nil {
		res.CertificateDER = out.Cert.Value
		res.CertificateLevel = out.Cert.CertificateLevel
	}
	if out.OnBehalfOf != nil {
		res.OnBehalfOfEtsi = out.OnBehalfOf.OrgEtsi
		res.OnBehalfOfName = out.OnBehalfOf.OrgName
	}
	if res.SignatureValue == "" {
		return nil, fmt.Errorf("eidsign: COMPLETE+OK session carried no signature: %s", snippet(raw))
	}
	return res, nil
}

func (c *client) Stamp(ctx context.Context, sessionID, fileName string, pdf []byte) ([]byte, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("eidsign: session_id is required")
	}
	if len(pdf) == 0 {
		return nil, errors.New("eidsign: empty PDF payload")
	}
	if c.mock {
		return c.mockStore.stamp(sessionID, pdf)
	}

	endpoint := c.base + "/signature/stamp/" + url.PathEscape(sessionID) +
		"?fileName=" + url.QueryEscape(strings.TrimSpace(fileName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(pdf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/pdf")
	req.Header.Set("Accept", "application/pdf")
	c.authorize(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eidsign: stamp request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	signed, err := io.ReadAll(io.LimitReader(resp.Body, maxStampBytes))
	if err != nil {
		return nil, fmt.Errorf("eidsign: reading stamped PDF: %w", err)
	}
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, ErrRPRejected
		}
		return nil, fmt.Errorf("eidsign: stamp returned status %d: %s", resp.StatusCode, snippet(signed))
	}
	// A proxy error page would otherwise be stored as if it were the signed
	// document and only surface when a citizen opened the download.
	if len(signed) < 5 || string(signed[:5]) != "%PDF-" {
		return nil, fmt.Errorf("eidsign: stamp did not return a PDF: %s", snippet(signed))
	}
	return signed, nil
}

// Representation is an organisation the citizen may act for. The field names
// match what the signing view renders, so it is returned to the browser
// unchanged.
type Representation struct {
	OrgEtsi     string `json:"org_etsi"`
	OrgRegister string `json:"org_register"`
	OrgName     string `json:"org_name"`
	OrgNameEn   string `json:"org_name_en,omitempty"`
	Role        string `json:"role,omitempty"`
	RightType   string `json:"right_type,omitempty"`
	Source      string `json:"source,omitempty"`
}

func (c *client) Representations(ctx context.Context, personEtsi string) ([]Representation, error) {
	if strings.TrimSpace(personEtsi) == "" {
		return nil, errors.New("eidsign: person_etsi is required")
	}
	if c.mock {
		return []Representation{{
			OrgEtsi: "NTRMN-1234567", OrgRegister: "1234567",
			OrgName: "Demo Corporation", OrgNameEn: "Demo Corporation",
			RightType: "ADMIN", Source: "REGISTRY",
		}}, nil
	}

	raw, status, err := c.get(ctx, "/organization/representations/etsi/"+url.PathEscape(personEtsi))
	if err != nil {
		return nil, err
	}
	// Representation lookup needs a permission the relying party may not hold.
	// It drives an optional dropdown, so a refusal is an empty list rather
	// than an error that would block signing entirely.
	if status == http.StatusForbidden || status == http.StatusNotFound {
		return []Representation{}, nil
	}
	if status >= 300 {
		return nil, fmt.Errorf("eidsign: representations status %d: %s", status, snippet(raw))
	}

	var out struct {
		Representations []struct {
			OrgEtsi     string `json:"orgEtsi"`
			OrgRegister string `json:"orgRegister"`
			OrgName     string `json:"orgName"`
			OrgNameEn   string `json:"orgNameEn"`
			Role        string `json:"role"`
			RightType   string `json:"rightType"`
			Source      string `json:"source"`
		} `json:"representations"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("eidsign: invalid representations response: %s", snippet(raw))
	}

	list := make([]Representation, 0, len(out.Representations))
	for _, rep := range out.Representations {
		list = append(list, Representation{
			OrgEtsi: rep.OrgEtsi, OrgRegister: rep.OrgRegister, OrgName: rep.OrgName,
			OrgNameEn: rep.OrgNameEn, Role: rep.Role, RightType: rep.RightType, Source: rep.Source,
		})
	}
	return list, nil
}

// ─── HTTP plumbing ───────────────────────────────────────────────────────────

func (c *client) authorize(req *http.Request) {
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
}

func (c *client) post(ctx context.Context, path string, payload any) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.authorize(req)
	return c.do(req)
}

func (c *client) get(ctx context.Context, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	c.authorize(req)
	return c.do(req)
}

func (c *client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("eidsign: %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("eidsign: reading response: %w", err)
	}
	return raw, resp.StatusCode, nil
}

// initiateError maps an initiate failure onto a typed error so callers can
// tell an operator problem (RP rejected) from a citizen one (not enrolled)
// without parsing prose.
func initiateError(raw []byte, status int) error {
	if status < 300 {
		return nil
	}
	switch status {
	case http.StatusUnauthorized:
		return ErrRPRejected
	case http.StatusForbidden:
		// 403 covers both "RP may not sign" and "not a representative". The
		// body distinguishes them; representation is the far likelier cause
		// when onBehalfOf was sent.
		if bytes.Contains(bytes.ToLower(raw), []byte("represent")) {
			return ErrNotRepresentative
		}
		return ErrRPRejected
	case http.StatusNotFound:
		return ErrNotEnrolled
	}
	return fmt.Errorf("eidsign: initiate returned status %d: %s", status, snippet(raw))
}

// terminalState folds eID's endResult vocabulary into the states the UI knows.
func terminalState(endResult string) string {
	switch strings.ToUpper(strings.TrimSpace(endResult)) {
	case "TIMEOUT", "EXPIRED":
		return StateExpired
	case "":
		return StateFailed
	default:
		// USER_REFUSED, USER_REFUSED_VC_CHOICE, WRONG_VC, DOCUMENT_UNUSABLE …
		return StateRefused
	}
}

// parseVC reads the verification code, which the protocol sends as a bare
// string on device-link responses and as {"value":"1234"} on notifications.
func parseVC(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString
	}
	var asObject struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return asObject.Value
	}
	return ""
}

func truncate60(s string) string {
	s = strings.TrimSpace(s)
	// The cap is 60 characters, and Mongolian Cyrillic is two bytes each, so
	// slicing bytes would both overshoot the limit and split a rune.
	runes := []rune(s)
	if len(runes) <= 60 {
		return s
	}
	return string(runes[:60])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// snippet trims an upstream body for error messages. eID error bodies can
// carry the citizen's identifiers, so this stays short and is only ever put
// into server-side errors, never returned verbatim to a browser.
func snippet(raw []byte) string {
	const limit = 200
	s := strings.TrimSpace(string(raw))
	if len(s) > limit {
		return s[:limit] + "…"
	}
	return s
}
