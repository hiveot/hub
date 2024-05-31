import {IHubClient} from "@hivelib/hubclient/IHubClient";
import {locateHub} from "@hivelib/hubclient/locateHub";
import path from "path";
import {HttpSSEClient} from "@hivelib/hubclient/httpclient/HttpSSEClient";



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
//	fullURL is the scheme://addr:port/[wspath] the server is listening on
//	clientID to connect as. Also used for the key and token file names
//	certDir is the location of the CA cert and key/token files
// This throws an error if a connection cannot be made
export async function ConnectToHub(
    fullURL: string, clientID: string, caCertPem: string): Promise<IHubClient> {

    // 1. determine the actual address
    if (fullURL == "") {
        // return after first result
        let uc = await locateHub()
        fullURL = uc.hubURL
    }
    if (clientID == ""||fullURL == "") {
        throw("Missing clientID or hub URL")
    }
    // 2. Determine the client protocol to use
    // TODO: support multiple client protocols
    let hc = new HttpSSEClient(fullURL, clientID, caCertPem)
    return hc
}
