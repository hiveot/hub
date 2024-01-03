package app

import (
	"fmt"
	"github.com/hiveot/hub/bindings/hiveoview/assets/components"
)

//const tplName = "appbar"

var appHeadMenuItems = []components.DropdownItem{
	components.DropdownItem{
		Type: components.MenuItemDivider,
	},
	components.DropdownItem{
		ID:    "editModeItem",
		Type:  components.MenuItemCheckbox,
		Label: "Edit Mode",
		Value: "false",
	},
	components.DropdownItem{
		ID:    "logout",
		Type:  components.MenuItemPost, // buttons post while links get
		Label: "Logout",
		Value: "/logout",
		Icon:  "logout",
	},
	components.DropdownItem{
		ID:    "aboutItem",
		Type:  components.MenuItemLink,
		Label: "About Hiveoview",
		Value: "/about",
		Icon:  "info",
	},
}

// GetAppHeadProps returns the properties used in rendering the appbar component
// TODO: get pages from client config/session store
func GetAppHeadProps(data map[string]any, title string, logo string, pages []string) {
	data["logo"] = logo
	data["title"] = title
	data["pages"] = pages
	data["menuItems"] = append(pages, []string{
		"",
		"About...",
	}...)

	// dynamically add the pages as menu items
	menuItems := make([]components.DropdownItem, 0)
	for i, page := range pages {
		menuItems = append(menuItems, components.DropdownItem{
			ID:    fmt.Sprintf("%s-%d", page, i),
			Type:  components.MenuItemLink,
			Label: page,
			Value: "/app/" + page,
			Icon:  "view-dashboard",
		})
	}
	menuItems = append(menuItems, appHeadMenuItems...)
	components.SetDropdownProps(data, "headerMenu", menuItems)
}

//
//// GetAppbar component renderer
//func GetAppbar(t *template.Template,
//	title string, logo string, pages []string) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		data := map[string]any{
//			"logo":  logo,
//			"title": title,
//			"pages": pages,
//		}
//		layouts.Render(w, t, tplName, data)
//
//	}
//}
