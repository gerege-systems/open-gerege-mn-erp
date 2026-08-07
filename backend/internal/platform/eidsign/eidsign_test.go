package eidsign

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient points a live-mode client at a stub RP API.
func newTestClient(base string) *client {
	return &client{
		base:      strings.TrimRight(base, "/") + "/v3",
		rpUUID:    "rp-uuid",
		rpName:    "Gerege ERP",
		secret:    "rp_sk_test",
		certLevel: defaultLevel,
		http:      &http.Client{Timeout: 5 * time.Second},
		mockStore: newMockSessions(),
	}
}

func TestSignByETSISendsDigestNotChallenge(t *testing.T) {
	var gotPath string
	var body signBody
	var auth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		// Decoding into signBody would hide a stray rpChallenge, so assert on
		// the raw object too.
		var generic map[string]any
		if err := json.Unmarshal(raw, &generic); err != nil {
			t.Errorf("initiate body is not JSON: %v", err)
		}
		if _, present := generic["rpChallenge"]; present {
			t.Error("signature initiate must not carry rpChallenge; that field is authentication-only")
		}
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sessionID":"sess-1","vc":{"value":"4821"}}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.Sign(context.Background(), SignRequest{
		PersonEtsi:  "PNOMN-111949212017",
		Digest:      DigestOf([]byte("hello")),
		DisplayText: "Гэрээ.pdf",
		FileName:    "Гэрээ.pdf",
		OnBehalfOf:  "NTRMN-1234567",
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if want := "/v3/signature/notification/etsi/PNOMN-111949212017"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if auth != "Bearer rp_sk_test" {
		t.Errorf("Authorization = %q, want the RP bearer secret", auth)
	}
	if body.Digest != DigestOf([]byte("hello")) {
		t.Errorf("digest = %q, want the base64 SHA-256 of the payload", body.Digest)
	}
	if body.HashType != "SHA256" || body.SignatureProtocol != "ACSP_V2" {
		t.Errorf("hashType/protocol = %q/%q, want SHA256/ACSP_V2", body.HashType, body.SignatureProtocol)
	}
	if body.CertificateLevel != "QUALIFIED" {
		t.Errorf("certificateLevel = %q; signing must not silently accept an advanced certificate", body.CertificateLevel)
	}
	if body.OnBehalfOf != "NTRMN-1234567" {
		t.Errorf("onBehalfOf = %q, want the organisation ETSI id", body.OnBehalfOf)
	}
	if len(body.Interactions) != 1 || body.Interactions[0].Type != "displayTextAndPIN" {
		t.Errorf("interactions = %+v, want a single displayTextAndPIN", body.Interactions)
	}
	if got.SessionID != "sess-1" || got.VerificationCode != "4821" {
		t.Errorf("start = %+v, want session sess-1 with code 4821", got)
	}
}

