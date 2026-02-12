package discoclient

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/lib/clients/tlsclient"
	jsoniter "github.com/json-iterator/go"
)

// The DirectoryClient holds discovered TDs from a directory service.
type DirectoryClient struct {
	mux sync.RWMutex

	// auth token to read the TD
	authToken string

	caCert *x509.Certificate

	// The connecting client
	clientID string

	// directory of TD documents
	directory map[string]*td.TD

	// The TD of the directory itself
	directoryTD *td.TD
	// Full URL to wget the directory TD
	//directoryTDURL string

	// connection pool by connection URL to access things as per TD form.
	// The url is protocol specific.
	//connectPool map[string]transports.IClientConnection

	// Connection to the directory for reading or update
	tlsClient *tlsclient.TLSClient

	timeout time.Duration
	// client for talking to the directory using http, wss or sse
	//httpClient *tlsclient.TLSClient
}

// NewClient creates a client for connecting to access a Thing based on the
// given TD. If the thingID is not known then this returns nil
//func (cl *DirectoryClient) NewClient(tdi *td.TD) transports.IClientConnection {
//	base := td.Base
//	// how to determine the protocol from the base URL???
//	// Option 1: see below
//	//	 https://host:port/
//	//	 wss://host:port/wss        <- not base scheme
//	//	 sse://host:port/sse        <- not base scheme
//	//	 mqtt://host:port/...
//	// Option 2: use form
//	//		which one?
//	//		wait with creating client until invoking an operation?
//	//        implies invoking operation on the directory
//	//            directory.Invoke(thingID, op, ...)
//	protocol := td.Protocol
//	cc, err := clients.NewClient(base, protocol, cl.clientID, cl.caCert, cl.GetForm, cl.timeout)
//	return cc
//}

// GetForms provides the forms for invoking an operation on a thing
// This returns nil if the thing is unknown
func (cl *DirectoryClient) GetForms(thingID string, operation string) []td.Form {
	cl.mux.RLock()
	tdi, found := cl.directory[thingID]
	defer cl.mux.RUnlock()
	if !found {
		return nil
	}
	f := tdi.GetForms(operation, "")
	return f
}

// Connect connects to the directory, reads the directory's TD.
// If successful, call List to read the content.
//
// tdURL is the URL of the directory service TD.
func (cl *DirectoryClient) Connect(tdURL string) error {
	cl.mux.RLock()
	defer cl.mux.RUnlock()

	// 1: connect to the directory and read its TD
	parts, _ := url.Parse(tdURL)
	tlsClient := tlsclient.NewTLSClient(parts.Host, nil, cl.caCert, cl.timeout)
	tlsClient.SetAuthToken(cl.authToken)
	cl.tlsClient = tlsClient

	// 2: read its TD
	resp, status, err := tlsClient.Get(parts.Path)
	_ = status
	if err != nil {
		return err
	}
	tdi := td.TD{}
	err = jsoniter.Unmarshal(resp, &tdi)
	if err != nil {
		return err
	}
	cl.directoryTD = &tdi
	return nil
}

// ListTD pages through a list of things to update the local directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api-things-listing
// which requires the http get at /things?limit=...
func (cl *DirectoryClient) ListTD(limit int) error {

	listPath := fmt.Sprintf("/things?limit=%d", limit)
	raw, stat, err := cl.tlsClient.Get(listPath)
	_ = stat
	if err != nil {
		return err
	}
	tdList := []*td.TD{}
	err = jsoniter.Unmarshal(raw, &tdList)
	if err != nil {
		return err
	}
	//
	cl.mux.Lock()
	for _, tdi := range tdList {
		cl.directory[tdi.ID] = tdi
	}
	cl.mux.Unlock()
	return nil
}

// ReadTD refreshes the cached TD from the directory
// This follows: https://w3c.github.io/wot-discovery/#exploration-directory-api
// which requires the http get at /things/{id}
func (cl *DirectoryClient) ReadTD(thingID string) (*td.TD, error) {

	raw, stat, err := cl.tlsClient.Get("/things/" + thingID)
	_ = stat
	if err != nil {
		return nil, err
	}
	tdi := &td.TD{}
	err = jsoniter.Unmarshal(raw, &tdi)
	if err != nil {
		return nil, err
	}
	//
	cl.mux.Lock()
	cl.directory[thingID] = tdi
	cl.mux.Unlock()
	return tdi, err
}

// UpdateThing updates the TD document in the remote directory.
// This does not update the local directory.
// Intended for use by Thing agents to publish their TD. Regular consumers
// are not allowed to do this.
//
//	tdjson is the TD document in JSON format
func (cl *DirectoryClient) UpdateThing(tdi *td.TD) error {

	tdJSON, _ := jsoniter.Marshal(&tdi)
	// FIXME: use the directory forms
	updatePath := fmt.Sprintf("/things/%s", tdi.ID)
	data, status, err := cl.tlsClient.Put(updatePath, tdJSON)
	_ = status
	_ = data
	return err
}

// NewDirectoryClient creates a new client for reading and holding the content
// of a directory.
//
// This first reads the directory's TD using https and the given auth token.
// Call ReadDirectory to establish a connection to the server using the protocol
// provided in the TD and reads the directory content. The connection is added
// to the connection pool for clients to access TDs.
//
// In Hiveot this pool is just the single connection as all Things are access via
// the digital twin on the Hub.
//
// Intended for consumers to hold the TD's they have access to.
//
// tdURL is the introduction address that provides the directory TD itself
func NewDirectoryClient(clientID string, token string, caCert *x509.Certificate, timeout time.Duration) *DirectoryClient {
	cl := &DirectoryClient{
		caCert:    caCert,
		clientID:  clientID,
		authToken: token,
		timeout:   timeout,
		directory: make(map[string]*td.TD),
	}

	return cl
}
