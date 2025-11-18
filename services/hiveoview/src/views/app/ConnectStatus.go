package app

import (
	"github.com/hiveot/hivehub/services/hiveoview/src"
	"github.com/hiveot/hivehub/services/hiveoview/src/session"
	"net/http"
)

const RenderConnectStatusTemplate = "ConnectStatus.gohtml"

// ConnectStatusTemplateData describes the message bus connection status of the current session
type ConnectStatusTemplateData struct {
	// the login ID which is used to connect
	LoginID string
	// description of the connection status
	Description string
	// mdi icon set icon name representing the status
	IconName string
	// optional error text if connection failed
	Error string
	// simple flag whether a connection is established
	IsConnected             bool
	RenderConnectStatusPath string
}

// GetConnectStatus returns the description of the connection status
func GetConnectStatus(r *http.Request) *ConnectStatusTemplateData {
	_, cs, _ := session.GetSessionFromContext(r)
	status := &ConnectStatusTemplateData{
		IconName:                "link-off",
		Description:             "disconnected",
		IsConnected:             false,
		Error:                   "",
		RenderConnectStatusPath: src.RenderConnectionStatusPath,
	}
	if cs == nil {
		status.Description = "Session not established"
	} else {
		isConnected := cs.IsConnected()
		lastError := cs.GetLastError()
		status.LoginID = cs.GetClientID()
		if lastError != nil {
			status.Error = lastError.Error()
		}
		if isConnected {
			status.IconName = "link"
			status.Description = "Connected to the Hub"
			status.IsConnected = true
		} else if lastError != nil {
			status.IconName = "link-off"
			status.Description = "Connection failed: " + lastError.Error()
		} else {
			status.IconName = "link-off"
			status.Description = "Not Connected"
		}
	}
	return status
}

// RenderConnectStatus renders the presentation of the client connection to the Hub message bus.
// This only renders the fragment. On a full page refresh this renders inside the base.html
func RenderConnectStatus(w http.ResponseWriter, r *http.Request) {
	status := GetConnectStatus(r)
	_, sess, _ := session.GetSessionFromContext(r)

	// render with base or as fragment
	//views.TM.RenderTemplate(w, r, RenderConnectStatusTemplate, data)
	buff, err := RenderAppOrFragment(r, RenderConnectStatusTemplate, status)
	sess.WritePage(w, buff, err)
}
