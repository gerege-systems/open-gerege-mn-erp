package platform

import "testing"

func TestRoleCodeValidation(t *testing.T) {
	valid := []string{"admin", "sales_manager", "inventory.read"}
	invalid := []string{"A", "Admin", " has-space", "x/owner", "-admin"}
	for _, v := range valid {
		if !roleCodePattern.MatchString(v) {
			t.Errorf("expected %q valid", v)
		}
	}
	for _, v := range invalid {
		if roleCodePattern.MatchString(v) {
			t.Errorf("expected %q invalid", v)
		}
	}
}

func TestAppRequestPermission(t *testing.T) {
	if got := appRequestPermission("io.example.contacts", "GET"); got != "contacts.read" {
		t.Fatalf("got %q", got)
	}
	if got := appRequestPermission("io.example.contacts", "POST"); got != "contacts.manage" {
		t.Fatalf("got %q", got)
	}
	if got := appRequestPermission("io.example.gov_services", "POST"); got != "" {
		t.Fatalf("government workflow must keep action-level checks, got %q", got)
	}
}

func TestValidEIDCallback(t *testing.T) {
	t.Setenv("PUBLIC_ORIGIN", "https://openerp.gerege.mn")
	t.Setenv("ENVIRONMENT", "production")
	if got, err := validEIDCallback("https://openerp.gerege.mn/auth/eid/callback"); err != nil || got == "" {
		t.Fatalf("expected callback to be accepted: %q, %v", got, err)
	}
	for _, raw := range []string{"http://openerp.gerege.mn/auth/eid/callback", "https://evil.example/auth/eid/callback", "https://openerp.gerege.mn/login"} {
		if _, err := validEIDCallback(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
