package ssoprovider_test

import (
	"testing"
	"time"

	"github.com/gerege-systems/open-gerege-nexus/backend/internal/platform/ssoprovider"
)

func TestSSOProviderTokenLifecycle(t *testing.T) {
	provider := ssoprovider.NewSSOProvider()

	// 1. Check default client
	client, err := provider.GetClient("gerege-dev-portal")
	if err != nil {
		t.Fatalf("expected default client to be registered: %v", err)
	}
	if client.ClientName != "Gerege Developer Portal App" {
		t.Errorf("unexpected client name: %s", client.ClientName)
	}

	// 2. Issue token
	token, err := provider.IssueToken(client.ClientID, "user_01", "openid profile", 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}

	// 3. Introspect active token
	res := provider.IntrospectToken(token)
	if !res.Active {
		t.Errorf("expected token to be active")
	}
	if res.ClientID != client.ClientID {
		t.Errorf("expected client_id %s, got %s", client.ClientID, res.ClientID)
	}

	// 4. Revoke token
	revoked := provider.RevokeToken(token)
	if !revoked {
		t.Errorf("expected token revocation to return true")
	}

	resAfter := provider.IntrospectToken(token)
	if resAfter.Active {
		t.Errorf("expected token to be inactive after revocation")
	}
}

func TestSSOProviderRegisterClient(t *testing.T) {
	provider := ssoprovider.NewSSOProvider()

	newClient := &ssoprovider.OAuth2Client{
		ClientID:     "custom-partner-app",
		ClientSecret: "secret_123",
		ClientName:   "Custom ERP Partner App",
		Scopes:       []string{"erp.read"},
	}
	provider.RegisterClient(newClient)

	c, err := provider.GetClient("custom-partner-app")
	if err != nil {
		t.Fatalf("unexpected error finding client: %v", err)
	}
	if c.ClientName != "Custom ERP Partner App" {
		t.Errorf("unexpected client name: %s", c.ClientName)
	}
}
