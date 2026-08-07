/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, @craftzbay, Gemini AI & Claude AI
 * Distributed under the Apache 2.0 License.
 *
 * Package ssoprovider provides an ORY Hydra-grade OAuth2 and OpenID Connect (OIDC)
 * Single Sign-On (SSO) Identity Provider engine.
 */

package ssoprovider

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/config"
)

type OAuth2Client struct {
	ID string `json:"id"`
	// ClientSecret is populated only in the response that creates the client;
	// every later read redacts it (see Redacted).
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	ClientName   string    `json:"client_name"`
	RedirectURIs []string  `json:"redirect_uris"`
	GrantTypes   []string  `json:"grant_types"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
}

// Redacted returns a copy safe to serialise to API consumers.
func (c *OAuth2Client) Redacted() *OAuth2Client {
	if c == nil {
		return nil
	}
	clone := *c
	clone.ClientSecret = ""
	return &clone
}

type TokenIntrospection struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	TokenType string `json:"token_type,omitempty"`
}

type SSOProvider struct {
	mu      sync.RWMutex
	issuer  string
	clients map[string]*OAuth2Client
	tokens  map[string]*TokenIntrospection
}

func NewSSOProvider() *SSOProvider {
	issuer := os.Getenv("SSO_ISSUER")
	if issuer == "" {
		// Legacy deployment endpoint, retained through the Gerege Nexus rebrand.
		// The issuer is baked into every token already granted and into the
		// relying parties' configuration, so it cannot follow a product rename;
		// it changes only when a new origin is provisioned and the clients are
		// re-registered. Deployments override it with SSO_ISSUER.
		issuer = "https://openerp.gerege.mn"
	}

	provider := &SSOProvider{
		issuer:  issuer,
		clients: make(map[string]*OAuth2Client),
		tokens:  make(map[string]*TokenIntrospection),
	}

	// Bootstrap the built-in developer-portal client. Its secret used to be a
	// constant compiled into the binary — a published credential for every
	// deployment. It now comes from the environment, and outside production a
	// random one is generated and logged for local use.
	secret := os.Getenv("SSO_DEFAULT_CLIENT_SECRET")
	switch {
	case secret != "":
	case config.IsProduction():
		slog.Warn("SSO_DEFAULT_CLIENT_SECRET is not set — the built-in developer portal client is disabled")
		return provider
	default:
		secret = "sec_" + generateRandomString(32)
		slog.Info("generated development SSO client secret",
			"client_id", "gerege-dev-portal", "client_secret", secret)
	}

	provider.RegisterClient(&OAuth2Client{
		ID:           "cli_01",
		ClientID:     "gerege-dev-portal",
		ClientSecret: secret,
		ClientName:   "Gerege Developer Portal App",
		// The https:// entry is the legacy deployment endpoint. A redirect URI is
		// matched exactly against what the client sends, so it stays until a new
		// origin exists and this allowlist is extended alongside it.
		RedirectURIs: []string{"http://localhost:3000/callback", "https://openerp.gerege.mn/callback"},
		GrantTypes:   []string{"authorization_code", "client_credentials", "refresh_token"},
		// erp.read/erp.write are legacy compatibility scope names, kept through
		// the Gerege Nexus rebrand. They are protocol identifiers already held
		// in issued tokens and third-party client registrations; renaming them
		// would invalidate live grants.
		Scopes:    []string{"openid", "profile", "email", "erp.read", "erp.write"},
		CreatedAt: time.Now(),
	})

	return provider
}

func (s *SSOProvider) RegisterClient(client *OAuth2Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client.ID == "" {
		client.ID = generateRandomString(12)
	}
	if client.ClientID == "" {
		client.ClientID = "app_" + generateRandomString(16)
	}
	if client.ClientSecret == "" {
		client.ClientSecret = "sec_" + generateRandomString(32)
	}
	client.CreatedAt = time.Now()
	s.clients[client.ClientID] = client
}

// ListClients returns every registered client with its secret redacted. The
// secret is shown once, in the response to the call that creates the client.
func (s *SSOProvider) ListClients() []*OAuth2Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*OAuth2Client, 0, len(s.clients))
	for _, c := range s.clients {
		list = append(list, c.Redacted())
	}
	slices.SortFunc(list, func(a, b *OAuth2Client) int {
		return strings.Compare(a.ClientID, b.ClientID)
	})
	return list
}

func (s *SSOProvider) GetClient(clientID string) (*OAuth2Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[clientID]
	if !ok {
		return nil, errors.New("client not found")
	}
	return client, nil
}

// IssueToken generates a new OAuth2 Access Token
func (s *SSOProvider) IssueToken(clientID, sub, scope string, duration time.Duration) (string, error) {
	token := "hydra_at_" + generateRandomString(32)
	exp := time.Now().Add(duration).Unix()

	s.mu.Lock()
	s.tokens[token] = &TokenIntrospection{
		Active:    true,
		Scope:     scope,
		ClientID:  clientID,
		Sub:       sub,
		Exp:       exp,
		TokenType: "Bearer",
	}
	s.mu.Unlock()

	return token, nil
}

// IntrospectToken inspects token validity (ORY Hydra standard)
func (s *SSOProvider) IntrospectToken(token string) *TokenIntrospection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tokens[token]
	if !ok {
		return &TokenIntrospection{Active: false}
	}

	if time.Now().Unix() > t.Exp {
		return &TokenIntrospection{Active: false}
	}

	return t
}

// RevokeToken invalidates an issued token
func (s *SSOProvider) RevokeToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tokens[token]; ok {
		delete(s.tokens, token)
		return true
	}
	return false
}

// HTTP Handlers for OpenID Connect & OAuth2 Provider

// HandleOIDCDiscovery advertises only what this provider actually implements.
// Announcing authorization_code/refresh_token and RS256 id_tokens while the
// server issues opaque client_credentials tokens breaks conformant clients at
// the first redirect.
func (s *SSOProvider) HandleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	doc := map[string]any{
		"issuer":                                s.issuer,
		"token_endpoint":                        s.issuer + "/oauth2/token",
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"introspection_endpoint":                s.issuer + "/oauth2/introspect",
		"revocation_endpoint":                   s.issuer + "/oauth2/revoke",
		"response_types_supported":              []string{},
		"grant_types_supported":                 []string{"client_credentials"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"subject_types_supported":               []string{"public"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "erp.read", "erp.write"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// HandleJWKS returns an empty key set: access tokens are opaque and validated
// through /oauth2/introspect. The previous handler published a placeholder RSA
// key ("n": "mock_rsa_n_val") that no client could ever verify against.
func (s *SSOProvider) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{}})
}

// AuthenticateClient verifies client credentials in constant time.
//
// The previous condition — `clientSecret != "" && client.ClientSecret != secret`
// — skipped verification entirely when the request omitted the secret, so any
// caller who knew a client_id could mint access tokens.
func (s *SSOProvider) AuthenticateClient(clientID, clientSecret string) (*OAuth2Client, error) {
	client, err := s.GetClient(clientID)
	if err != nil {
		return nil, ErrInvalidClient
	}
	if subtle.ConstantTimeCompare([]byte(client.ClientSecret), []byte(clientSecret)) != 1 {
		return nil, ErrInvalidClient
	}
	return client, nil
}

// ErrInvalidClient is returned for any client authentication failure; callers
// must not distinguish "unknown client" from "wrong secret".
var ErrInvalidClient = errors.New("invalid_client")

func (s *SSOProvider) HandleTokenEndpoint(w http.ResponseWriter, r *http.Request) {
	// RFC 6749 §2.3.1 allows credentials either in the body or via HTTP Basic.
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	client, err := s.AuthenticateClient(clientID, clientSecret)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth2"`)
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType == "" {
		grantType = "client_credentials"
	}
	if len(client.GrantTypes) > 0 && !slices.Contains(client.GrantTypes, grantType) {
		writeOAuthError(w, http.StatusBadRequest, "unauthorized_client",
			"grant type "+grantType+" is not enabled for this client")
		return
	}
	// Only client_credentials is implemented today; advertising support for
	// authorization_code without an /oauth2/auth endpoint would be a lie.
	if grantType != "client_credentials" {
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type",
			"only client_credentials is currently implemented")
		return
	}

	scope := strings.Join(client.Scopes, " ")
	accessToken, err := s.IssueToken(client.ClientID, client.ClientID, scope, tokenTTL)
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(tokenTTL.Seconds()),
		"scope":        scope,
	})
}

// HandleIntrospectEndpoint requires client authentication: an unauthenticated
// introspection endpoint lets anyone probe token validity (RFC 7662 §2.1).
func (s *SSOProvider) HandleIntrospectEndpoint(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if _, err := s.AuthenticateClient(clientID, clientSecret); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	res := s.IntrospectToken(r.FormValue("token"))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *SSOProvider) HandleRevokeEndpoint(w http.ResponseWriter, r *http.Request) {
	clientID, clientSecret, hasBasic := r.BasicAuth()
	if !hasBasic {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if _, err := s.AuthenticateClient(clientID, clientSecret); err != nil {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	s.RevokeToken(r.FormValue("token"))
	w.WriteHeader(http.StatusOK)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

const tokenTTL = 1 * time.Hour

// generateRandomString returns n hex characters of crypto/rand output.
//
// The old implementation drew n bytes and then truncated the 2n-character hex
// string back to n, silently halving the entropy of every generated secret.
func generateRandomString(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand never fails on supported platforms; treat it as fatal
		// rather than returning a predictable value.
		panic("ssoprovider: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)[:n]
}
