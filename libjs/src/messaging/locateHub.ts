
import { Bonjour } from 'bonjour-service';
import * as tslog from 'tslog';

const HIVEOT_HUB_SERVICENAME = "hiveot";
const WOT_SERVICENAME = "wot";
const DefaultHiveotSsePath = "/hiveot/sse";


const log = new  tslog.Logger({prettyLogTimeZone:"local"})

// locateHub uses DNS-SD to search the hiveot record of the hub gateway for up to 3 seconds.
// This is based on the WoT discovery specification: https://w3c.github.io/wot-discovery/
//
// ... with a few additions:
//  connectURL is the preferred connection endpoint for clients
//  authURL is the auth server endpoint (http base)
//  instance is the name of the service, eg hiveot for the hiveot directory service
//  sseURL is the expected sse endpoint. Since sse is the only supported protocol in JS
//  this is a fallback. Eventually the directory TD contains supported protocols.
//
// If found, it returns with the hub connection address:
//    https://<addr>:<port>/<ssepath>
//    wss://<addr>:<port>/<wsspath>
export async function locateHub(): Promise<{
    addr?:string, connectURL?: string, authURL?:string, hiveotSseURL?:string,
    instance?:string, td?:string,type?:string,scheme?:string }> {

    return new Promise((resolve, reject) => {

        const locator = new Bonjour();
        locator.findOne({ type: WOT_SERVICENAME }, 3000, function (service: any) {
            if (!service || !service.addresses || service.addresses.length == 0 || !service.txt) {
                // reject("service not found");
                resolve({});
                return
            }
            // from nodejs, only websockets can be used for the capnp connection
            let addr = service.addresses[0];
            let kv = service.txt;
            let authURL = kv["auth"];       // URL of the auth service
            let connectURL = kv["connect"]; // URL to connect the client to
            let scheme = kv["scheme"];      // client protocol
            let td = kv["td"];              // path to wget directory service TD
            let type = kv["type"];          // service type, eg Directory

            // construct the sse url
            let sseURL = "https://"+addr+":"+service.port + DefaultHiveotSsePath


            log.info("found service: ", addr);
            resolve({
                instance:service.name,
                addr:addr,
                connectURL: connectURL,
                authURL: authURL,
                hiveotSseURL: sseURL,
                scheme:scheme,
                td:td,
                type:type
            });
        });
    });
}
