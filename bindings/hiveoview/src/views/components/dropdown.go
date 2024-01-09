package components

//const tplName = "appbar"

const (
	// MenuItemButton navigates using a button that triggers a target
	MenuItemButton = "button"
	// MenuItemCheckbox show a checkbox in the menu
	MenuItemCheckbox = "checkbox"
	// MenuItemLink navigates to href
	MenuItemLink = "link"
	// MenuItemDivider show a divider in the menu
	MenuItemDivider = "divider"
)

type DropdownItem struct {
	ID     string // Menu item ID
	Type   string // checkbox, divider, label, link
	Label  string // label to display
	Value  any    // checkbox value, link href
	Icon   any    // icon object, if any
	Target string // HX-target field for redirects
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
		dataItem["Target"] = item.Target

		dataItems = append(dataItems, dataItem)
	}
	data[menuID] = dataItems
}
