package menu

import "testing"

func TestEveryAppBlueprintHasTwoPopulatedGroups(t *testing.T) {
	for appID, bp := range blueprints {
		// The working module screen is contributed by mod.Menus(), so two future
		// module entries make the group total at least three.
		if len(bp.Modules) < 2 {
			t.Errorf("%s module group has %d future entries; want at least 2", appID, len(bp.Modules))
		}
		if len(bp.Settings) < 3 {
			t.Errorf("%s settings group has %d entries; want at least 3", appID, len(bp.Settings))
		}
		if bp.Slug == "" {
			t.Errorf("%s has an empty route slug", appID)
		}
	}
}
