package tlsserver_old

import (
	"fmt"
	"github.com/hiveot/hub/lib/tlsclient"
	"net/http"
	"strconv"
)

// GetQueryInt reads the request query parameter and convert it to an integer
//
//	request is the request with the query parameter
//	paramName is the name of the parameter
//	defaultValue to use if parameter not provided
//
// Returns an integer value, error if conversion failed (bad request)
func (srv *TLSServer) GetQueryInt(
	request *http.Request, paramName string, defaultValue int) (value int, err error) {

	// Check for a limit parameter
	var value64 int64 = int64(defaultValue)

	parts := request.URL.Query()
	paramAsString, found := parts[paramName]
	if found {
		if len(paramAsString) == 1 {
			value64, err = strconv.ParseInt(paramAsString[0], 10, 32)
		} else {
			err = fmt.Errorf("invalid query parameter %s", paramName)
		}
	}
	return int(value64), err
}

// GetQueryLimitOffset reads the limit and offset query parameters of a given request.
// These query parameters have standardized names to limit the size of API results.
// Provide a defaultLimit for use if limit is not provided. This is also the maximum limit.
// Offset is 0 by default.
// Returns limit and offset or an error if the query parameter is incorrect
func (srv *TLSServer) GetQueryLimitOffset(request *http.Request, defaultLimit int) (limit int, offset int, err error) {

	// offset and limit are optionally provided through query params
	limit, err = srv.GetQueryInt(request, tlsclient.ParamLimit, defaultLimit)
	if defaultLimit != 0 && limit > defaultLimit {
		limit = defaultLimit
	}
	if err == nil {
		offset, err = srv.GetQueryInt(request, tlsclient.ParamOffset, 0)
	}
	return limit, offset, err
}

// GetQueryString reads the request query parameter and returns the first string
//
//	request is the request with the query parameter
//	paramName is the name of the parameter
//	defaultValue to use if parameter not provided
//
// Returns a single string, with defaultValue if not found
func (srv *TLSServer) GetQueryString(
	request *http.Request, paramName string, defaultValue string) string {

	parts := request.URL.Query()
	paramAsString, found := parts[paramName]
	if found {
		if len(paramAsString) >= 1 {
			return paramAsString[0]
		}
	}
	return defaultValue
}
