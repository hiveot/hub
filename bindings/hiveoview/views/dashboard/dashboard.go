package dashboard

import (
	"html/template"
	"log/slog"
	"net/http"
)

// GetDashboard renders the dashboard view
func GetDashboard(t *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{}
		err := t.ExecuteTemplate(w, "dashboard.html", data)
		if err != nil {
			slog.Error("Error rendering template", "err", err)
		}
	}
}
