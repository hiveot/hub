import { Bonjour } from 'bonjour-service';
import * as tslog from 'tslog';

// const HIVEOT_HUB_SERVICENAME = "hiveot";
const WOT_SERVICENAME = "wot";
const DefaultHiveotSsePath = "/hiveot/sse";

// hiveot endpoint identifiers
const AuthEndpoint = "login"
const WSSEndpoint = "wss"
const SSEEndpoint = "sse"


const log = new  tslog.Logger({prettyLogTimeZone:"local"})

// locateHub uses DNS-SD to search the hiveot record of the hub gateway for up to 3 seconds.
// This is based on the WoT discovery specification: https://w3c.github.io/wot-discovery/
//
// ... with a few additions:
//	instanceName is the name of the server instance. "hiveot" for the hub.
//	serviceName is the discover name. Default is 'wot'. This can be changed for testing.
//  serviceType is "Directory" or "Thing"
//	tddURL is the URL the directory TD is served at.
//	authURL http endpoint for login to obtain a token
//  wssURL URL of the websocket connection endpoint
//  sseURL URL of the sse connection endpoint
export default async function locateHub(): Promise<{
    instanceName?:string, serviceType?:string,
    tddURL?:string, authURL?:string, wssURL?: string, sseURL?: string,}> {

    return new Promise((resolve, reject) => {

        const locator = new Bonjour();
        locator.findOne({ type: WOT_SERVICENAME }, 3000, function (service: any) {
            if (!service || !service.addresses || service.addresses.length == 0 || !service.txt) {
                // reject("service not found");
                resolve({});
                return
            }
            // get the connection URLs from the TXT records
            const addr = service.addresses[0];
            const kv = service.txt;
            const authURL = kv[AuthEndpoint];       // URL of the auth service
            const wssURL = kv[WSSEndpoint]; // URL to connect the client to
            const sseURL = kv[SSEEndpoint]; // URL to connect the client to
            const scheme = kv["scheme"];      // client protocol
            const tddURL = kv["td"];          // path to wget directory service TD
            const serviceType = kv["type"];          // service type, eg Directory

            // construct the sse url
            // let sseURL = "https://"+addr+":"+service.port + DefaultHiveotSsePath


            log.info("found service: ", addr);
            resolve({
                instanceName:service.name,
                serviceType:serviceType,
                tddURL:tddURL,
                authURL: authURL,
                sseURL: sseURL,
                wssURL:wssURL,
                // scheme:scheme,
            });
        });
    });
}
