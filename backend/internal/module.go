package internal

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Dependency describes an app module dependency and semver constraint.
type Dependency struct {
	ID                string `json:"id"`
	VersionConstraint string `json:"version_constraint"`
}

// PermissionDefinition defines an RBAC permission provided by a module.
type PermissionDefinition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MenuDefinition defines a navigation menu item for an app module.
type MenuDefinition struct {
	ID       string `json:"id"`
	AppID    string `json:"app_id,omitempty"`
	AppName  string `json:"app_name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Label    string `json:"label"`
	Path     string `json:"path,omitempty"`
	Icon     string `json:"icon"`
	Order    int    `json:"order"`

	// Labels holds per-locale overrides keyed by ISO 639-1 code. The menu API
	// resolves Label from the caller's locale before responding, so the client
	// never has to translate server-owned content.
	Labels map[string]string `json:"-"`
}

// LocalizedLabel returns the label for the requested locale, falling back to
// the default Label when no translation exists.
func (m MenuDefinition) LocalizedLabel(locale string) string {
	if label, ok := m.Labels[locale]; ok && label != "" {
		return label
	}
	return m.Label
}

// Module defines the contract every compile-time app module must implement.
type Module interface {
	ID() string
	Name() string
	Version() string
	Dependencies() []Dependency
	Permissions() []PermissionDefinition
	Menus() []MenuDefinition
	RegisterRoutes(r chi.Router, tenantAuthMiddleware func(http.Handler) http.Handler)
}
