package idprovclient

import (
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/services/idprov/idprovapi"
)

// IdProvUserClient is a marshaller for provisioning client messages over https
// This uses the default serializer to marshal and unmarshal messages, eg JSON
type IdProvUserClient struct {
	// The full URL of the provisioning service
	provServiceURL string
}

// SubmitIdProvRequest send a request to provision this client and obtain an auth token
// This returns the request status, an encrypted token and an error
// If the status is approved the token will contain the auth token.
// The token is only usable by the owner of the private key and has a limited lifespan
// It should immediately be used to connect to the Hub and refresh for a new token with
// a longer lifespan. JWT decode allows to determine the expiry.
func SubmitIdProvRequest(clientID string, pubKey string, mac string, tlsClient *tlsclient.TLSClient) (
	status idprovapi.ProvisionStatus, token string, err error) {

	req := idprovapi.ProvisionRequestArgs{
		ClientID: clientID,
		PubKey:   pubKey,
		MAC:      mac,
	}
	reqData, _ := ser.Marshal(req)
	respData, err := tlsClient.Post(idprovapi.ProvisionRequestPath, reqData)
	resp := idprovapi.ProvisionRequestResp{}
	err = ser.Unmarshal(respData, &resp)
	return resp.Status, resp.Token, err
}
