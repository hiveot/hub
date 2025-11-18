package idprovclient

import (
	"github.com/hiveot/hivehub/services/idprov/idprovapi"
	"github.com/hiveot/hivekitgo/utils/tlsclient"
	jsoniter "github.com/json-iterator/go"
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
	reqData, _ := jsoniter.Marshal(req)
	respData, _, err := tlsClient.Post(idprovapi.ProvisionRequestPath, reqData)
	resp := idprovapi.ProvisionRequestResp{}
	err = jsoniter.Unmarshal(respData, &resp)
	return resp.Status, resp.Token, err
}
