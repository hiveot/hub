package app

import "github.com/hiveot/hub/bindings/hiveoview/assets/components"

//const tplName = "appbar"

var appHeadMenuItems = []components.DropdownItem{
	components.DropdownItem{
		ID:    "page1Item",
		Type:  components.MenuItem_Link,
		Label: "page1",
		Value: "page1",
	},
	components.DropdownItem{
		ID:    "page2Item",
		Type:  components.MenuItem_Link,
		Label: "page2",
		Value: "page2",
	},
	components.DropdownItem{
		Type: components.MenuItem_Divider,
	},
	components.DropdownItem{
		ID:    "editModeItem",
		Type:  components.MenuItem_Checkbox,
		Label: "Edit Mode",
		Value: "false",
	},
	components.DropdownItem{
		ID:    "aboutItem",
		Type:  components.MenuItem_Link,
		Label: "About Hiveoview",
		Value: "/app/about",
		Icon:  "mdi:info",
	},
}

// SetAppHeadProps sets the properties used in rendering the appbar component
// TODO: get pages from client config/session store
func SetAppHeadProps(data map[string]any, title string, logo string, pages []string) {
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
