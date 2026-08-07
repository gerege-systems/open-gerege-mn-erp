package eidsign

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// mockSessions serves canned ceremonies so the app, its screens and its tests
// can run without a live relying party registration.
//
// The mock deliberately keeps the *shape* of the real thing — a session that
// stays RUNNING for a moment, a four-digit verification code, a terminal
// COMPLETE — because the polling UI is the part most likely to be wrong, and a
// mock that completed instantly would never exercise it.
//
// What it cannot fake is the cryptography. Stamp returns the submitted PDF
// unchanged: there is no PKCS#7, no certificate chain and no revocation data
// in it. A document signed in mock mode is a demo artefact, and callers mark
// it as such rather than presenting it as a legally valid signature.
type mockSessions struct {
	mu       sync.Mutex
	sessions map[string]*mockSession
}

type mockSession struct {
	started    time.Time
	code       string
	fileName   string
	onBehalfOf string
}

// mockApproval is how long a mock ceremony pretends the citizen is reaching
// for their phone.
const mockApproval = 2 * time.Second

func newMockSessions() *mockSessions {
	return &mockSessions{sessions: make(map[string]*mockSession)}
}

func (m *mockSessions) start(req SignRequest) *StartResult {
	id := randomHex(16)
	session := &mockSession{
		started:    time.Now(),
		code:       randomDigits(4),
		fileName:   req.FileName,
		onBehalfOf: req.OnBehalfOf,
	}
	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()
	return &StartResult{SessionID: id, VerificationCode: session.code}
}

func (m *mockSessions) poll(sessionID string) *SessionResult {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return &SessionResult{State: StateExpired, EndResult: "TIMEOUT"}
	}
	if time.Since(session.started) < mockApproval {
		return &SessionResult{State: StateRunning}
	}
	res := &SessionResult{
		State:              StateComplete,
		EndResult:          "OK",
		DocumentNumber:     "mock-device-" + sessionID[:8],
		SignatureValue:     base64.StdEncoding.EncodeToString([]byte("MOCK_SIGNATURE_" + sessionID)),
		SignatureAlgorithm: "sha256WithRSAEncryption",
		CertificateLevel:   "QUALIFIED",
	}
	if session.onBehalfOf != "" {
		res.OnBehalfOfEtsi = session.onBehalfOf
		res.OnBehalfOfName = "Demo Corporation"
	}
	return res
}

func (m *mockSessions) stamp(sessionID string, pdf []byte) ([]byte, error) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	if time.Since(session.started) < mockApproval {
		return nil, errors.New("eidsign: session has not completed")
	}
	// Returned unchanged — see the type comment. Copied so a caller cannot
	// mutate the upload through the returned slice.
	out := make([]byte, len(pdf))
	copy(out, pdf)
	return out, nil
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand does not fail in practice; a time-derived id keeps the
		// mock usable rather than panicking a demo.
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))[:n*2]
	}
	return hex.EncodeToString(buf)
}

func randomDigits(n int) string {
	const digits = "0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "0000"
	}
	for i := range buf {
		buf[i] = digits[int(buf[i])%len(digits)]
	}
	return string(buf)
}
