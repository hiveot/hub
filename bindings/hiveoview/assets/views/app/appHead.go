package app

import "github.com/hiveot/hub/bindings/hiveoview/assets/components"

//const tplName = "appbar"

var appHeadMenuItems = []components.DropdownItem{
	components.DropdownItem{
		ID:    "page1Item",
		Type:  components.MenuItemLink,
		Label: "page1",
		Value: "page1",
		Icon:  "view-dashboard",
	},
	components.DropdownItem{
		ID:    "page2Item",
		Type:  components.MenuItemLink,
		Label: "page2",
		Value: "page2",
		Icon:  "view-dashboard-outline",
	},
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
		Value: "/app/about",
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

	components.SetDropdownProps(data, "headerMenu", appHeadMenuItems)
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
