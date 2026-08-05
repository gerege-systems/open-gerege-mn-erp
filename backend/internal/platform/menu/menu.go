package menu

import (
	"context"
	"sort"

	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal"
	"github.com/gerege-systems/open-gerege-mn-erp/backend/internal/platform/appregistry"
)

type InstalledAppStore interface {
	GetEnabledAppIDsForTenant(ctx context.Context, tenantID string) ([]string, error)
}

// Odoo-style root menus group installed application actions by business area.
// A root is emitted only when at least one enabled module contributes a child,
// preventing empty navigation branches for tenants with a small app set.
var rootMenus = []internal.MenuDefinition{
	// Path is a rolling-deploy compatibility fallback. Older frontends render
	// every API item as a Link and crash when href is absent; the hierarchy-aware
	// frontend renders these records as accordion buttons and ignores Path.
	{ID: "master_data", Label: "Master Data", Path: "/apps", Icon: "database", Order: 10, Labels: map[string]string{"mn": "Үндсэн бүртгэл"}},
	{ID: "operations", Label: "Operations", Path: "/apps", Icon: "workflow", Order: 20, Labels: map[string]string{"mn": "Үйл ажиллагаа"}},
	{ID: "platform_tools", Label: "Platform Tools", Path: "/apps", Icon: "layers", Order: 30, Labels: map[string]string{"mn": "Платформын хэрэгслүүд"}},
}

func GetTenantMenus(ctx context.Context, store InstalledAppStore, tenantID, locale string) ([]internal.MenuDefinition, error) {
	enabledAppIDs, err := store.GetEnabledAppIDsForTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	enabledMap := make(map[string]bool)
	for _, id := range enabledAppIDs {
		enabledMap[id] = true
	}

	// Serialise an empty menu set as [] rather than null.
	menus := make([]internal.MenuDefinition, 0)
	for _, mod := range appregistry.List() {
		if enabledMap[mod.ID()] {
			for _, item := range mod.Menus() {
				item.AppID = mod.ID()
				item.AppName = mod.Name()
				// Resolve the label server-side so the client renders whatever
				// the API hands it.
				item.Label = item.LocalizedLabel(locale)
				menus = append(menus, item)
			}
		}
	}

	usedParents := make(map[string]bool)
	for _, item := range menus {
		if item.ParentID != "" {
			usedParents[item.ParentID] = true
		}
	}
	for _, root := range rootMenus {
		if usedParents[root.ID] {
			root.Label = root.LocalizedLabel(locale)
			menus = append(menus, root)
		}
	}

	sort.Slice(menus, func(i, j int) bool {
		if menus[i].ParentID == menus[j].ParentID {
			return menus[i].Order < menus[j].Order
		}
		return menus[i].ID < menus[j].ID
	})

	return menus, nil
}