func TestSignByDocumentNumberUsesDeviceRoute(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"sessionID":"s","vc":{"value":"1111"}}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.Sign(context.Background(), SignRequest{
		DocumentNumber: "dev-uuid",
		Digest:         DigestOf([]byte("x")),
	}); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if want := "/v3/signature/notification/document/dev-uuid"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestSignMapsUpstreamFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"unregistered RP", http.StatusUnauthorized, `{"error":"unknown rp"}`, ErrRPRejected},
		{"citizen not enrolled", http.StatusNotFound, `{"error":"no signing certificate"}`, ErrNotEnrolled},
		{"not a representative", http.StatusForbidden, `{"error":"not a representative of NTRMN-1"}`, ErrNotRepresentative},
		{"RP lacks SIGNATURE permission", http.StatusForbidden, `{"error":"permission denied"}`, ErrRPRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			_, err := newTestClient(srv.URL).Sign(context.Background(), SignRequest{
				PersonEtsi: "PNOMN-1", Digest: DigestOf([]byte("x")),
			})
			if err != tc.want {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSessionRefusalIsNotMistakenForSuccess(t *testing.T) {
	// The trap this guards: eID reports state COMPLETE for a refused ceremony
	// too. Only result.endResult distinguishes them.
	cases := []struct {
		endResult string
		want      string
	}{
		{"USER_REFUSED", StateRefused},
		{"USER_REFUSED_VC_CHOICE", StateRefused},
		{"WRONG_VC", StateRefused},
		{"TIMEOUT", StateExpired},
		{"", StateFailed},
	}
	for _, tc := range cases {
		t.Run(tc.endResult, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"state":"COMPLETE","result":{"endResult":"` + tc.endResult + `"}}`))
			}))
			defer srv.Close()

			got, err := newTestClient(srv.URL).Session(context.Background(), "s", time.Second)
			if err != nil {
				t.Fatalf("Session: %v", err)
			}
			if got.State != tc.want {
				t.Errorf("state = %q, want %q", got.State, tc.want)
			}
			if got.SignatureValue != "" {
				t.Error("a refused ceremony must not carry a signature value")
			}
		})
	}
}

func TestSessionCompleteReturnsSignatureAndDelegation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("timeoutMs"); got != "25000" {
			t.Errorf("timeoutMs = %q, want 25000", got)
		}
		_, _ = w.Write([]byte(`{
			"state":"COMPLETE",
			"result":{"endResult":"OK","documentNumber":"dev-9"},
			"signature":{"value":"c2ln","signatureAlgorithm":"sha256WithRSAEncryption"},
			"cert":{"value":"Y2VydA==","certificateLevel":"QUALIFIED"},
			"onBehalfOf":{"orgEtsi":"NTRMN-1234567","orgName":"Demo Corporation"}
		}`))
	}))
	defer srv.Close()

	got, err := newTestClient(srv.URL).Session(context.Background(), "s", 25*time.Second)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if got.State != StateComplete || got.SignatureValue != "c2ln" {
		t.Errorf("session = %+v, want a COMPLETE result carrying the signature", got)
	}
	if got.CertificateDER != "Y2VydA==" || got.CertificateLevel != "QUALIFIED" {
		t.Errorf("certificate = %q/%q, want the DER and its level", got.CertificateDER, got.CertificateLevel)
	}
	if got.OnBehalfOfName != "Demo Corporation" || got.DocumentNumber != "dev-9" {
		t.Errorf("session = %+v, want the delegation and device recorded", got)
	}
}

func TestSessionRejectsCompleteWithoutSignature(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"state":"COMPLETE","result":{"endResult":"OK"}}`))
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).Session(context.Background(), "s", time.Second); err == nil {
		t.Error("a COMPLETE+OK session with no signature must be an error, not an empty success")
	}
}

func TestSessionCapsLongPollAtProtocolLimit(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("timeoutMs")
		_, _ = w.Write([]byte(`{"state":"RUNNING"}`))
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).Session(context.Background(), "s", 10*time.Minute); err != nil {
		t.Fatalf("Session: %v", err)
	}
	if got != "120000" {
		t.Errorf("timeoutMs = %q, want it clamped to the protocol maximum of 120000", got)
	}
}

func TestStampReturnsSignedPDF(t *testing.T) {
	var gotFileName, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFileName = r.URL.Query().Get("fileName")
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(append([]byte("%PDF-1.7 signed "), body...))
	}))
	defer srv.Close()

	signed, err := newTestClient(srv.URL).Stamp(context.Background(), "sess-1", "Гэрээ.pdf", []byte("%PDF-1.4 original"))
	if err != nil {
		t.Fatalf("Stamp: %v", err)
	}
	if gotFileName != "Гэрээ.pdf" {
		t.Errorf("fileName = %q, want the original document name", gotFileName)
	}
	if gotContentType != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", gotContentType)
	}
	if !strings.HasPrefix(string(signed), "%PDF-") {
		t.Errorf("signed document = %q, want a PDF", snippet(signed))
	}
}

func TestStampRejectsNonPDFResponse(t *testing.T) {
	// An edge proxy answering 200 with an HTML error page would otherwise be
	// stored as the signed document and only fail when a citizen opened it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>502 Bad Gateway</body></html>"))
	}))
	defer srv.Close()

	if _, err := newTestClient(srv.URL).Stamp(context.Background(), "s", "a.pdf", []byte("%PDF-1.4")); err == nil {
		t.Error("a non-PDF stamp response must be rejected, not stored as a signed document")
	}
}

