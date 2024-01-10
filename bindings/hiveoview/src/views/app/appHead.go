package app

// GetAppHeadProps returns the properties used in rendering the appbar component
// TODO: get pages from client config/session store
func GetAppHeadProps(data map[string]any, title string, logo string, pages []string) {
	data["logo"] = logo
	data["title"] = title
	data["pages"] = pages
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
