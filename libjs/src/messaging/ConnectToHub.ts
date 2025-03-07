import {IAgentConnection} from "@hivelib/messaging/IAgentConnection";
import {locateHub} from "@hivelib/messaging/locateHub";
import {HttpSSEClient} from "@hivelib/messaging/httpclient/HttpSSEClient";



// ConnectToHub helper function to connect to the Hub using existing token and key files.
// This assumes that CA cert, user keys and auth token have already been set up and
// are available in the certDir.
// The key-pair file is named {certDir}/{clientID}.key
// The token file is named {certDir}/{clientID}.token
//
// 1. If no fullURL is given then use discovery to determine the URL
// 2. Determine the core to use
// 3. Load the CA cert
// 4. Create a hub client
// 5. Connect using token and key files
//
//	fullURL is the scheme://addr:port/[path] the server is listening on
//	clientID to connect as. Also used for the key and token file names
//	certDir is the location of the CA cert and key/token files
// This throws an error if a connection cannot be made
export async function ConnectToHub(
    fullURL: string|undefined, clientID: string, caCertPem: string, disableCertCheck: boolean): Promise<IAgentConnection> {

    // 1. determine the actual address
    if (fullURL == "") {
        // return after first result
        let {sseURL} = await locateHub()
        // currently only supporting SSE on this client
        fullURL = sseURL
    }
    if (!clientID || !fullURL) {
        throw("Missing clientID or hub URL")
    }
    // 2. Determine the client protocol to use
    // TODO: support multiple client protocols
    //
    // FIXME: (node:84862) [DEP0123] DeprecationWarning: Setting the TLS ServerName
    //  to an IP address is not permitted by RFC 6066. This will be ignored in a
    //  future version.
    // Discovery provides an IP, not a servername. Also on local networks there is no
    // local DNS or servername, just an IP. What to do?
    //  https://stackoverflow.com/questions/73526773/tls-servername-as-ip-is-not-permitted
    let hc = new HttpSSEClient(fullURL, clientID, caCertPem, disableCertCheck)
    return hc
}
