package login

import (
	"fmt"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/views"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
)

const LoginTemplateFile = "login.gohtml"

type LoginTemplateData struct {
	LoginID       string
	LoginError    string
	PostLoginPath string
	Cid           string // just for rendering the login page - no value expected
}

// RenderLogin renders the login form
func RenderLogin(w http.ResponseWriter, r *http.Request) {

	// hx-headers doesnt work on posting a form, so use query instead to pass a CID
	cid := "login-" + shortid.MustGenerate()
	postLoginPath := fmt.Sprintf("%s?%s=%s", src.UIPostFormLoginPath, httpserver.ConnectionIDHeader, cid)
	data := LoginTemplateData{
		LoginID:       "",
		LoginError:    "",
		PostLoginPath: postLoginPath,
		Cid:           cid,
	}

	loginError := r.URL.Query().Get("error")
	if loginError != "" {
		data.LoginError = loginError
	}

	// don't cache the login
	// FIXME: delete the post from history so that a back button press doesn't re-post login cred.
	// apparently the cache control doesn't help for this.
	w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
	buff, err := views.TM.RenderFull(LoginTemplateFile, data)
	if err != nil {
		slog.Error("Login render error:", "err", err.Error())
	}
	_ = err
	_, _ = buff.WriteTo(w)
}
