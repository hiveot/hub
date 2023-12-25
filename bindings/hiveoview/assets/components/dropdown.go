package components

//const tplName = "appbar"

const (
	MenuItem_Checkbox = "checkbox"
	MenuItem_Link     = "link"
	MenuItem_Divider  = "divider"
)

type DropdownItem struct {
	ID    string // Menu item ID
	Type  string // checkbox, divider, label, link
	Label string // label to display
	Value any    // checkbox value, link href
	Icon  any    // icon object, if any
}

// SetDropdownProps sets the properties used in rendering a dropdown menu
//
//	data is contains the existing template properties to add to
//	menuID is the ID under which the menu is stored for use by the template
//	items is the list of menu items
func SetDropdownProps(data map[string]any, menuID string, items []DropdownItem) {
	dataItems := make([]any, 0)
	for _, item := range items {
		dataItem := make(map[string]any)
		dataItem["Type"] = item.Type
		dataItem["Label"] = item.Label
		dataItem["Value"] = item.Value
		dataItem["Icon"] = item.Icon
		dataItems = append(dataItems, dataItem)
	}
	data[menuID] = dataItems
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
