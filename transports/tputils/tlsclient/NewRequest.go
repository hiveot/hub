package tlsclient

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// NewRequest creates a request object containing a bearer token if available
func NewRequest(method string, fullURL string, bearerToken string, body []byte) (*http.Request, error) {
	method = strings.ToUpper(method)
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(method, fullURL, bodyReader)

	if err != nil {
		return nil, err
	}

	// set the intended destination
	// in web browser this is the origin that provided the web page,
	// here it means the server that we'd like to talk to.
	parts, err := url.Parse(fullURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Set("Origin", origin)
	if bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+bearerToken)
	} else {
		// no authentication
	}
	return req, nil
}