func TestPersonEtsiFor(t *testing.T) {
	cases := map[string]string{
		"111949212017":       "PNOMN-111949212017",
		"PNOMN-111949212017": "PNOMN-111949212017",
		"pnomn-999":          "PNOMN-999",
		"  111  ":            "PNOMN-111",
		"":                   "",
	}
	for in, want := range cases {
		if got := PersonEtsiFor(in); got != want {
			t.Errorf("PersonEtsiFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOrgEtsiFor(t *testing.T) {
	cases := map[string]string{
		"1234567":       "NTRMN-1234567",
		"NTRMN-1234567": "NTRMN-1234567",
		"":              "",
	}
	for in, want := range cases {
		if got := OrgEtsiFor(in); got != want {
			t.Errorf("OrgEtsiFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDigestOfIsBase64SHA256(t *testing.T) {
	// Known SHA-256 of "abc", base64 encoded.
	const want = "ungWv48Bz+pBQUDeXa4iI7ADYaOWF3qctBD/YfIAFa0="
	if got := DigestOf([]byte("abc")); got != want {
		t.Errorf("DigestOf = %q, want %q", got, want)
	}
	if _, err := base64.StdEncoding.DecodeString(DigestOf([]byte("abc"))); err != nil {
		t.Errorf("digest is not valid base64: %v", err)
	}
}

func TestTruncate60DoesNotSplitCyrillicRunes(t *testing.T) {
	// Mongolian Cyrillic is two bytes per rune, so a byte slice would both cut
	// past the 60-character protocol cap and leave an invalid rune behind.
	long := strings.Repeat("б", 80)
	got := truncate60(long)
	if n := len([]rune(got)); n != 60 {
		t.Errorf("truncate60 kept %d runes, want 60", n)
	}
	if !utf8Valid(got) {
		t.Error("truncate60 produced invalid UTF-8")
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

func TestParseVCAcceptsBothWireShapes(t *testing.T) {
	if got := parseVC(json.RawMessage(`{"value":"1234"}`)); got != "1234" {
		t.Errorf("object vc = %q, want 1234", got)
	}
	if got := parseVC(json.RawMessage(`"5678"`)); got != "5678" {
		t.Errorf("string vc = %q, want 5678", got)
	}
	if got := parseVC(nil); got != "" {
		t.Errorf("absent vc = %q, want empty", got)
	}
}

func TestMockCeremonyCompletesAfterApproval(t *testing.T) {
	c := &client{mock: true, mockStore: newMockSessions()}
	start, err := c.Sign(context.Background(), SignRequest{PersonEtsi: "PNOMN-1", Digest: "d", FileName: "a.pdf"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(start.VerificationCode) != 4 {
		t.Errorf("verification code = %q, want four digits", start.VerificationCode)
	}

	// A mock that completed instantly would never exercise the polling UI.
	got, err := c.Session(context.Background(), start.SessionID, time.Second)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if got.State != StateRunning {
		t.Errorf("state = %q immediately after start, want RUNNING", got.State)
	}

	c.mockStore.sessions[start.SessionID].started = time.Now().Add(-2 * mockApproval)
	got, err = c.Session(context.Background(), start.SessionID, time.Second)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if got.State != StateComplete || got.SignatureValue == "" {
		t.Errorf("session = %+v, want COMPLETE with a signature", got)
	}

	signed, err := c.Stamp(context.Background(), start.SessionID, "a.pdf", []byte("%PDF-1.4 x"))
	if err != nil {
		t.Fatalf("Stamp: %v", err)
	}
	if string(signed) != "%PDF-1.4 x" {
		t.Errorf("mock stamp = %q, want the submitted PDF returned unchanged", signed)
	}
}

func TestMockUnknownSessionExpires(t *testing.T) {
	c := &client{mock: true, mockStore: newMockSessions()}
	got, err := c.Session(context.Background(), "nope", time.Second)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	if got.State != StateExpired {
		t.Errorf("state = %q for an unknown session, want EXPIRED", got.State)
	}
}
