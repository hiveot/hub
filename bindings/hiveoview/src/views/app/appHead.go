package app

//const AppHeadTemplate = "appHead.gohtml"
//const AppMenuTemplate = "appMenu.gohtml"

// GetAppHeadProps returns the properties used in rendering the appbar component
// TODO: get pages from client config/session store
func GetAppHeadProps(data map[string]any, title string, logo string) {
	data["logo"] = logo
	data["title"] = title
}
