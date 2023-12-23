package components

//const tplName = "appbar"

// SetAppbarProps sets the properties used in rendering the appbar component
func SetAppbarProps(data map[string]any, title string, logo string, pages []string) {
	data["logo"] = logo
	data["title"] = title
	data["pages"] = pages
	data["menuItems"] = append(pages, []string{
		"",
		"About...",
	}...)
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
